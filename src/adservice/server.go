package main

import (
	"fmt"
	"net"
	"os"
	"time"

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
	log.Infof("[adservice]starting grpc server at :%s", port)
	go run(port)
	select {}
}

func run(port string) string {
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

	go srv.Serve(listener)
	return listener.Addr().String()
}
