package cartstore

import (
	"context"

	pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
)

// ICartStore 购物车存储统一接口，支持 Redis / Spanner / AlloyDB / 内存 多种后端
type ICartStore interface {
	AddItem(ctx context.Context, userId, productId string, quantity int32) error
	GetCart(ctx context.Context, userId string) (*pb.Cart, error)
	EmptyCart(ctx context.Context, userId string) error
	Ping(ctx context.Context) bool
}
