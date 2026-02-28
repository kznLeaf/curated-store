# recommendationservice

## 依赖管理

```bash
python -m venv venv        # 创建虚拟环境
source venv/bin/activate   # 激活虚拟环境，如果使用了.envrc 则可以忽略
pip install pip-tools      # 包含了 pip-compile 命令
pip install grpcio-tools   # 用于生成 proto 产物
pip-compile requirements.in # 生成 requirementes.txt
pip install -r requirements.txt # 每次更新 requirements.in 之后都要用这条命令来刷新
```

为了避免每次进入项目时都手动激活虚拟环境，可以安装[direnv](https://direnv.net/)：

```bash
sudo pacman -S direnv
```

然后在本服务的目录下运行`direnv allow`以启用`.envrc`文件。这样每次切换到该服务目录时都会自动启用虚拟环境。

---

`grpcio-health-checking` 这个包没有附带类型存根（`.pyi` 文件），Pylance 找不到源码来做类型推断，所以会提示“无法从源解析导入“grpc_health.v1.health_pb2”。

gRPC 的交叉版本运行保证，参见： https://protobuf.dev/support/cross-version-runtime-guarantee/ 核心是运行时的版本不能比生成的protoc代码的版本更旧。

## 配置镜像源

[Dockerfile](./Dockerfile)为了加速构建，使用中科大的镜像源，主要是`apk update`和`pip install`需要加速。

```Dockerfile
FROM --platform=$BUILDPLATFORM python:3.14.3-alpine AS base
FROM base AS builder
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache g++ linux-headers \
    && rm -rf /var/cache/apk/*
# get packages
COPY requirements.txt .
# 使用国内镜像源安装 Python 依赖
RUN pip install -r requirements.txt -i https://pypi.mirrors.ustc.edu.cn/simple/

FROM base
ARG PY_VERSION=3.14
# Enable unbuffered logging
ENV PYTHONUNBUFFERED=1
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories \
    && apk update \
    && apk add --no-cache libstdc++ \
    && rm -rf /var/cache/apk/*
# get packages
WORKDIR /recommendationservice
# Grab packages from builder
COPY --from=builder /usr/local/lib/python${PY_VERSION}/ /usr/local/lib/python${PY_VERSION}/
# Add the application
COPY . .
# set listen port
ENV PORT="8080"
EXPOSE 8080
ENTRYPOINT ["python", "recommendation_server.py"]
```

## 格式化代码

```bash
pip install ruff
ruff format . 
ruff check --fix .
```