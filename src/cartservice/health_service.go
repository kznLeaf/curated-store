package main

import (
	"context"

	"github.com/kznLeaf/curated-store/src/cartservice/cartstore"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/status"
)

// HealthCheckService 实现 gRPC Health Check 协议（grpc.health.v1）
type HealthCheckService struct {
	healthpb.UnimplementedHealthServer
	store cartstore.ICartStore
}

func NewHealthCheckService(store cartstore.ICartStore) *HealthCheckService {
	return &HealthCheckService{store: store}
}

func (s *HealthCheckService) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {

	servingStatus := healthpb.HealthCheckResponse_NOT_SERVING

	if s.store.Ping(ctx) {
		servingStatus = healthpb.HealthCheckResponse_SERVING
	}
	return &healthpb.HealthCheckResponse{Status: servingStatus}, nil
}

func (s *HealthCheckService) Watch(_ *healthpb.HealthCheckRequest, stream healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

func (s *HealthCheckService) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	return &healthpb.HealthListResponse{}, nil
}
