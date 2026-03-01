// 没有配置任何外部存储（没有 REDIS_ADDR、没有 Spanner、没有 AlloyDB）时，服务会自动降级使用内存存储

package cartstore

import (
	"context"
	"sync"

	pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
)

type MemoryCartStore struct {
	mu    sync.RWMutex
	carts map[string]*pb.Cart
}

func NewMemoryCartStore() *MemoryCartStore {
	return &MemoryCartStore{
		carts: make(map[string]*pb.Cart),
	}
}

func (s *MemoryCartStore) AddItem(_ context.Context, userId, productId string, quantity int32) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cart, exists := s.carts[userId]
	if !exists {
		cart = &pb.Cart{UserId: userId, Items: []*pb.CartItem{}}
		s.carts[userId] = cart
	}

	for _, item := range cart.Items { // 如果购物车中已经有这个商品了，就增加数量
		if item.ProductId == productId {
			item.Quantity += quantity
			return nil
		}
	}

	cart.Items = append(cart.Items, &pb.CartItem{
		ProductId: productId,
		Quantity:  quantity,
	})

	return nil
}

func (s *MemoryCartStore) GetCart(_ context.Context, userId string) (*pb.Cart, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cart, exists := s.carts[userId]
	if !exists { // 如果用户没有购物车，就返回一个空的购物车
		return &pb.Cart{UserId: userId, Items: []*pb.CartItem{}}, nil
	}

	result := &pb.Cart{
		UserId: cart.UserId,
		Items:  make([]*pb.CartItem, len(cart.Items)),
	}
	for i, item := range cart.Items {
		result.Items[i] = &pb.CartItem{
			ProductId: item.ProductId,
			Quantity:  item.Quantity,
		}
	}

	return result, nil
}

func (s *MemoryCartStore) EmptyCart(_ context.Context, userId string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.carts[userId] = &pb.Cart{UserId: userId, Items: []*pb.CartItem{}}
	return nil
}

func (s *MemoryCartStore) Ping(_ context.Context) bool { return true }
