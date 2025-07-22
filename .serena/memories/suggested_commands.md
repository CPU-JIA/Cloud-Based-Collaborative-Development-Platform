# 建议的开发命令

## 构建和运行
```bash
# 构建所有服务
make build

# 运行特定服务
go run cmd/project-service/main.go
go run cmd/git-gateway-service/main.go

# 编译检查
go build ./...
```

## 代码质量
```bash
# 格式化代码
go fmt ./...

# 静态分析
go vet ./...

# 测试
go test ./...
```

## 数据库操作
```bash
# 运行数据库迁移
make migrate-up

# 回滚迁移
make migrate-down
```

## Git操作
```bash
git add .
git commit -m "feat: 描述"
git push origin main
```