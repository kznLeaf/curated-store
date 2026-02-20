# 产品目录服务

## 公开的API

```proto
// ---------------Product Catalog----------------

service ProductCatalogService {
    rpc ListProducts(Empty) returns (ListProductsResponse) {}
    rpc GetProduct(GetProductRequest) returns (Product) {}
    rpc SearchProducts(SearchProductsRequest) returns (SearchProductsResponse) {}
}

message Product {
    string id = 1;
    string name = 2;
    string description = 3;
    string picture = 4;
    Money price_usd = 5;

    // Categories such as "clothing" or "kitchen" that can be used to look up
    // other related products.
    repeated string categories = 6;
}

message ListProductsResponse {
    repeated Product products = 1;
}

message GetProductRequest {
    string id = 1;
}

message SearchProductsRequest {
    string query = 1;
}

message SearchProductsResponse {
    repeated Product results = 1;
}
```

共三个服务的实现

```go
func (p *productCatalog) ListProducts(context.Context, *pb.Empty) (*pb.ListProductsResponse, error) 

func (p *productCatalog) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.Product, error) 

func (p *productCatalog) SearchProducts(ctx context.Context, req *pb.SearchProductsRequest) (*pb.SearchProductsResponse, error) {
```

运行`gofmt -s -w .`格式化并简化代码。

在跟目录下运行下面的命令部署应用：

```bash
skaffold run
```

部署之后设置端口转发：

```bash
kubectl port-forward deployment/frontend 8080:8080
```

然后就可以访问`http://localhost:8080/66VCHSJNUP`，测试三个API，查看调试日志：


```bash
$ kubectl get pods
NAME                                     READY   STATUS    RESTARTS   AGE
frontend-664b67c9d4-7t2bv                1/1     Running   0          50s
productcatalogservice-75c69f777d-lbk62   1/1     Running   0          50s

$ kubectl logs frontend-664b67c9d4-7t2bv

$ kubectl logs productcatalogservice-75c69f777d-lbk62
```



