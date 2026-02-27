# Copyright 2020 Google LLC
# Copyright 2026 kznLeaf
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# sed -i 直接修改 Alpine 的包源配置文件，把默认的官方源地址替换成中科大镜像源。
# 两个构建阶段（builder 和 base）都需要各自换源，因为多阶段构建中每个 FROM 都是独立的文件系统。

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