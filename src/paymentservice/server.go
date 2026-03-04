package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	pb "github.com/kznLeaf/curated-store/src/paymentservice/genproto"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type paymentServer struct {
	pb.UnimplementedPaymentServiceServer
}

var (
	log  *logrus.Logger
	port = "50051"
)

// Check 属于 HealthServer 接口的一部分，获取指定服务的状态。
func (p *paymentServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// List 属于 HealthServer 接口的一部分，获取所有可用服务的非原子快照。
func (p *paymentServer) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	// 简单的空实现
	return &healthpb.HealthListResponse{}, nil
}

// Watch 属于 HealthServer 接口的一部分，每当服务状态发生变化时，服务器都会发送一条新的消息。
// TODO 实现以支持更细粒度的健康检查。
func (p *paymentServer) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (s *paymentServer) Charge(_ context.Context, req *pb.ChargeRequest) (*pb.ChargeResponse, error) {
	log.Debug("PaymentService Charge invoked")
	resp, err := charge(req)
	if err != nil {
		if cardErr, ok := err.(*creditCardError); ok {
			log.Warn("charge failed", "reason", cardErr.msg)
			return nil, status.Error(codes.InvalidArgument, cardErr.msg)
		}
		log.Error("unexpected charge error", "error", err)
		return nil, status.Error(codes.Internal, err.Error())
	}
	return resp, nil
}

func init() {
	log = logrus.New()
	log.Formatter = &logrus.JSONFormatter{
		FieldMap: logrus.FieldMap{
			logrus.FieldKeyTime:  "timestamp",
			logrus.FieldKeyLevel: "severity",
			logrus.FieldKeyMsg:   "message",
		},
		TimestampFormat: time.RFC3339Nano,
	}
	log.Out = os.Stdout
}

func main() {
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("[paymentservice]starting grpc server at :%s", port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	var srv *grpc.Server
	srv = grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler())) // StatsHandler 同时处理 Unary 和 Stream 请求的追踪。
	svc := &paymentServer{}                                              // 创建service具体实现的实例
	pb.RegisterPaymentServiceServer(srv, svc)                            // 将该服务的实例注册到gRPC服务器
	healthpb.RegisterHealthServer(srv, svc)                              // 注册健康检查服务

	// 设置优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		srv.GracefulStop()
	}()

	if err := srv.Serve(listener); err != nil {
		log.Fatal(err)
	}
}
