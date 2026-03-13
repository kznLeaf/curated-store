package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	pb "github.com/kznLeaf/curated-store/src/shippingservice/genproto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

func main() {
	port := defaultPort
	if value, ok := os.LookupEnv("PORT"); ok {
		port = value
	}

	ctx := context.Background()
	tp := xgrpc.InitTracing(ctx, log)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	var srv *grpc.Server
	srv = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler())) // StatsHandler 同时处理 Unary 和 Stream 请求的追踪。

	svc := &server{}
	pb.RegisterShippingServiceServer(srv, svc) // 将该服务的实例注册到gRPC服务器
	healthpb.RegisterHealthServer(srv, svc)    // 注册健康检查服务

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-stop.Done() // 等待中断信号

	srv.GracefulStop() // 优雅地停止服务器，允许正在处理的请求完成
	log.Infof("[shipping]starting grpc server at :%s", port)
}
