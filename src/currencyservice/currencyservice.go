package main

// 这里实现 gRPC API

import (
	"context"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"math"

	pb "github.com/kznLeaf/curated-store/src/currencyservice/genproto"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
)

// currencyService implements the gRPC CurrencyService.
type currencyService struct {
	pb.UnimplementedCurrencyServiceServer
	rates map[string]float64
}

// GetSupportedCurrencies rpc 返回支持的货币代码列表。
func (c *currencyService) GetSupportedCurrencies(ctx context.Context, req *pb.Empty) (*pb.GetSupportedCurrenciesResponse, error) {
	log.Info("[CurrencyService] GetSupportedCurrencies invoked")
	codes := make([]string, 0, len(c.rates))
	for code := range c.rates {
		codes = append(codes, code)
	}
	return &pb.GetSupportedCurrenciesResponse{CurrencyCodes: codes}, nil
}

// Convert rpc 将金额从一种货币转换为另一种货币。
func (s *currencyService) Convert(ctx context.Context, req *pb.CurrencyConversionRequest) (*pb.Money, error) {
	log.Info("[CurrencyService] Convert invoked", "from", req.GetFrom(), "to", req.GetToCode())
	from := req.GetFrom()
	if from == nil {
		return nil, status.Error(codes.InvalidArgument, "from money is required")
	}
	// from = &pb.Money{
	// 	CurrencyCode: "JPY",
	// 	Units:        from.GetUnits(),
	// 	Nanos:        from.GetNanos(),
	// }

	fromRate, ok := s.rates[from.GetCurrencyCode()] // 获取源货币的汇率
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", from.GetCurrencyCode())
	}
	toRate, ok := s.rates[req.GetToCode()] // 获取目标货币的汇率
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported currency: %s", req.GetToCode())
	}

	// Step 1: Convert from source currency → USD
	UsdUnits := float64(from.GetUnits()) / fromRate
	UsdNanos := float64(from.GetNanos()) / fromRate
	Usds := carry(UsdUnits, math.Round(UsdNanos))

	// Step 2: Convert USD → target currency
	resultUnits := Usds.units * toRate
	resultNanos := Usds.nanos * toRate
	result := carry(resultUnits, resultNanos)
	// log.Infof("[currencyservice]货币转换成功")

	money := &pb.Money{
		CurrencyCode: req.GetToCode(),
		Units:        int64(math.Floor(result.units)),
		Nanos:        int32(math.Floor(result.nanos)),
	}
	log.Infof("[currencyservice]successfully converted %d.%09d %s to %d.%09d %s",
		from.GetUnits(), from.GetNanos(), from.GetCurrencyCode(),
		money.GetUnits(), money.GetNanos(), money.GetCurrencyCode())

	return money, nil
}

// intermediate holds floating-point units and nanos during conversion.
type intermediate struct {
	units float64
	nanos float64
}

// carry handles decimal/fractional overflow between units and nanos,
// mirroring the JS _carry() function.
//
//	func carry(units, nanos float64) intermediate {
//		const fractionSize = 1e9 // 1 billion nanos in 1 unit
//		nanos += math.Mod(units, 1) * fractionSize // 将单位的小数部分转换为纳秒并加到nanos
//		units = math.Floor(units) + math.Floor(nanos/fractionSize) // 将nanos中超过fractionSize的部分转换为unit并加到units
//		nanos = math.Mod(nanos, fractionSize)
//		return intermediate{units: units, nanos: nanos}
//	}
func carry(units float64, nanos float64) intermediate {
	const fractionSize = 1e9
	units += math.Floor(nanos / fractionSize)
	nanos = math.Mod(nanos, fractionSize)
	return intermediate{units: units, nanos: nanos}
}

// Check 属于 HealthServer 接口的一部分，获取指定服务的状态。
func (s *currencyService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// List 属于 HealthServer 接口的一部分，获取所有可用服务的非原子快照。
func (s *currencyService) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	// 简单的空实现
	return &healthpb.HealthListResponse{}, nil
}

// Watch 属于 HealthServer 接口的一部分，每当服务状态发生变化时，服务器都会发送一条新的消息。
func (s *currencyService) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}
