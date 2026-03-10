package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"sync"
	"time"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	pb "github.com/kznLeaf/curated-store/src/productcatalogservice/genproto"
	"go.opentelemetry.io/otel"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"google.golang.org/grpc"
)

var (
	catalogMutex *sync.Mutex
	log          *logrus.Logger

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
	// if os.Getenv("ENABLE_TRACING") == "1" { 始终启用追踪功能，便于调试和性能分析
	tp := initTracing()
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

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
	// gRPC auto-instrumentation
	srv = grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler()))

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

func initTracing() *sdktrace.TracerProvider {
	var (
		collectorAddr string
		collectorConn *grpc.ClientConn
	)

	ctx := context.Background()

	xgrpc.MustMapEnv(&collectorAddr, "COLLECTOR_SERVICE_ADDR")
	xgrpc.MustConnGRPC(ctx, &collectorConn, collectorAddr)

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(collectorConn))
	if err != nil {
		log.Warnf("warn: Failed to create trace exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()))
	otel.SetTracerProvider(tp)
	return tp
}
