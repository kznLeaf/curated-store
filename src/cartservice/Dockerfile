FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine AS builder
ARG TARGETOS
ARG TARGETARCH

WORKDIR /src
COPY go.mod go.sum ./
RUN GOPROXY=https://goproxy.cn,direct go mod download

COPY . .

# Skaffold passes in debug-oriented compiler flags
ARG SKAFFOLD_GO_GCFLAGS
RUN GOOS=${TARGETOS} GOARCH=${TARGETARCH} CGO_ENABLED=0 go build -gcflags="${SKAFFOLD_GO_GCFLAGS}" -o /cartservice .

FROM scratch

WORKDIR /src
COPY --from=builder /cartservice ./server
# 复制用于HTTPS请求验证的证书：
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt 

ENV GOTRACEBACK=single

EXPOSE 7070
ENTRYPOINT ["/src/server"]