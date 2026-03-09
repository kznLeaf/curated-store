package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	pb "github.com/kznLeaf/curated-store/src/checkoutservice/genproto"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	// "go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

const (
	listenPort = "5050" // 默认监听端口
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
	// 读取服务地址
	xgrpc.MustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.cartSvcAddr, "CART_SERVICE_ADDR")
	// 利用上一步读取的服务地址，建立gRPC连接
	xgrpc.MustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.currencySvcConn, svc.currencySvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.shippingSvcConn, svc.shippingSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.cartSvcConn, svc.cartSvcAddr)

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
