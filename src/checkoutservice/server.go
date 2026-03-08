package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	pb "github.com/kznLeaf/curated-store/src/checkoutservice/genproto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	// "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const (
	listenPort  = "5050" // 默认监听端口
)

var log *logrus.Logger

// init 初始化日志记录器，配置 JSON 格式输出
func init() {
	log = logrus.New()
	log.Level = logrus.DebugLevel
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
	port := listenPort
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	// 创建服务实例并读取后端服务地址
	svc := new(checkoutService)
	mustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&svc.cartSvcAddr, "CART_SERVICE_ADDR")
	mustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	mustMapEnv(&svc.emailSvcAddr, "EMAIL_SERVICE_ADDR")
	mustMapEnv(&svc.paymentSvcAddr, "PAYMENT_SERVICE_ADDR")

	// 建立与 6 个后端服务的 gRPC 连接
	mustConnGRPC(ctx, &svc.shippingSvcConn, svc.shippingSvcAddr)
	mustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	mustConnGRPC(ctx, &svc.cartSvcConn, svc.cartSvcAddr)
	mustConnGRPC(ctx, &svc.currencySvcConn, svc.currencySvcAddr)
	mustConnGRPC(ctx, &svc.emailSvcConn, svc.emailSvcAddr)
	mustConnGRPC(ctx, &svc.paymentSvcConn, svc.paymentSvcAddr)

	log.Infof("service config: %+v", svc)

	// 监听 TCP 端口
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatal(err)
	}

	server := grpc.NewServer(grpc.StatsHandler(otelgrpc.NewServerHandler()))

	// 注册服务实现和健康检查
	pb.RegisterCheckoutServiceServer(server, svc)
	healthpb.RegisterHealthServer(server, svc)
	log.Infof("starting to listen on tcp: %q", lis.Addr().String())
	err = server.Serve(lis)
	log.Fatal(err)

}

// mustMapEnv 从环境变量读取值，如果不存在则 panic
func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		panic(fmt.Sprintf("environment variable %q not set", envKey))
	}
	*target = v
}

// mustConnGRPC 强制建立 gRPC 连接。
// 该函数会尝试连接指定地址的 gRPC 服务，如果连接成功，则将连接对象保存到 conn 指向的变量中。如果连接失败，函数会记录错误并调用 logrus.Fatalf 来终止程序运行。
func mustConnGRPC(_ context.Context, conn **grpc.ClientConn, addr string) {
	var err error

	// NewClient 立即返回，不需要设置超时。连接的建立和维护由 gRPC 库负责，库会自动处理连接的重试和恢复。
	*conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Fatalf("failed to connect to gRPC service %q: %v", addr, err)
	}
}
