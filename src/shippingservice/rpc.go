package main

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"time"

	pb "github.com/kznLeaf/curated-store/src/shippingservice/genproto"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type server struct {
	pb.UnimplementedShippingServiceServer
}

const (
	defaultPort = "50051"
)

var log *logrus.Logger

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

func (s *server) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

func (s *server) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	// 简单的空实现
	return &healthpb.HealthListResponse{}, nil
}

func (s *server) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

// GetQuote 产生用美元表示的运费报价，该服务定义在 proto/demo.proto 文件的 ShippingService 服务中，需要手动实现接口。
// rpc GetQuote(GetQuoteRequest) returns (GetQuoteResponse) {}
//
// 参数：
//   - in: 请求参数
//
// 返回：
//   - *pb.GetQuoteResponse: 响应结果
//   - error: 错误信息
func (s *server) GetQuote(ctx context.Context, in *pb.GetQuoteRequest) (*pb.GetQuoteResponse, error) {
	log.Info("[GetQuote] 收到请求")
	defer log.Info("[GetQuote] 请求处理完毕")

	for _, item := range in.Items {
		log.Debugf("[GetQuote] 计算运费 商品ID: %s, 数量: %d", item.GetProductId(), item.GetQuantity())
	}

	return &pb.GetQuoteResponse{ // TODO 目前运费服务返回的是固定的运费，后续再完善这些逻辑
		CostUsd: &pb.Money{
			CurrencyCode: "USD",
			Units:        int64(17),
			Nanos:        int32(1 * 10000000)},
	}, nil
}

// ShipOrder mocks that the requested items will be shipped.
// It supplies a tracking ID for notional lookup of shipment delivery status.
// 模拟发货流程，先打印日志，然后拼接完整的地址，再把地址作为盐生成物流追踪ID，并封装到响应结果中返回。只被结账服务调用。
func (s *server) ShipOrder(ctx context.Context, in *pb.ShipOrderRequest) (*pb.ShipOrderResponse, error) {
	log.Info("[ShipOrder] 收到请求")
	defer log.Info("[ShipOrder] 请求处理完毕")
	// 创建一个追踪ID
	baseAddress := fmt.Sprintf("%s, %s, %s", in.Address.StreetAddress, in.Address.City, in.Address.State)
	trackId := createTrackingId(baseAddress)
	return &pb.ShipOrderResponse{TrackingId: trackId}, nil
}

func createTrackingId(salt string) string {
	return fmt.Sprintf("%c%c-%d%s-%d%s",
		getRandomLetterCode(),
		getRandomLetterCode(),
		len(salt),
		getRandomNumber(3),
		len(salt)/2,
		getRandomNumber(7),
	)
}

// getRandomLetterCode generates a code point value for a capital letter.
// 生成 A-Z 之间的随机字母代码 (ASCII 65='A', 90='Z')
func getRandomLetterCode() uint32 {
	return 65 + uint32(rand.Intn(26)) // 修复: 应该是 0-25 共26个字母
}

// getRandomNumber generates a string representation of a number with the requested number of digits.
// 生成指定位数的随机数字字符串
func getRandomNumber(digits int) string {
	str := ""
	for i := 0; i < digits; i++ {
		str = fmt.Sprintf("%s%d", str, rand.Intn(10))
	}
	return str
}
