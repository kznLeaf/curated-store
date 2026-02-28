import os
import random
import grpc
import demo_pb2
import demo_pb2_grpc
from concurrent import futures
from grpc_health.v1 import health_pb2
from grpc_health.v1 import health_pb2_grpc
from logger import getJSONLogger
# --------------------------------------------------

logger = getJSONLogger("recommendationservice-server")


# gRPC server class， implements recommendation service
# 实现了三个方法：ListRecommendations、Check、Watch
# TODO 目前的 ListRecommendations 实现虽然要求传入userID，但是实际上并没有用到。后续可以给予用户ID做个性化推荐
class RecommendationService(demo_pb2_grpc.RecommendationServiceServicer):
    def ListRecommendations(self, request, context):
        max_responses = 5
        cat_response = product_catalog_stub.ListProducts(
            demo_pb2.Empty()
        )  # 调用产品目录服务的 ListProducts 方法，获取所有产品信息
        product_ids = [
            product.id for product in cat_response.products
        ]  # 从产品信息中提取产品 ID 列表
        filtered_ids = list(
            set(product_ids) - set(request.product_ids)
        )  # 从产品 ID 列表中去除用户已经购买的产品 ID
        num_return = min(
            max_responses, len(filtered_ids)
        )  # 计算实际返回的推荐产品数量，最多不超过 max_responses

        indices = random.sample(
            range(len(filtered_ids)), num_return
        )  # 从过滤后的产品 ID 列表中随机选择 num_return 个产品 ID 的索引
        product_list = [
            filtered_ids[i] for i in indices
        ]  # 根据随机选择的索引，从过滤后的产品 ID 列表中获取推荐的产品 ID 列表
        logger.info(
            "[recommandation service][Recv ListRecommendations] product_ids={}".format(
                product_list
            )
        )
        # 构建返回值
        response = demo_pb2.ListRecommendationsResponse()
        response.product_ids.extend(
            product_list
        )  # 将推荐的产品 ID 列表添加到响应对象中
        return response

    def Check(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.SERVING
        )

    def Watch(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.UNIMPLEMENTED
        )

    def List(self, request, context):
        return demo_pb2.ListRecommendationsResponse(
            status=health_pb2.HealthCheckResponse.UNIMPLEMENTED
        )


if __name__ == "__main__":
    logger.info("Starting recommendation service...")

    port = os.environ.get("PORT", "8080")
    catalog_addr = os.environ.get(
        "PRODUCT_CATALOG_SERVICE_ADDR", ""
    )  # 获取产品目录服务的地址
    if catalog_addr == "":
        raise Exception("PRODUCT_CATALOG_SERVICE_ADDR environment variable not set")
    logger.info("Product catalog service address: %s", catalog_addr)

    channel = grpc.insecure_channel(catalog_addr)  # 创建 gRPC 连接
    product_catalog_stub = demo_pb2_grpc.ProductCatalogServiceStub(
        channel
    )  # 创建产品目录服务的 stub

    # 创建 gRPC Server
    server = grpc.server(
        futures.ThreadPoolExecutor(max_workers=10)
    )  # 创建一个 gRPC 服务器，使用线程池执行器处理请求

    # 将 RecommendationService 添加到 gRPC Server 中
    service = RecommendationService()
    demo_pb2_grpc.add_RecommendationServiceServicer_to_server(service, server)
    health_pb2_grpc.add_HealthServicer_to_server(
        service, server
    )  # 将健康检查服务添加到 gRPC Server 中

    logger.info("[recommandation service] Listening on port %s", port)
    server.add_insecure_port("[::]:" + port)  # 监听指定端口
    server.start()  # 启动 gRPC 服务器

    server.wait_for_termination()  # 等待服务器终止
