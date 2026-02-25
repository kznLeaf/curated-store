package main

import (
	"context"
	"math/rand"

	pb "github.com/kznLeaf/curated-store/src/adservice/genproto"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

type adServiceServer struct {
	pb.UnimplementedAdServiceServer
}

// Check 属于 HealthServer 接口的一部分，获取指定服务的状态。
func (s *adServiceServer) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// List 属于 HealthServer 接口的一部分，获取所有可用服务的非原子快照。
func (s *adServiceServer) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	// 简单的空实现
	return &healthpb.HealthListResponse{}, nil
}

// Watch 属于 HealthServer 接口的一部分，每当服务状态发生变化时，服务器都会发送一条新的消息。
// TODO 实现以支持更细粒度的健康检查。
func (s *adServiceServer) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

// GetAds 用两种主要机制来选择广告。一种是基于请求中包含的 context_words 关键字来选择，
func (s *adServiceServer) GetAds(ctx context.Context, req *pb.AdRequest) (*pb.AdResponse, error) {
	log.WithField("context_words", req.ContextKeys).Info("[GetAds]received a request")

	var allAds []*pb.Ad

	// 根据上下文关键字查找匹配的广告
	for _, category := range req.ContextKeys {
		allAds = append(allAds, getAdsByCategory(category)...)
	}

	// 如果没有匹配的广告,返回随机广告
	if len(allAds) == 0 {
		allAds = getRandomAds()
	}

	return &pb.AdResponse{Ads: allAds}, nil
}

// adsMap 广告服务维护一个按产品类别组织的静态内存地图。每则广告包含：
//   - RedirectUrl：用户点击广告时重定向到的URL
//   - Text：广告文本
var adsMap = map[string][]*pb.Ad{
	"clothing": {
		{RedirectUrl: "/product/66VCHSJNUP", Text: "Tank top for sale. 20% off."},
	},
	"accessories": {
		{RedirectUrl: "/product/1YMWWN1N4O", Text: "Watch for sale. Buy one, get second kit for free"},
	},
	"footwear": {
		{RedirectUrl: "/product/L9ECAV7KIM", Text: "Loafers for sale. Buy one, get second one for free"},
	},
	"hair": {
		{RedirectUrl: "/product/2ZYFJ3GM2N", Text: "Hairdryer for sale. 50% off."},
	},
	"decor": {
		{RedirectUrl: "/product/0PUK6V6EV0", Text: "Candle holder for sale. 30% off."},
	},
	"kitchen": {
		{RedirectUrl: "/product/9SIQT8TOJO", Text: "Bamboo glass jar for sale. 10% off."},
		{RedirectUrl: "/product/6E92ZMYYFZ", Text: "Mug for sale. Buy two, get third one for free"},
	},
	"CD": {
		{RedirectUrl: "/product/MAHOYOSSSS", Text: "これはやばい"},
	},
}

func getAdsByCategory(category string) []*pb.Ad {
	if ads, ok := adsMap[category]; ok {
		return ads
	}
	return []*pb.Ad{}
}

// getAllAds 获取所有广告列表
func getAllAds() []*pb.Ad {
	var all []*pb.Ad
	for _, ads := range adsMap {
		all = append(all, ads...)
	}
	return all
}

// getRandomAds 从所有广告中随机选择一些广告来返回给用户。
func getRandomAds() []*pb.Ad {
	all := getAllAds()
	if len(all) == 0 {
		return nil
	}
	result := make([]*pb.Ad, maxAdsToServe)
	for i := 0; i < maxAdsToServe; i++ {
		result[i] = all[rand.Intn(len(all))]
	}
	return result
}
