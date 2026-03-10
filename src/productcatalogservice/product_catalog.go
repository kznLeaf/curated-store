package main

import (
	"context"
	"fmt"
	"strings"

	pb "github.com/kznLeaf/curated-store/src/productcatalogservice/genproto"
	"go.opentelemetry.io/otel/attribute"
	otelcodes "go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"

	"google.golang.org/grpc/status"
)

type productCatalog struct {
	pb.UnimplementedProductCatalogServiceServer
	catalog pb.ListProductsResponse
	maps    map[string]*pb.Product
}

// Check 属于 HealthServer 接口的一部分，获取指定服务的状态。
func (p *productCatalog) Check(ctx context.Context, req *healthpb.HealthCheckRequest) (*healthpb.HealthCheckResponse, error) {
	return &healthpb.HealthCheckResponse{Status: healthpb.HealthCheckResponse_SERVING}, nil
}

// List 属于 HealthServer 接口的一部分，获取所有可用服务的非原子快照。
func (p *productCatalog) List(ctx context.Context, req *healthpb.HealthListRequest) (*healthpb.HealthListResponse, error) {
	// 简单的空实现
	return &healthpb.HealthListResponse{}, nil
}

// Watch 属于 HealthServer 接口的一部分，每当服务状态发生变化时，服务器都会发送一条新的消息。
// TODO 实现以支持更细粒度的健康检查。
func (p *productCatalog) Watch(req *healthpb.HealthCheckRequest, ws healthpb.Health_WatchServer) error {
	return status.Errorf(codes.Unimplemented, "health check via Watch not implemented")
}

// parseCatalog 解析产品目录。被 ListProducts 和 GetProduct 调用，确保目录已加载并构建了产品 ID 到产品的映射。
func (p *productCatalog) parseCatalog() []*pb.Product {
	if p.maps == nil || reloadCatalog || len(p.catalog.Products) == 0 {
		// 如果目录为空或需要重载，先加载
		if len(p.catalog.Products) == 0 || reloadCatalog {
			if err := loadCatalog(&p.catalog); err != nil {
				return []*pb.Product{}
			}
		}
		// 构建或重建产品 ID 到产品的映射
		p.maps = make(map[string]*pb.Product)
		for _, product := range p.catalog.Products {
			p.maps[product.Id] = product
		}
	}
	return p.catalog.Products
}

// rpc ListProducts(Empty) returns (ListProductsResponse) {}
func (p *productCatalog) ListProducts(ctx context.Context, _ *pb.Empty) (*pb.ListProductsResponse, error) {
	span := trace.SpanFromContext(ctx)
	products := p.parseCatalog()
	span.SetAttributes(
		attribute.Int("app.products.count", len(products)),
	)
	return &pb.ListProductsResponse{Products: products}, nil
}

// rpc GetProduct(GetProductRequest) returns (Product) {} 查询单个产品的信息
func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	span := trace.SpanFromContext(ctx)
	span.SetAttributes(
		attribute.String("app.product.id", req.Id),
	)
	if p.maps == nil {
		p.parseCatalog()
	}
	if product, ok := p.maps[req.Id]; ok {
		span.AddEvent("Product Found")
		span.SetAttributes(
			attribute.String("app.product.id", req.Id),
			attribute.String("app.product.name", product.Name),
		)
		return product, nil
	}
	span.AddEvent("Product Not Found")
	msg := fmt.Sprintf("Product Not Found: %s", req.Id)
	span.SetStatus(otelcodes.Error, msg)
	span.AddEvent(msg)
	return nil, status.Error(codes.NotFound, msg)
}

// 前端传来搜索的商品名称，调用这个函数进行匹配
// TODO 也许可以用 ES 优化。目前采用的是遍历所有商品的名称和描述的方式
// SearchProducts 搜索产品，由前端服务调用
func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	span := trace.SpanFromContext(ctx)

	var ps []*pb.Product
	for _, product := range p.maps {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	span.SetAttributes(
		attribute.Int("app.products_search.count", len(ps)),
	)

	return &pb.SearchProductsResponse{Results: ps}, nil
}
