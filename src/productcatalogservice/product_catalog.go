package main

import (
	"context"
	pb "github.com/kznLeaf/curated-store/tree/main/src/productcatalogservice/genproto"
	"strings"
	"time"
)

type productCatalog struct {
	pb.UnimplementedProductCatalogServiceServer
	catalog pb.ListProductsResponse
	maps    map[string]*pb.Product
}

// parseCatalog 解析产品目录
func (p *productCatalog) parseCatalog() []*pb.Product {
	if reloadCatalog || len(p.catalog.Products) == 0 {
		// 需要重载
		err := loadCatalog(&p.catalog)
		if err != nil {
			return []*pb.Product{}
		}
		// 构建产品 ID 到产品的映射
		p.maps = make(map[string]*pb.Product)
		for _, product := range p.catalog.Products {
			p.maps[product.Id] = product
		}
	}
	return p.catalog.Products
}

// rpc ListProducts(Empty) returns (ListProductsResponse) {}
func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) {
	time.Sleep(extraLatency)
	return &pb.ListProductsResponse{Products: p.parseCatalog()}, nil
}

// rpc GetProduct(GetProductRequest) returns (Product) {} 查询单个产品的信息
func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) {
	time.Sleep(extraLatency)
	if p.maps == nil {
		p.parseCatalog()
	}
	if product, ok := p.maps[req.Id]; ok {
		return product, nil
	}
	return &pb.Product{}, nil
}

// 前端传来搜索的商品名称，调用这个函数进行匹配
// TODO 也许可以用 ES 优化。目前采用的是遍历匹配的方式
// SearchProducts 搜索产品，由前端服务调用
func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
	time.Sleep(extraLatency)

	var ps []*pb.Product
	for _, product := range p.maps {
		if strings.Contains(strings.ToLower(product.Name), strings.ToLower(req.Query)) ||
			strings.Contains(strings.ToLower(product.Description), strings.ToLower(req.Query)) {
			ps = append(ps, product)
		}
	}

	return &pb.SearchProductsResponse{Results: ps}, nil
}
