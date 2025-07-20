# Cloud-Based Collaborative Development Platform
# 企业级开发构建脚本

.PHONY: help dev build test lint clean docker deploy db-migrate db-reset

# 项目配置
PROJECT_NAME := cloud-collaborative-platform
MODULE_NAME := github.com/cloud-platform/collaborative-dev
DOCKER_REGISTRY := registry.company.com
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
COMMIT_HASH := $(shell git rev-parse HEAD)

# Go配置
GO := go
GOOS := linux
GOARCH := amd64
CGO_ENABLED := 0

# 服务列表
SERVICES := iam-service tenant-service project-service git-gateway-service cicd-service notification-service kb-service

# 构建标志
LDFLAGS := -ldflags "-X main.Version=$(VERSION) -X main.BuildTime=$(BUILD_TIME) -X main.CommitHash=$(COMMIT_HASH) -s -w"

# 默认目标
help: ## 显示帮助信息
	@echo "Cloud-Based Collaborative Development Platform"
	@echo "可用的命令:"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

# =============================================================================
# 开发环境
# =============================================================================

deps: ## 安装依赖
	$(GO) mod download
	$(GO) mod tidy

dev: deps ## 启动开发环境
	@echo "启动开发环境..."
	docker-compose -f deployments/docker/docker-compose.dev.yml up -d
	@echo "等待数据库启动..."
	sleep 5
	$(MAKE) db-migrate
	$(GO) run cmd/iam-service/main.go &
	$(GO) run cmd/tenant-service/main.go &
	$(GO) run cmd/project-service/main.go

dev-stop: ## 停止开发环境
	docker-compose -f deployments/docker/docker-compose.dev.yml down
	pkill -f "go run cmd"

logs: ## 查看开发环境日志
	docker-compose -f deployments/docker/docker-compose.dev.yml logs -f

# =============================================================================
# 构建
# =============================================================================

build: deps ## 构建所有服务
	@echo "构建所有微服务..."
	@for service in $(SERVICES); do \
		echo "构建 $$service..."; \
		GOOS=$(GOOS) GOARCH=$(GOARCH) CGO_ENABLED=$(CGO_ENABLED) \
		$(GO) build $(LDFLAGS) -o bin/$$service cmd/$$service/main.go; \
	done

build-local: deps ## 构建本地版本
	@echo "构建本地版本..."
	@for service in $(SERVICES); do \
		echo "构建 $$service..."; \
		$(GO) build $(LDFLAGS) -o bin/$$service cmd/$$service/main.go; \
	done

# =============================================================================
# 测试
# =============================================================================

test: ## 运行所有测试
	@echo "运行单元测试..."
	$(GO) test -v -race -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html

test-unit: ## 运行单元测试
	$(GO) test -v -short ./...

test-integration: ## 运行集成测试
	$(GO) test -v -tags=integration ./tests/integration/...

test-e2e: ## 运行端到端测试
	$(GO) test -v -tags=e2e ./tests/e2e/...

benchmark: ## 运行性能测试
	$(GO) test -v -bench=. -benchmem ./tests/performance/...

# =============================================================================
# 代码质量
# =============================================================================

lint: ## 代码质量检查
	@echo "运行代码质量检查..."
	golangci-lint run ./...
	go vet ./...
	gofmt -s -l .
	@echo "代码质量检查完成"

format: ## 格式化代码
	gofmt -s -w .
	goimports -w .

security-scan: ## 安全扫描
	gosec ./...
	nancy sleuth

# =============================================================================
# 数据库操作
# =============================================================================

db-migrate: ## 运行数据库迁移
	@echo "运行数据库迁移..."
	cd database && ./scripts/init_database.sh

db-migrate-iam: ## 运行IAM服务数据库迁移
	@echo "运行IAM服务数据库迁移..."
	cd database && ./scripts/run_iam_migrations.sh

db-reset: ## 重置数据库
	@echo "重置数据库..."
	cd database && ./scripts/reset_database.sh

db-backup: ## 备份数据库
	cd database && ./scripts/backup_database.sh

db-status: ## 检查数据库状态
	@echo "检查数据库状态..."
	@cd database && psql $$DATABASE_URL -c "SELECT schemaname,tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;" 2>/dev/null || echo "无法连接到数据库"

db-check-iam: ## 检查IAM表状态
	@echo "检查IAM表状态..."
	@cd database && psql $$DATABASE_URL -c "SELECT table_name FROM information_schema.tables WHERE table_name IN ('users', 'roles', 'permissions', 'user_roles', 'role_permissions', 'user_sessions') ORDER BY table_name;" 2>/dev/null || echo "无法连接到数据库或IAM表不存在"

