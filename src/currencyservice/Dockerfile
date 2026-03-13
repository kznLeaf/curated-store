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

# 构建阶段，使用负责构建的平台来编译加速构建速度
FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
# 先单独复制依赖文件并下载，只有依赖变化时才重新下载
COPY go.mod go.sum ./
# RUN go mod download
RUN GOPROXY=https://goproxy.cn,direct go mod download

COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
# 禁用CGO,确保生成的二进制文件是完全静态链接的，以便跨平台运行。构建产物--可执行文件server，保存到根目录下的productcatalogservice目录
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /productcatalogservice .

# 运行阶段，先创建一个空镜像，然后从构建阶段复制编译好的二进制文件和必要的证书
FROM scratch

WORKDIR /src
COPY --from=builder /productcatalogservice ./server
# 复制用于HTTPS请求验证的证书：
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt 
COPY currency_conversion.json .

# 设置 Go 运行时行为：程序崩溃时只打印当前 goroutine 的堆栈，而非所有 goroutine（避免日志爆炸）
# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

EXPOSE 7000
ENTRYPOINT ["/src/server"]
