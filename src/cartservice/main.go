package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kznLeaf/curated-store/src/cartservice/cartstore"
	pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
	"github.com/kznLeaf/curated-store/src/common"
	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"google.golang.org/grpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

var (
	log  *logrus.Logger
	port = "7070"
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
	ctx := context.Background()

	tp := common.InitTracing(ctx, log)
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			log.Fatalf("Tracer Provider Shutdown: %v", err)
		}
	}()

	port = getEnv("PORT", "7070")
	listener, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen on port %s: %v", port, err)
	}
	log.Infof("[cartservice]starting grpc server at %s", port)

	store, err := initCartStore(ctx)
	if err != nil {
		log.Fatalf("failed to initialize cart store: %v", err)
	}

	server := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))
	pb.RegisterCartServiceServer(server, NewCartService(store))
	healthpb.RegisterHealthServer(server, NewHealthCheckService(store))

	// 设置优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-quit
		// log.Info("shutting down gRPC server...")
		server.GracefulStop()
	}()

	if err := server.Serve(listener); err != nil {
		log.Fatalf("failed to serve gRPC server: %v", err)
	}
}

func getEnv(key, defaultValue string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// initCartStore 根据环境变量选择存储后端（优先级：Redis > Spanner > AlloyDB > 内存）
func initCartStore(_ context.Context) (cartstore.ICartStore, error) {
	if redisAddr := os.Getenv("REDIS_ADDR"); redisAddr != "" {
		log.Info("[cartservice] using Redis cart store")
		return cartstore.NewRedisCartStore(redisAddr), nil
	}

	log.Info("[cartservice] using memory cart store (no external storage configured)")
	return cartstore.NewMemoryCartStore(), nil
}