db-verify-iam: ## 验证IAM迁移
	@echo "验证IAM服务数据库迁移..."
	cd database && ./scripts/verify_iam_migration.sh

# =============================================================================
# Docker构建
# =============================================================================

docker-build: ## 构建Docker镜像
	@echo "构建Docker镜像..."
	@for service in $(SERVICES); do \
		echo "构建 $$service Docker镜像..."; \
		docker build -f deployments/docker/Dockerfile.$$service \
			-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)/$$service:$(VERSION) \
			-t $(DOCKER_REGISTRY)/$(PROJECT_NAME)/$$service:latest \
			.; \
	done

docker-push: docker-build ## 推送Docker镜像
	@echo "推送Docker镜像..."
	@for service in $(SERVICES); do \
		echo "推送 $$service..."; \
		docker push $(DOCKER_REGISTRY)/$(PROJECT_NAME)/$$service:$(VERSION); \
		docker push $(DOCKER_REGISTRY)/$(PROJECT_NAME)/$$service:latest; \
	done

docker-dev: ## 启动Docker开发环境
	docker-compose -f deployments/docker/docker-compose.dev.yml up --build

# =============================================================================
# Kubernetes部署
# =============================================================================

k8s-deploy: ## 部署到Kubernetes
	@echo "部署到Kubernetes..."
	kubectl apply -f deployments/kubernetes/namespace.yaml
	kubectl apply -f deployments/kubernetes/configmap.yaml
	kubectl apply -f deployments/kubernetes/secret.yaml
	kubectl apply -f deployments/kubernetes/services/
	kubectl apply -f deployments/kubernetes/ingress.yaml

k8s-undeploy: ## 从Kubernetes卸载
	kubectl delete -f deployments/kubernetes/

k8s-logs: ## 查看Kubernetes日志
	kubectl logs -f -l app=$(PROJECT_NAME)

# =============================================================================
# 生成代码
# =============================================================================

proto-gen: ## 生成protobuf代码
	@echo "生成protobuf代码..."
	buf generate

openapi-gen: ## 生成OpenAPI文档
	@echo "生成OpenAPI文档..."
	swag init -g cmd/*/main.go -o api/docs

mock-gen: ## 生成Mock代码
	@echo "生成Mock代码..."
	mockgen -source=pkg/interfaces/repository.go -destination=tests/mocks/repository.go

# =============================================================================
# 工具
# =============================================================================

tools-install: ## 安装开发工具
	@echo "安装开发工具..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securecodewarrior/go-tool-nancy@latest
	go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
	go install github.com/swaggo/swag/cmd/swag@latest
	go install github.com/golang/mock/mockgen@latest
	go install golang.org/x/tools/cmd/goimports@latest

clean: ## 清理构建产物
	@echo "清理构建产物..."
	rm -rf bin/
	rm -rf coverage.out coverage.html
	rm -rf vendor/
	$(GO) clean -cache -modcache -testcache

# =============================================================================
# 环境变量检查
# =============================================================================

check-env: ## 检查环境变量
	@echo "检查必要的环境变量..."
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL 未设置" && exit 1)
	@test -n "$(REDIS_URL)" || (echo "REDIS_URL 未设置" && exit 1)
	@test -n "$(KAFKA_BROKERS)" || (echo "KAFKA_BROKERS 未设置" && exit 1)
	@echo "环境变量检查通过"

# =============================================================================
# 文档
# =============================================================================

docs-serve: ## 启动文档服务器
	@echo "启动文档服务器..."
	cd docs && python -m http.server 8080

docs-build: ## 构建文档
	@echo "构建项目文档..."
	# 这里可以添加文档构建逻辑

# =============================================================================
# 发布
# =============================================================================

release: clean test lint docker-build docker-push ## 发布新版本
	@echo "发布版本 $(VERSION)"
	git tag $(VERSION)
	git push origin $(VERSION)

# =============================================================================
# 监控
# =============================================================================

monitoring-up: ## 启动监控栈
	docker-compose -f deployments/docker/docker-compose.monitoring.yml up -d

monitoring-down: ## 停止监控栈
	docker-compose -f deployments/docker/docker-compose.monitoring.yml down

# 版本信息
version: ## 显示版本信息
	@echo "版本: $(VERSION)"
	@echo "构建时间: $(BUILD_TIME)"
	@echo "提交哈希: $(COMMIT_HASH)"