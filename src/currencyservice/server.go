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
	pb "github.com/kznLeaf/curated-store/src/currencyservice/genproto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
)

var (
	currencyMutex *sync.Mutex
	log           *logrus.Logger
	port          = "7000"
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
	currencyMutex = &sync.Mutex{} // 用于刷新汇率表
}

func main() {
	ctx := context.Background()
	tp := xgrpc.InitTracing(ctx, log)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Infof("[currencyservice]starting server at :%s", port)

	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}
	srv := grpc.NewServer(
		grpc.StatsHandler(otelgrpc.NewServerHandler())) // StatsHandler 同时处理 Unary 和 Stream 请求的追踪。

	svc := &currencyService{rates: make(map[string]float64)}
	svc.rates, err = loadCurrencyData() // 创建service具体实现的实例
	if err != nil {
		log.Fatalf("could not load currency data: %v", err)
	}

	pb.RegisterCurrencyServiceServer(srv, svc) // 注册服务实现到 gRPC 服务器
	healthpb.RegisterHealthServer(srv, svc)    // 注册健康检查服务

	stop, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGKILL)
	defer cancel()

	go func() {
		if err := srv.Serve(listener); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	<-stop.Done()
	srv.GracefulStop()
	log.Info("shutting down server...")
}
