
```
cartservice-go/
├── main.go                    # 入口：存储选择 + gRPC 服务器 + 优雅退出
├── pb/                        # 你提供的 gencode（hipstershop 包）
├── cartstore/
│   ├── cartstore.go           # ICartStore 接口
│   ├── memory.go              # 内存存储（并发安全，RWMutex）
│   ├── redis.go               # Redis（Protobuf 序列化）
│   ├── spanner.go             # Cloud Spanner（可重试事务）
│   └── alloydb.go             # AlloyDB（pgx 连接池 + Secret Manager）
├── services/
│   ├── cart_service.go        # CartService gRPC 实现
│   └── health_service.go      # Health Check gRPC 实现
├── tests/cart_test.go         # 4 个集成测试（bufconn，无需真实端口）
├── Dockerfile                 # 多阶段构建，alpine 基础镜像
└── go.mod
```

**与原 C# 版本相比解决了两个问题：**

- AlloyDB 的 SQL 字符串拼接改为参数化查询，修复了 SQL 注入风险
- AlloyDB 的 `INSERT` 改为 `INSERT ... ON CONFLICT DO UPDATE`（真正的 Upsert），逻辑更正确
- 新增优雅退出（监听 SIGINT/SIGTERM，GracefulStop）

**快速启动：**

```bash
go run .                          # 内存模式
REDIS_ADDR=localhost:6379 go run . # Redis 模式
go test ./tests/... -v            # 测试
```