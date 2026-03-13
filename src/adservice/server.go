package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	pb "github.com/kznLeaf/curated-store/src/adservice/genproto"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc"
)

var (
	log  *logrus.Logger
	port = "9555"
)

const (
	maxAdsToServe = 2
)

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
	ctx := context.Background()
	tp := xgrpc.InitTracing(ctx, log)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	log.Infof("[adservice]starting grpc server at :%s", port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	var srv *grpc.Server
	srv = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler())) // StatsHandler 同时处理 Unary 和 Stream 请求的追踪。
	svc := &adServiceServer{}               // 创建service具体实现的实例
	pb.RegisterAdServiceServer(srv, svc)    // 将该服务的实例注册到gRPC服务器
	healthpb.RegisterHealthServer(srv, svc) // 注册健康检查服务

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-stop.Done()

	srv.GracefulStop()
	log.Info("adservice gRPC server stopped")
}
