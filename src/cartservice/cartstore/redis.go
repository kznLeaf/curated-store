package cartstore

import (
	// pb "github.com/kznLeaf/curated-store/src/cartservice/genproto"
	"github.com/redis/go-redis/v9"
)

// https://pkg.go.dev/github.com/redis/go-redis/v9#section-readme
// 核心逻辑：以 userId 为 key，把整个 Cart 对象序列化成 Protobuf 二进制存入 Redis。

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