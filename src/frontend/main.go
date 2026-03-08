package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/mux"
	"github.com/kznLeaf/curated-store/infra/xgrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
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

	cartSvcAddr string
	cartSvcConn *grpc.ClientConn

	checkoutSvcAddr string
	checkoutSvcConn *grpc.ClientConn
}

const (
	port         = "8080"
	cookieMaxAge = 60 * 60 * 48 // 48小时

	// 会话管理: 从 Cookie 中提取 sessionID 作为购物车标识符。
	cookiePrefix    = "shop_"
	cookieSessionID = cookiePrefix + "session-id"
	cookieCurrency  = cookiePrefix + "currency"
)

var (
	baseUrl = ""
)

func main() {
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
	ctx := context.Background()

	svc := new(frontendServer)

	srvPort := port
	// PORT 环境变量定义在k8s清单文件中。
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}
	addr := os.Getenv("LISTEN_ADDR")
	log.Infof("frontend service listening on port %s, addr: %s\n", srvPort, addr)

	type serviceBootstrap struct {
		name   string
		envKey string
		addr   *string
		conn   **grpc.ClientConn
	}

	services := []serviceBootstrap{
		{name: "productcatalogservice", envKey: "PRODUCT_CATALOG_SERVICE_ADDR", addr: &svc.productCatalogSvcAddr, conn: &svc.productCatalogSvcConn},
		{name: "currencyservice", envKey: "CURRENCY_SERVICE_ADDR", addr: &svc.currencySvcAddr, conn: &svc.currencySvcConn},
		{name: "shippingservice", envKey: "SHIPPING_SERVICE_ADDR", addr: &svc.shippingSvcAddr, conn: &svc.shippingSvcConn},
		{name: "adservice", envKey: "AD_SERVICE_ADDR", addr: &svc.adSvcAddr, conn: &svc.adSvcConn},
		{name: "recommendationservice", envKey: "RECOMMENDATION_SERVICE_ADDR", addr: &svc.recommendationSvcAddr, conn: &svc.recommendationSvcConn},
		{name: "cartservice", envKey: "CART_SERVICE_ADDR", addr: &svc.cartSvcAddr, conn: &svc.cartSvcConn},
		{name: "checkoutservice", envKey: "CHECKOUT_SERVICE_ADDR", addr: &svc.checkoutSvcAddr, conn: &svc.checkoutSvcConn},
	}

	// 读取服务地址
	for _, s := range services {
		xgrpc.Must(xgrpc.MustMapEnv(s.addr, s.envKey), log, "failed to read env %s", s.envKey)
	}

	// 利用上一步读取的服务地址，建立 gRPC 连接
	for _, s := range services {
		xgrpc.Must(xgrpc.MustConnGRPC(ctx, s.conn, *s.addr), log, "failed to connect to %s (%s)", s.name, *s.addr)
	}

	// baseUrl := os.Getenv("BASE_URL") // 该环境变量位于 kustomize/components/custom-base-url/kustomization.yaml

	// 设置路由规则和处理函数
	r := mux.NewRouter()
	r.HandleFunc(baseUrl+"/", svc.homeHandler).Methods(http.MethodGet, http.MethodHead)                               // 首页 get
	r.HandleFunc(baseUrl+"/product/{id}", svc.productHandler).Methods(http.MethodGet, http.MethodHead)                // 产品详情页 get
	r.HandleFunc(baseUrl+"/_healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "[frontend]ok") }) // 健康检查
	r.HandleFunc(baseUrl+"/setCurrency", svc.setCurrencyHandler).Methods(http.MethodPost)                             // 用户手动切换货币
	r.HandleFunc(baseUrl+"/cart", svc.viewCartHandler).Methods(http.MethodGet, http.MethodHead)
	r.PathPrefix(baseUrl + "/static/").Handler(http.StripPrefix(baseUrl+"/static/", http.FileServer(http.Dir("./static/")))) // 加载static/目录下的静态资源
	r.HandleFunc(baseUrl+"/cart", svc.addToCartHandler).Methods(http.MethodPost)                                             // 添加商品到购物车
	r.HandleFunc(baseUrl+"/cart", svc.viewCartHandler).Methods(http.MethodGet, http.MethodHead)
	r.HandleFunc(baseUrl+"/cart/checkout", svc.placeOrderHandler).Methods(http.MethodPost) // 结账 post
	r.HandleFunc(baseUrl+"/cart/empty", svc.emptyCartHandler).Methods(http.MethodPost)     // 清空购物车 post
	r.HandleFunc(baseUrl+"/assistant", svc.assistantHandler).Methods(http.MethodGet)
	var handler http.Handler = r                   // r 实现了 http.Handler 接口，属于业务Handler
	handler = &logHandler{log: log, next: handler} // Router实现了 http.Handler 接口
	handler = ensureSessionID(handler)             // 注入 sessionID 管理中间件
	log.Infof("starting server on %s:%s", addr, srvPort)

	// 启动 HTTP 服务器。传入handler，这样每次收到HTTP请求自动调用中间件链和路由规则
	log.Fatal(http.ListenAndServe(addr+":"+srvPort, handler))
}
