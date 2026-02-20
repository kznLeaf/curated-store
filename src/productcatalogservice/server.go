package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	pb "github.com/kznLeaf/curated-store/src/productcatalogservice/genproto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

var (
	catalogMutex *sync.Mutex
	log          *logrus.Logger
	extraLatency time.Duration

	port = "3550"

	reloadCatalog bool
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
	catalogMutex = &sync.Mutex{} // 用于目录并发控制
}

func main() {
	flag.Parse()

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("[productcatalog]starting grpc server at :%s", port)
	run(port)
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

	svc := &productCatalog{}        // 创建service具体实现的实例
	err = loadCatalog(&svc.catalog) // 加载数据到service中
	if err != nil {
		log.Fatalf("could not parse product catalog: %v", err)
	}
	pb.RegisterProductCatalogServiceServer(srv, svc) // 将该服务的实例注册到gRPC服务器
	healthpb.RegisterHealthServer(srv, svc)          // 注册健康检查服务

	go srv.Serve(listener)
	return listener.Addr().String()
}
