package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.39.0"

	"github.com/kznLeaf/curated-store/infra/xgrpc"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
)

// frontServer 用于管理前端与后端的交互
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

	collectorAddr string
	collectorConn *grpc.ClientConn
}

const (
	port         = "8080"
	cookieMaxAge = 60 * 60 * 48 // 48小时

	// 会话管理: 从 Cookie 中提取 sessionID 作为购物车标识符。
	cookiePrefix    = "shop_"
	cookieSessionID = cookiePrefix + "session-id"
	cookieCurrency  = cookiePrefix + "currency" // cookie 中存储用户选择的货币
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

	tp, err := initTracing(log, ctx, svc)
	if err != nil {
		log.Warnf("warn: Failed to initialize tracing: %v", err)
	} else {
		defer func() {
			if err := tp.Shutdown(ctx); err != nil {
				log.Fatalf("Tracer Provider Shutdown: %v", err)
			}
		}()
	}

	srvPort := port
	// PORT 环境变量定义在k8s清单文件中。
	if os.Getenv("PORT") != "" {
		srvPort = os.Getenv("PORT")
	}
	addr := os.Getenv("LISTEN_ADDR")
	log.Infof("frontend service listening on port %s, addr: %s\n", srvPort, addr)

	xgrpc.MustMapEnv(&svc.productCatalogSvcAddr, "PRODUCT_CATALOG_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.currencySvcAddr, "CURRENCY_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.shippingSvcAddr, "SHIPPING_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.adSvcAddr, "AD_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.recommendationSvcAddr, "RECOMMENDATION_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.cartSvcAddr, "CART_SERVICE_ADDR")
	xgrpc.MustMapEnv(&svc.checkoutSvcAddr, "CHECKOUT_SERVICE_ADDR")
	// 利用上一步读取的服务地址，建立gRPC连接
	xgrpc.MustConnGRPC(ctx, &svc.productCatalogSvcConn, svc.productCatalogSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.currencySvcConn, svc.currencySvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.shippingSvcConn, svc.shippingSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.adSvcConn, svc.adSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.recommendationSvcConn, svc.recommendationSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.cartSvcConn, svc.cartSvcAddr)
	xgrpc.MustConnGRPC(ctx, &svc.checkoutSvcConn, svc.checkoutSvcAddr)

	router := http.NewServeMux()

	router.HandleFunc("GET /", svc.homeHandler)
	router.HandleFunc("GET /product/{id}", svc.productHandler)
	router.HandleFunc("GET /_healthz", func(w http.ResponseWriter, _ *http.Request) { fmt.Fprint(w, "[frontend]ok") }) // 健康检查
	router.HandleFunc("POST /setCurrency", svc.setCurrencyHandler)
	router.HandleFunc("GET /cart", svc.viewCartHandler)
	router.Handle("GET /static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	router.HandleFunc("POST /cart", svc.addToCartHandler)
	router.HandleFunc("POST /cart/checkout", svc.placeOrderHandler)
	router.HandleFunc("POST /cart/empty", svc.emptyCartHandler)
	router.HandleFunc("GET /assistant", svc.assistantHandler)
	router.HandleFunc("GET /login", svc.loginHandler)

	var handler http.Handler = router
	handler = &logHandler{log: log, next: handler}
	handler = ensureSessionID(handler)
	handler = authorize(handler)
	handler = otelhttp.NewHandler(handler, "frontend")

	log.Infof("starting server on %s:%s", addr, srvPort)
	srv := http.Server{
		Addr:    addr + ":" + srvPort,
		Handler: handler,
	}

	idleConnectClose := make(chan struct{})
	go func() {
		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		if err := srv.Shutdown(ctx); err != nil {
			log.Fatalf("Server shutdown failed; %v", err)
		}
		close(idleConnectClose)
	}()

	if err := srv.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("HTTP server listenAndServe: %v", err)
	}

	<-idleConnectClose
}

// initTracing 初始化 OpenTelemetry 追踪
// reference: https://github.com/open-telemetry/opentelemetry-go-contrib/blob/main/instrumentation/net/http/otelhttp/example/server/server.go
func initTracing(log logrus.FieldLogger, ctx context.Context, svc *frontendServer) (*sdktrace.TracerProvider, error) {
	xgrpc.MustMapEnv(&svc.collectorAddr, "COLLECTOR_SERVICE_ADDR")
	xgrpc.MustConnGRPC(ctx, &svc.collectorConn, svc.collectorAddr)
	exporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(svc.collectorConn))
	if err != nil {
		log.Warnf("warn: Failed to create trace exporter: %v", err)
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("frontend-service"),
		)),
	)
	otel.SetTracerProvider(tp)

	// https://opentelemetry.io/docs/specs/otel/context/api-propagators/
	otel.SetTextMapPropagator(
		propagation.NewCompositeTextMapPropagator(
			propagation.TraceContext{}, propagation.Baggage{}))

	return tp, nil
}
