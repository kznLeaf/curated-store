from concurrent import futures
import os

import grpc
import demo_pb2
import demo_pb2_grpc
from grpc_health.v1 import health_pb2
from grpc_health.v1 import health_pb2_grpc

from jinja2 import Environment, FileSystemLoader, select_autoescape

from logger import getJSONLogger

from opentelemetry import trace
from opentelemetry.exporter.otlp.proto.grpc.trace_exporter import OTLPSpanExporter
from opentelemetry.sdk.resources import Resource
from opentelemetry.sdk.trace import TracerProvider
from opentelemetry.sdk.trace.export import BatchSpanProcessor
from opentelemetry.sdk.trace.sampling import ALWAYS_ON
from opentelemetry.instrumentation.grpc import GrpcInstrumentorClient
from opentelemetry.instrumentation.grpc import GrpcInstrumentorServer

logger = getJSONLogger("emailservice-server")

# Loads confirmation email template from file
env = Environment(
    loader=FileSystemLoader("templates"),  # 从 templates 目录加载模板文件
    autoescape=select_autoescape(
        ["html", "xml"]
    ),  # 只对 html 和 xml 文件进行自动转义，其他类型的文件不进行转义
)
template = env.get_template(
    "confirmation.html"
)  # 从 templates 目录中加载 confirmation.html 模板文件


class BaseEmailService:
    def Check(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.SERVING
        )

    def Watch(self, request, context):
        return health_pb2.HealthCheckResponse(
            status=health_pb2.HealthCheckResponse.UNIMPLEMENTED
        )

class DummyEmailService(BaseEmailService):
    def SendOrderConfirmation(self, request, context):
        logger.info(
            "A request to send order confirmation email to {} has been received.".format(
                request.email
            )
        )

        return demo_pb2.Empty()

def must_map_env(key: str) -> str:
    value = os.environ.get(key)
    if value is None:
        raise Exception(f"{key} environment variable must be set")
    return value

if __name__ == "__main__":

    resource = Resource.create(attributes={
        "service.name": "emailservice",
    })
    collector_addr = must_map_env("COLLECTOR_SERVICE_ADDR")
    exporter = OTLPSpanExporter(endpoint=collector_addr, insecure=True)

    tp = TracerProvider(resource=resource, sampler=ALWAYS_ON)
    tp.add_span_processor(BatchSpanProcessor(exporter))
    trace.set_tracer_provider(tp)

    # Instrument gRPC client before making calls
    GrpcInstrumentorClient().instrument()
    GrpcInstrumentorServer().instrument() # Instrument gRPC server before starting the server(Very Important!)

    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    service = DummyEmailService()
    demo_pb2_grpc.add_EmailServiceServicer_to_server(
        service, server
    )  # TODO 这里应该实现真正的 EmailService，而不是 DummyEmailService
    health_pb2_grpc.add_HealthServicer_to_server(service, server)

    port = os.environ.get("PORT", "8080")

    server.add_insecure_port("[::]:" + port)
    server.start()
    logger.info("Email service started on port %s", port)
    try:
        server.wait_for_termination()
    finally:
        tp.shutdown()
