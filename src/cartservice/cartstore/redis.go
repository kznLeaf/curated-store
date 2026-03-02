// https://pkg.go.dev/github.com/redis/go-redis/v9#section-readme
// 核心逻辑：以 userId 为 key，把整个 Cart 对象序列化成 Protobuf 二进制存入 Redis。
// 数据结构如下：
// Redis Key:   "user-123"
// Redis Value: <Cart 的 Protobuf 二进制>
//
//	└── userId: "user-123"
//	└── items:
//	    ├── {productId: "prod-A", quantity: 2}
//	    └── {productId: "prod-B", quantity: 1}
package cartstore

import (
	"context"
	"errors"

	pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

// RedisCartStore 基于 Redis 的购物车存储。
// Cart 对象以 Protobuf 二进制序列化，以 userId 为 key 存入 Redis。
type RedisCartStore struct {
	client *redis.Client
}

func NewRedisCartStore(addr string) *RedisCartStore {
	client := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &RedisCartStore{client: client}
}

// AddItem 将商品添加到用户的购物车中。实现细节如下：
// 1. 从 Redis 获取当前用户的购物车数据（如果存在）。
// 2. 如果购物车不存在，创建一个新的 Cart 对象。
// 3. 遍历 cart.items 将指定的商品添加到 Cart 中，如果商品已存在则更新数量。
// 4. 将更新后的 Cart 对象序列化成 Protobuf 二进制，并存回 Redis。
func (s *RedisCartStore) AddItem(ctx context.Context, userId, productId string, quantity int32) error {
	cart, err := s.getCartInternal(ctx, userId)
	if err != nil {
		return err
	}

	// 查找商品是否已存在
	var found bool
	for _, item := range cart.Items {
		if item.ProductId == productId {
			item.Quantity += quantity
			found = true
			break
		}
	}

	// 如果商品不存在，添加新商品项
	if !found {
		cart.Items = append(cart.Items, &pb.CartItem{
			ProductId: productId,
			Quantity:  quantity,
		})
	}

	data, err := proto.Marshal(cart)
	if err != nil {
		return status.Errorf(codes.Internal, "Failed to marshal cart data for user %s: %v", userId, err)
	}

	err = s.client.Set(ctx, userId, data, 0).Err()
	if err != nil {
		return status.Errorf(codes.FailedPrecondition, "Failed to save cart for user %s: %v", userId, err)
	}
	return nil
}

func (s *RedisCartStore) GetCart(ctx context.Context, userId string) (*pb.Cart, error) {
	return s.getCartInternal(ctx, userId)
}

func (s *RedisCartStore) EmptyCart(ctx context.Context, userId string) error {
	// 创建一个空的购物车实例，序列化之后直接SET
	emptyCart := &pb.Cart{
		UserId: userId,
	}
	data, err := proto.Marshal(emptyCart)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to serialize empty cart: %v", err)
	}

	if err := s.client.Set(ctx, userId, data, 0).Err(); err != nil {
		return status.Errorf(codes.FailedPrecondition, "can't access redis cart storage: %v", err)
	}
	return nil
}

func (s *RedisCartStore) Ping(ctx context.Context) bool {
	if err := s.client.Ping(ctx).Err(); err != nil {
		return false
	}
	return true
}

// 从 Redis 读取并反序列化购物车数据。
func (s *RedisCartStore) getCartInternal(ctx context.Context, userID string) (*pb.Cart, error) {
	data, err := s.client.Get(ctx, userID).Bytes() // GET key
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// 购物车不存在，返回一个新的 Cart 对象
			return &pb.Cart{UserId: userID}, nil
		}
		return nil, status.Errorf(codes.FailedPrecondition, "Failed to access cart for user %s: %v", userID, err)
	}

	cart := &pb.Cart{}
	if err := proto.Unmarshal(data, cart); err != nil { // 反序列化 Protobuf 二进制
		return nil, status.Errorf(codes.Internal, "Failed to unmarshal cart data for user %s: %v", userID, err)
	}
	return cart, nil
}
