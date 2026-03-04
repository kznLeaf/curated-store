package main

import (
	"fmt"
	"net"
	"os"

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

	go run(port)
	log.Infof("[shipping]starting grpc server at :%s", port)
	select {}
}

// run 启动 gRPC 服务器
// 并注册服务实现和健康检查服务(待定)
func run(port string) string {
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

	go srv.Serve(listener)
	return listener.Addr().String()
}
