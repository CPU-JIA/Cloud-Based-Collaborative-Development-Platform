# 多阶段构建 - Git Gateway Service
FROM golang:1.21-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o git-gateway-service ./cmd/git-gateway-service/

# 运行阶段
FROM alpine:latest

# 安装Git和必要工具
RUN apk --no-cache add ca-certificates tzdata curl git openssh-client
RUN addgroup -g 1000 appuser && \
    adduser -D -s /bin/sh -u 1000 -G appuser appuser

# 创建Git仓库目录
RUN mkdir -p /var/git/repositories && \
    chown -R appuser:appuser /var/git/repositories

WORKDIR /root/
COPY --from=builder /app/git-gateway-service .
RUN chown appuser:appuser git-gateway-service

USER appuser
EXPOSE 8080

# 挂载点
VOLUME ["/var/git/repositories"]

HEALTHCHECK --interval=30s --timeout=3s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./git-gateway-service"]