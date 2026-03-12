package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	pb "github.com/kznLeaf/curated-store/src/productcatalogservice/genproto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
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
	ctx := context.Background()

	var (
		collectorAddr string
		collectorConn *grpc.ClientConn
	)

	xgrpc.MustMapEnv(&collectorAddr, "COLLECTOR_SERVICE_ADDR")
	xgrpc.MustConnGRPC(ctx, &collectorConn, collectorAddr)

	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(collectorConn),
	)
	if err != nil {
		log.Fatalf("Failed to create trace exporter: %v", err)
	}

	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()))

	otel.SetTracerProvider(tp)

	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("[productcatalog]starting grpc server at :%s", port)

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

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Error(fmt.Sprintf("Failed to serve gRPC server, err: %v", err))
		}
	}()

	<-stop.Done()

	srv.GracefulStop()
	log.Info("Product Catalog gRPC server stopped")
}
