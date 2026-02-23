package main

// 本文件用于调用gRPC服务的API,并且封装成更方便使用的函数。handlers.go中会调用这些函数来获取数据并渲染页面。

import (
	"context"
	"fmt"

	pb "github.com/kznLeaf/curated-store/src/frontend/genproto"
)

const (
	// 该标志用在货币代码相同的情况下不再进行额外的RPC调用以转换货币。true表示优化
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

// SearchProducts 搜索产品，由前端服务调用
func (fe *frontendServer) SearchProducts(ctx context.Context, query string) ([]*pb.Product, error) {
	// 1. 利用注册好的gRPC连接，创建产品目录服务的客户端
	catalogClient := pb.NewProductCatalogServiceClient(fe.productCatalogSvcConn)
	// 2. 调用 SearchProducts 方法搜索产品
	resp, err := catalogClient.SearchProducts(ctx, &pb.SearchProductsRequest{Query: query})
	if err != nil {
		return nil, err
	}
	// 3. 返回搜索结果
	return resp.Results, nil
}

//////////////////currencyService//////////////////////////////

// getCurrencies 获取支持的货币列表
// 调用货币服务获取所有支持的货币代码，并与本地白名单进行过滤
//
// 参数:
//
//	ctx - 请求上下文，用于超时控制和取消
//
// 返回:
//
//	[]string - 过滤后的支持货币代码列表
//	error - 如果调用失败返回错误
func (fe *frontendServer) getCurrencies(ctx context.Context) ([]string, error) {
	currencyClient := pb.NewCurrencyServiceClient(fe.currencySvcConn)
	resp, err := currencyClient.GetSupportedCurrencies(ctx, &pb.Empty{})
	if err != nil {
		return nil, err
	}
	return resp.CurrencyCodes, nil
}

func (fe *frontendServer) convertCurrency(ctx context.Context, money *pb.Money, currency string) (*pb.Money, error) {
	if avoidNoopCurrencyConversionRPC && money.GetCurrencyCode() == currency {
		return money, nil
	}
	return pb.NewCurrencyServiceClient(fe.currencySvcConn).
		Convert(ctx, &pb.CurrencyConversionRequest{
			From:   money,
			ToCode: currency})
}

// getShippingQuote 调用 shippingservice 获取运费报价，货币单位是传入的 currency
func (fe *frontendServer) getShippingQuote(ctx context.Context, items []*pb.CartItem, currency string) (*pb.Money, error) {
	shippingClient := pb.NewShippingServiceClient(fe.shippingSvcConn)
	resp, err := shippingClient.GetQuote(ctx, &pb.GetQuoteRequest{
		Items:   items,
		Address: nil, // TODO 目前没有地址信息，包括 getShippingQuote 的实现中实际上也没有用到收货地址，后续再完善。
	})
	if err != nil {
		return nil, err
	}

	localized, err := fe.convertCurrency(ctx, resp.CostUsd, currency)
	if err != nil {
		return nil, fmt.Errorf("[getShippingQuote]: %w", err)
	}
	return localized, nil
}
