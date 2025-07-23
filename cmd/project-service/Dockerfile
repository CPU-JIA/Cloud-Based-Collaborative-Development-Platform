# 多阶段构建 - 构建阶段
FROM golang:1.21-alpine AS builder

WORKDIR /app
RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o project-service ./cmd/project-service/

# 运行阶段
FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata curl
RUN addgroup -g 1000 appuser && \
    adduser -D -s /bin/sh -u 1000 -G appuser appuser

WORKDIR /root/
COPY --from=builder /app/project-service .
RUN chown appuser:appuser project-service

USER appuser
EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=60s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

CMD ["./project-service"]