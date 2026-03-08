package xgrpc

import (
	"context"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// MustMapEnv 强制将环境变量映射到目标字符串指针
// 这里的环境变量由k8s自动为Service创建，并且短横线被替换为下划线。
// 例如，my-nginx服务会自动在 node 中设置环境变量 MY_NGINX_SERVICE_HOST 和 MY_NGINX_SERVICE_PORT。
// 因此通过读取环境变量，就可以实现服务发现。这也是k8s的两种服务发现方式之一。
func MustMapEnv(target *string, envKey string) error {
	v := os.Getenv(envKey)
	if v == "" {
		return fmt.Errorf("environment variable %q is not set", envKey)
	}
	*target = v
	return nil
}

// MustConnGRPC 强制建立 gRPC 连接。
// 该函数会尝试连接指定地址的 gRPC 服务，如果连接成功，则将连接对象保存到 conn 指向的变量中。如果连接失败，函数会记录错误并调用 logrus.Fatalf 来终止程序运行。
func MustConnGRPC(ctx context.Context, conn **grpc.ClientConn, addr string) error {
	var err error

	// NewClient 立即返回，不需要设置超时。连接的建立和维护由 gRPC 库负责，库会自动处理连接的重试和恢复。
	*conn, err = grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	return nil
}

func Must(err error, log *logrus.Logger, msg string, fields ...any) {
	if err != nil {
		log.Fatalf("%s: %v", fmt.Sprintf(msg, fields...), err)
	}
}
