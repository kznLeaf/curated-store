# checkouservice

通过 gRPC 对外提供接口，并在内部协调 6 个下游服务完成完整的下单流程。

---

## 对外暴露的接口

- **`PlaceOrder`**：核心接口，接收用户ID、货币、收货地址、邮箱、信用卡信息，完成整个下单流程并返回订单结果。
- **`Check` / `Watch` / `List`**：gRPC 健康检查接口（`Watch` 未实现）。

---

## 调用的 6 个下游服务

| 服务 | 调用时机 | 做什么 |
|------|----------|--------|
| **CartService** | 下单开始时 | `GetCart` 获取购物车内容；下单完成后 `EmptyCart` 清空购物车 |
| **ProductCatalogService** | 准备订单项时 | `GetProduct` 获取每个商品的最新价格（USD） |
| **CurrencyService** | 准备订单项 + 运费时 | `Convert` 将价格从 JPY 转换为用户所选货币 |
| **ShippingService** | 计算费用和安排发货 | `GetQuote` 计算运费；`ShipOrder` 安排发货并返回物流跟踪ID |
| **PaymentService** | 金额确认后 | `Charge` 扣款，返回交易ID |
| **EmailService** | 发货后 | `SendOrderConfirmation` 发送订单确认邮件 |

---

## 完整下单流程（PlaceOrder）

1. 生成订单UUID
2. 从 CartService 获取购物车
3. 从 ProductCatalogService 获取商品价格 + CurrencyService 转换货币
4. 从 ShippingService 获取运费报价 + CurrencyService 转换货币
5. 计算总金额（商品 × 数量 + 运费）
6. PaymentService 信用卡扣款
7. ShippingService 安排发货
8. CartService 清空购物车 + EmailService 发确认邮件

---

## 基础设施

- **链路追踪**：集成 OpenTelemetry（OTLP over gRPC），支持全链路追踪，可通过 `ENABLE_TRACING=1` 开启。
- **性能分析**：集成 Google Cloud Profiler，可通过 `ENABLE_PROFILER=1` 开启。
- **日志**：使用 logrus，JSON 格式输出，适配云环境日志采集。
- **货币计算**：内置 `money` 包，处理精确的货币加法和乘法（避免浮点误差）。

总结来说，这是一个典型的**微服务编排层**，自身不存储任何业务数据，只负责按顺序调用各专项服务、聚合结果，完成电商下单的完整业务流程。