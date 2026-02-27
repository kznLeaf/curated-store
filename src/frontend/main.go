package main

import (
	"context"
	"fmt"
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

	shippingSvcAddr string
	shippingSvcConn *grpc.ClientConn

	adSvcAddr string
	adSvcConn *grpc.ClientConn

	recommendationSvcAddr string
	recommendationSvcConn *grpc.ClientConn
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

var (
	log     *logrus.Logger
	baseUrl = ""
)

func init() {
	// 设置日志
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

	svc := new(frontendServer)

	srvPort := port
	// PORT 环境变量定义在k8s清单文件中。
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}
	addr := os.Getenv("LISTEN_ADDR")
	log.Infof("前端服务正在监听端口 %s, addr: %s\n", srvPort, addr)

	// 读取服务地址
	mustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	mustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	mustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	mustMapEnv(&svc.adSvcAddr, "AD_SERVICE_ADDR")
	mustMapEnv(&svc.recommendationSvcAddr, "RECOMMENDATION_SERVICE_ADDR")

	// 利用上一步读取的服务地址，建立gRPC连接
	mustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	mustConnGRPC(ctx, &svc.currencySvcConn, svc.currencySvcAddr)
	mustConnGRPC(ctx, &svc.shippingSvcConn, svc.shippingSvcAddr)
	mustConnGRPC(ctx, &svc.adSvcConn, svc.adSvcAddr)
	mustConnGRPC(ctx, &svc.recommendationSvcConn, svc.recommendationSvcAddr)

	// baseUrl := os.Getenv("BASE_URL") // 该环境变量位于 kustomize/components/custom-base-url/kustomization.yaml

	// 设置路由规则和处理函数
	r := mux.NewRouter()
	r.HandleFunc(baseUrl+"/", svc.homeHandler).Methods(http.MethodGet, http.MethodHead)                               // 首页 get
	r.HandleFunc(baseUrl+"/product/{id}", svc.productHandler).Methods(http.MethodGet, http.MethodHead)                // 产品详情页 get
	r.HandleFunc(baseUrl+"/_healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "[frontend]ok") }) // 健康检查
	r.HandleFunc(baseUrl+"/setCurrency", svc.setCurrencyHandler).Methods(http.MethodPost)                             // 用户手动切换货币
	r.HandleFunc(baseUrl+"/cart", svc.viewCartHandler).Methods(http.MethodGet, http.MethodHead)
	r.PathPrefix(baseUrl + "/static/").Handler(http.StripPrefix(baseUrl + "/static/", http.FileServer(http.Dir("./static/")))) // 加载static/目录下的静态资源


	var handler http.Handler = r // Router实现了 http.Handler 接口

	log.Infof("starting server on %s:%s", addr, srvPort)

	// 启动 HTTP 服务器。传入handler，这样每次收到HTTP请求自动调用中间件链和路由规则
	log.Fatal(http.ListenAndServe(addr+":"+srvPort, handler))
}

// mustMapEnv 强制将环境变量映射到目标字符串指针
// 这里的环境变量由k8s自动为Service创建，并且短横线被替换为下划线。
// 例如，my-nginx服务会自动在 node 中设置环境变量 MY_NGINX_SERVICE_HOST 和 MY_NGINX_SERVICE_PORT。
// 因此通过读取环境变量，就可以实现服务发现。这也是k8s的两种服务发现方式之一。
func mustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		logrus.Fatalf("环境变量 %q 未设置", envKey)
	}
	*target = v
}

// mustConnGRPC 强制建立 gRPC 连接。
// 该函数会尝试连接指定地址的 gRPC 服务，如果连接成功，则将连接对象保存到 conn 指向的变量中。如果连接失败，函数会记录错误并调用 logrus.Fatalf 来终止程序运行。
func mustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error

	// NewClient 立即返回，不需要设置超时。连接的建立和维护由 gRPC 库负责，库会自动处理连接的重试和恢复。
	*conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logrus.Fatalf("无法连接到 gRPC 服务 %q: %v", addr, err)
	}
}
