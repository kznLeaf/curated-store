package main

import (
	"context"

	"github.com/kznLeaf/curated-store/src/cartservice/cartstore"
	pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type cartService struct {
	pb.UnimplementedCartServiceServer
	store cartstore.ICartStore
}

func NewCartService(store cartstore.ICartStore) *cartService {
	return &cartService{store: store}
}

func (s *cartService) AddItem(ctx context.Context, req *pb.AddItemRequest) (*pb.Empty, error) {
	if req.Item == nil {
		return nil, status.Error(codes.InvalidArgument, "item must not be nil")
	}
	log.Infof("[CartService] AddItem userID=%s productID=%s quantity=%d", req.UserId, req.Item.ProductId, req.Item.Quantity)
	if err := s.store.AddItem(ctx, req.UserId, req.Item.ProductId, req.Item.Quantity); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to add item to cart: %v", err)
	}
	return &pb.Empty{}, nil
}

func (s *cartService) GetCart(ctx context.Context, req *pb.GetCartRequest) (*pb.Cart, error) {
	log.Infof("[CartService] GetCart userID=%s", req.UserId)
	return s.store.GetCart(ctx, req.UserId)
}

func (s *cartService) EmptyCart(ctx context.Context, req *pb.EmptyCartRequest) (*pb.Empty, error) {
	log.Infof("[CartService] EmptyCart userID=%s", req.UserId)
	if err := s.store.EmptyCart(ctx, req.UserId); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to empty cart: %v", err)
	}
	return &pb.Empty{}, nil
}
