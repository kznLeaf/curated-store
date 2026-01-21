package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// frontServer 用于管理前端与后段的交互
//
// - <ServiceName>SvcAddr：服务地址
// - <ServiceName>SvcConn: gRPC连接
type frontendServer struct {
	productCatalogSvcAddr string
	productCatalogSvcConn *grpc.ClientConn

	currencySvcAddr string
	currencySvcConn *grpc.ClientConn

	cartSvcAddr string
	cartSvcConn *grpc.ClientConn

	recommendationSvcAddr string
	recommendationSvcConn *grpc.ClientConn

	checkoutSvcAddr string
	checkoutSvcConn *grpc.ClientConn

	shippingSvcAddr string
	shippingSvcConn *grpc.ClientConn

	adSvcAddr string
	adSvcConn *grpc.ClientConn

	collectorAddr string // 用于分布式追踪的收集器 的 地址 和 gRPC连接
	collectorConn *grpc.ClientConn

	shoppingAssistantSvcAddr string
}

const (
	port            = "8080"
	defaultCurrency = "USD"
	cookieMaxAge    = 60 * 60 * 48 // 48小时

	// 会话管理: 从 Cookie 中提取 sessionID 作为购物车标识符。
	cookiePrefix    = "shop_"
	cookieSessionID = cookiePrefix + "session-id"
	cookieCurrency  = cookiePrefix + "currency"
)

func main() {
	ctx := context.Background()

	// 设置日志
	log := logrus.New()
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

	svc := new(frontendServer)

	svcPort := port
	if os.Getenv("PORT") != "" {
		svcPort = os.Getenv("PORT")
	}
	_ = svcPort
	// addr := os.Getenv("LISTEN_ADDR")

	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")

	mustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)

	baseUrl := os.Getenv("BASE_URL") // 该环境变量位于 kustomize/components/custom-base-url/kustomization.yaml
	baseUrl = ""
	// 设置路由规则和处理函数
	r := mux.NewRouter()
	r.HandleFunc(baseUrl + "/product/{id}", svc.productHandler).Methods(http.MethodGet, http.MethodHead) // 产品详情页 get
}

// mustMapEnv 强制将环境变量映射到目标字符串指针
func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		logrus.Fatalf("环境变量 %q 未设置", envKey)
	}
	*target = v
}

func mustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	*conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Fatalf("无法连接到 gRPC 服务 %q: %v", addr, err)
	}
}
