package main

import (
	"context"
	pb "github.com/kznLeaf/curated-store/src/frontend/genproto"
)

const (
	avoidNoopCurrencyConversionRPC = false
)

// getProducts 获取所有产品列表
// 调用产品目录服务获取完整的产品列表，包含所有产品的基本信息
//
// 参数:
//
//	ctx - 请求上下文，用于超时控制和取消
//
// 返回:
//
//	[]*pb.Product - 产品列表
//	error - 如果调用失败返回错误
func (fe *frontendServer) GetProducts(ctx context.Context) ([]*pb.Product, error) {

	// 1. 利用注册好的gRPC连接，创建产品目录服务的客户端
	catalogClient := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn)
	
	// 2. 调用 ListProducts 方法获取产品列表
	resp, err := catalogClient.ListProducts(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	
	// 3. 返回产品列表
	return resp.Products, nil
}

// 远程调用产品目录服务，获取指定ID的产品信息
func (fe *frontendServer) GetProduct(ctx context.Context, id string) (*pb.Product, error) {
	// 1. 利用注册好的gRPC连接，创建产品目录服务的客户端
	catalogClient := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn)
	// 2. 调用 GetProduct 方法获取指定ID的产品信息
	resp, err := catalogClient.GetProduct(ctx, &pb.GetProductRequest{Id: id})
	if err != nil {
		return nil, err
	}
	// 3. 返回产品信息
	return resp, nil
}

