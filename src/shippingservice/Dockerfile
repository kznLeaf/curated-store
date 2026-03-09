FROM --platform=$BUILDPLATFORM golang:1.25.0-alpine AS builder
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
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /shippingservice .

# 运行阶段，先创建一个空镜像，然后从构建阶段复制编译好的二进制文件和必要的证书
FROM scratch

WORKDIR /src
COPY --from=builder /shippingservice ./server
# 复制用于HTTPS请求验证的证书：
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt 

# 设置 Go 运行时行为：程序崩溃时只打印当前 goroutine 的堆栈，而非所有 goroutine（避免日志爆炸）
# Definition of this variable is used by 'skaffold debug' to identify a golang binary.
# Default behavior - a failure prints a stack trace for the current goroutine.
# See https://golang.org/pkg/runtime/
ENV GOTRACEBACK=single

EXPOSE 50051
ENTRYPOINT ["/src/server"]
