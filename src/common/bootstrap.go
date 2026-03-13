package common

import (
	"context"
	"os"

	"github.com/sirupsen/logrus"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MustMapEnv 强制将环境变量映射到目标字符串指针
// 这里的环境变量由k8s自动为Service创建，并且短横线被替换为下划线。
// 例如，my-nginx服务会自动在 node 中设置环境变量 MY_NGINX_SERVICE_HOST 和 MY_NGINX_SERVICE_PORT。
// 因此通过读取环境变量，就可以实现服务发现。这也是k8s的两种服务发现方式之一。
func MustMapEnv(target *string, envKey string) {
	v := os.Getenv(envKey)
	if v == "" {
		logrus.Fatalf("environment variable %q is not set", envKey)
	}
	*target = v
}

// MustConnGRPC 强制建立 gRPC 连接。
// 该函数会尝试连接指定地址的 gRPC 服务，如果连接成功，则将连接对象保存到 conn 指向的变量中。如果连接失败，函数会记录错误并调用 logrus.Fatalf 来终止程序运行。
func MustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) {
	var err error

	// NewClient 立即返回，不需要设置超时。连接的建立和维护由 gRPC 库负责，库会自动处理连接的重试和恢复。
	*conn, err = grpc.NewClient(addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithStatsHandler(otelgrpc.NewClientHandler())) // gRPC client埋点，让该服务对下游的调用产生出站client span，保证上下文能传播到下游
	if err != nil {
		logrus.Fatalf("failed to connect to gRPC service %q: %v", addr, err)
	}
}

// InitTracing 初始化 OpenTelementry 链路追踪。这个函数会创建一个 OTLP gRPC 导出器，将追踪数据发送到指定的 collector 地址 COLLECTOR_SERVICE_ADDR，
// 并设置全局的 TracerProvider 和 Propagator。调用以后需要调用返回的 tp 的 Shutdown 方法来确保追踪数据被正确发送。
func InitTracing(ctx context.Context, log *logrus.Logger) *sdktrace.TracerProvider {

	var (
		collectorAddr string
		collectorConn *grpc.ClientConn
	)

	MustMapEnv(&collectorAddr, "COLLECTOR_SERVICE_ADDR")
	MustConnGRPC(ctx, &collectorConn, collectorAddr)

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

	return tp
}
