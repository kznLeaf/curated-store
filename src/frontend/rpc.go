package main

// 本文件用于调用gRPC服务的API,并且封装成更方便使用的函数。handlers.go中会调用这些函数来获取数据并渲染页面。

import (
	"context"
	"fmt"
	"time"

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

// getAd 调用 adservice 获取广告
func (fe *frontendServer) getAd(ctx context.Context, ctxKeys []string) ([]*pb.Ad, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Millisecond*100)
	defer cancel()
	resp, err := pb.NewAdServiceClient(fe.adSvcConn).GetAds(ctx, &pb.AdRequest{
		ContextKeys: ctxKeys,
	})
	return resp.GetAds(), err
}

// getRecommendations 获取产品推荐，在 producthandler emptyCartHandler placeOrderHandler 中调用
//
// 根据用户 ID 和购物车中的产品 ID 列表，调用推荐服务获取推荐的相似产品。
// 然后获取每个推荐产品的完整信息，最多返回前 4 个以适应 UI 显示限制。
//
// 参数:
//
//	ctx - 请求上下文，用于超时控制和取消
//	userID - 用户的唯一标识符
//	productIDs - 购物车中当前产品 ID 的列表
//
// 返回:
//
//	[]*pb.Product - 推荐产品的详细信息列表（最多 4 个）
//	error - 如果获取推荐或产品信息失败返回错误
func (fe *frontendServer) getRecommendations(ctx context.Context, userID string, productIDs []string) ([]*pb.Product, error) {
	client := pb.NewRecommendationServiceClient(fe.recommendationSvcConn)
	resp, err := client.ListRecommendations(ctx, &pb.ListRecommendationsRequest{
		UserId:     userID,
		ProductIds: productIDs,
	})
	if err != nil {
		return nil, err
	}
	out := make([]*pb.Product, len(resp.GetProductIds()))
	for i, v := range resp.GetProductIds() {
		p, err := fe.GetProduct(ctx, v)
		if err != nil {
			return nil, fmt.Errorf("failed to get recommended product info (#%s)", v)
		}
		out[i] = p
	}
	if len(out) > 4 {
		out = out[:4] // take only first four to fit the UI
	}
	return out, nil
}

// getCart 获取用户购物车内容
//
// 根据用户 ID 调用购物车服务获取该用户的购物车中所有商品
//
// 参数:
//
//	ctx - 请求上下文，用于超时控制和取消
//	userID - 用户的唯一标识符（通常来自 session ID）
//
// 返回:
//
//	[]*pb.CartItem - 购物车中的商品列表
//	error - 如果调用失败返回错误
func (fe *frontendServer) getCart(ctx context.Context, userID string) ([]*pb.CartItem, error) {
	resp, err := pb.NewCartServiceClient(fe.cartSvcConn).GetCart(ctx, &pb.GetCartRequest{UserId: userID})
	return resp.GetItems(), err
}
