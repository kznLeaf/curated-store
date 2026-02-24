paymentservice实现的功能：

- gRPC 服务器，暴露 `PaymentService.Charge` 和 `Health.Check` 两个接口
- `Charge` 逻辑：验证信用卡（仅支持 VISA/MasterCard）、验证是否过期、返回随机 transaction_id
- 结构化日志（JSON 格式，带 severity 字段）
- 支持 OpenTelemetry tracing（环境变量控制开关）网络不可用，无法安装依赖或运行，但我可以直接把完整的 Go 代码写出来。JS → Go 的文件对应关系：

| JS 原文件 | Go 文件 | 说明 |
|-----------|---------|------|
| `index.js` | `main.go` | 程序入口，环境变量读取，tracing 开关 |
| `server.js` | `server.go` | gRPC 服务注册和请求处理 |
| `charge.js` | `charge.go` | 信用卡验证逻辑 |
| `logger.js` | `logger.go` | 结构化 JSON 日志，字段名与原版一致 |
| *(无)* | `tracing.go` | OTel 初始化抽离为独立文件 |

`charge.js` 用了 `simple-card-validator` 库，Go 版本改用标准 Luhn 算法 + 前缀规则手动实现，没有引入额外依赖，逻辑完全透明可控。

gRPC 错误处理更规范——原 JS 版本直接把自定义 Error 传给 callback，Go 版本统一通过 `status.Error(codes.InvalidArgument, ...)` 返回，客户端可以正确读取 gRPC 状态码。

Dockerfile 改用 `FROM scratch` 的极简镜像，最终镜像比 Node.js 版本小很多。