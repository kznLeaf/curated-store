FROM python:3.12.8-alpine AS builder 
COPY --from=ghcr.io/astral-sh/uv:0.10.8 /uv /uvx /bin/

# Change the working directory to the `app` directory
WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache wget g++ linux-headers \
    && rm -rf /var/cache/apk/*

# Install dependencies in a single layer to optimize caching
RUN --mount=type=cache,target=/root/.cache/uv \
    --mount=type=bind,source=uv.lock,target=uv.lock \
    --mount=type=bind,source=pyproject.toml,target=pyproject.toml \
    UV_INDEX_URL=https://pypi.mirrors.ustc.edu.cn/simple/ \
    uv sync --frozen --no-install-project --no-dev

# Copy the project into the image
COPY . .

# Sync the project
RUN --mount=type=cache,target=/root/.cache/uv \
    UV_INDEX_URL=https://pypi.mirrors.ustc.edu.cn/simple/ \
    uv sync --frozen --no-dev

# Stage 2: Create the final image
FROM python:3.12.8-alpine AS runtime

WORKDIR /app

RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk add --no-cache libstdc++ \
    && rm -rf /var/cache/apk/*

COPY --from=builder /app/.venv /app/.venv
COPY --from=builder /app/email_server.py /app/email_server.py
COPY --from=builder /app/demo_pb2.py /app/demo_pb2.py
COPY --from=builder /app/demo_pb2_grpc.py /app/demo_pb2_grpc.py
COPY --from=builder /app/logger.py /app/logger.py
COPY --from=builder /app/templates /app/templates

# enable gevent support in debugger
ENV GEVENT_SUPPORT=True

ENV PATH="/app/.venv/bin:$PATH"

ENTRYPOINT ["python", "email_server.py"]