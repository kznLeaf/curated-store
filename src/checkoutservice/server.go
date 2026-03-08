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

	type serviceBootstrap struct {
		name   string
		envKey string
		addr   *string
		conn   **grpc.ClientConn
	}

	services := []serviceBootstrap{
		{name: "shippingservice", envKey: "SHIPPING_SERVICE_ADDR", addr: &svc.shippingSvcAddr, conn: &svc.shippingSvcConn},
		{name: "productcatalogservice", envKey: "PRODUCT_CATALOG_SERVICE_ADDR", addr: &svc.productCatalogSvcAddr, conn: &svc.productCatalogSvcConn},
		{name: "cartservice", envKey: "CART_SERVICE_ADDR", addr: &svc.cartSvcAddr, conn: &svc.cartSvcConn},
		{name: "currencyservice", envKey: "CURRENCY_SERVICE_ADDR", addr: &svc.currencySvcAddr, conn: &svc.currencySvcConn},
		{name: "emailservice", envKey: "EMAIL_SERVICE_ADDR", addr: &svc.emailSvcAddr, conn: &svc.emailSvcConn},
		{name: "paymentservice", envKey: "PAYMENT_SERVICE_ADDR", addr: &svc.paymentSvcAddr, conn: &svc.paymentSvcConn},
	}

	for _, s := range services {
		xgrpc.Must(xgrpc.MustMapEnv(s.addr, s.envKey), log, "failed to read env %s", s.envKey)
	}

	for _, s := range services {
		xgrpc.Must(xgrpc.MustConnGRPC(ctx, s.conn, *s.addr), log, "failed to connect to %s (%s)", s.name, *s.addr)
	}

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
