# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装依赖
RUN apk add --no-cache git ca-certificates tzdata

# 复制go mod文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o iam-service ./cmd/iam-service/

# 运行阶段
FROM alpine:latest

# 安装依赖
RUN apk --no-cache add ca-certificates tzdata curl

# 创建非root用户
RUN addgroup -g 1000 appuser && \
    adduser -D -s /bin/sh -u 1000 -G appuser appuser

# 设置工作目录
WORKDIR /root/

# 复制二进制文件
COPY --from=builder /app/iam-service .

# 更改所有权
RUN chown appuser:appuser iam-service

# 切换到非root用户
USER appuser

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动应用
CMD ["./iam-service"]