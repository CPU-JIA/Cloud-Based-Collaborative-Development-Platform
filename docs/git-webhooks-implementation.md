# Git事件钩子和回调机制实现文档

## 概述

本文档详细介绍了项目服务中Git事件钩子和回调机制的实现，该系统能够接收并处理来自Git网关的事件通知，实现事件驱动的架构。

## 架构组件

### 1. Webhook处理器 (WebhookHandler)

**文件位置**: `internal/project-service/webhook/webhook_handler.go`

**功能特性**:
- 接收HTTP webhook请求
- 验证HMAC-SHA256签名确保安全性
- 解析Git事件负载
- 异步处理事件以提高响应性能
- 支持多种事件类型（仓库、分支、提交、推送、标签）

**支持的事件类型**:
```go
type GitEvent struct {
    EventType    string    `json:"event_type"`    // repository, branch, commit, push, tag
    EventID      string    `json:"event_id"`      // 事件唯一标识
    Timestamp    time.Time `json:"timestamp"`     // 事件时间戳
    ProjectID    string    `json:"project_id"`    // 项目ID
    RepositoryID string    `json:"repository_id"` // 仓库ID
    UserID       string    `json:"user_id,omitempty"` // 用户ID
    Payload      json.RawMessage `json:"payload"`   // 事件负载
}
```

### 2. 事件处理器 (EventProcessor)

**文件位置**: `internal/project-service/webhook/event_processor.go`

**功能特性**:
- 实现具体的事件处理逻辑
- 记录项目和仓库活动日志
- 处理不同类型的Git事件
- 支持扩展的业务逻辑处理

**处理的事件类型**:

1. **仓库事件** (repository):
   - `created` - 仓库创建
   - `updated` - 仓库更新
   - `deleted` - 仓库删除
   - `archived` - 仓库归档
   - `unarchived` - 仓库取消归档

2. **分支事件** (branch):
   - `created` - 分支创建
   - `deleted` - 分支删除
   - `default_changed` - 默认分支变更

3. **提交事件** (commit):
   - `created` - 提交创建

4. **推送事件** (push):
   - 包含推送的提交信息、分支信息等

5. **标签事件** (tag):
   - `created` - 标签创建
   - `deleted` - 标签删除

### 3. 回调管理器 (CallbackManager)

**文件位置**: `internal/project-service/webhook/callback_manager.go`

**功能特性**:
- 向外部系统发送webhook回调
- 支持事件过滤和路由
- 实现重试机制和错误处理
- HMAC签名验证
- 支持自定义HTTP头

**回调配置**:
```go
type CallbackConfig struct {
    URL       string            `json:"url"`        // 回调URL
    Secret    string            `json:"secret"`     // HMAC密钥
    Headers   map[string]string `json:"headers"`    // 自定义HTTP头
    Timeout   time.Duration     `json:"timeout"`    // 超时时间
    RetryMax  int               `json:"retry_max"`  // 最大重试次数
    EventMask []string          `json:"event_mask"` // 事件过滤器
}
```

## 安全机制

### 1. 签名验证

使用HMAC-SHA256算法验证webhook请求的真实性：

```go
func (h *WebhookHandler) verifySignature(c *gin.Context) bool {
    signature := c.GetHeader("X-Hub-Signature-256")
    // 计算期望的签名
    mac := hmac.New(sha256.New, []byte(h.secret))
    mac.Write(body)
    expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))
    return hmac.Equal([]byte(signature), []byte(expectedSignature))
}
```

### 2. 请求验证

- 验证事件结构的完整性
- 检查必要字段的存在
- 验证UUID格式的正确性

## API端点

### 1. Webhook接收端点

```http
POST /api/v1/webhooks/git
Content-Type: application/json
X-Hub-Signature-256: sha256=<signature>

{
  "event_type": "repository",
  "event_id": "uuid",
  "timestamp": "2024-01-01T00:00:00Z",
  "project_id": "uuid",
  "repository_id": "uuid",
  "user_id": "uuid",
  "payload": {
    "action": "created",
    "repository": {
      "id": "uuid",
      "name": "test-repo",
      "project_id": "uuid",
      "visibility": "private",
      "default_branch": "main"
    }
  }
}
```

### 2. 健康检查端点

```http
GET /api/v1/webhooks/health

Response:
{
  "status": "healthy",
  "service": "git-webhook-handler",
  "timestamp": "2024-01-01T00:00:00Z"
}
```

## 配置和部署

### 1. 环境变量

```bash
# Webhook签名密钥
WEBHOOK_SECRET=your-secret-key

# Git网关服务地址
GIT_GATEWAY_URL=http://localhost:8083
```

### 2. 服务集成

在`cmd/project-service/main.go`中的集成：

```go
// 初始化webhook系统
eventProcessor := webhook.NewDefaultEventProcessor(projectRepo, projectService, logger)
webhookSecret := os.Getenv("WEBHOOK_SECRET")
webhookHandler := webhook.NewWebhookHandler(eventProcessor, webhookSecret, logger)

// 集成到项目处理器
projectHandler := handlers.NewProjectHandler(projectService, webhookHandler, logger)

// 注册路由
webhooks := v1.Group("/webhooks")
{
    webhooks.GET("/health", projectHandler.GetWebhookHealth)
    webhooks.POST("/git", projectHandler.HandleGitWebhook)
}
```

## 测试覆盖

### 1. 单元测试

**文件位置**: `test/webhook_test.go`

**测试覆盖**:
- Webhook处理器功能测试
- 事件处理器业务逻辑测试
- 回调管理器功能测试
- 错误处理和边界条件测试

### 2. 测试场景

1. **正常事件处理**:
   - 仓库创建事件
   - 推送事件
   - 其他Git事件类型

2. **错误处理**:
   - 无效的JSON格式
   - 缺少必要字段
   - 签名验证失败

3. **回调功能**:
   - 事件过滤器
   - 重试机制
   - 签名生成

## 性能特性

### 1. 异步处理

- 使用goroutine异步处理事件
- 立即返回HTTP响应，避免阻塞
- 超时控制和panic恢复

### 2. 错误恢复

- 完整的错误处理机制
- 日志记录和监控
- 优雅的降级处理

## 扩展性

### 1. 自定义事件处理器

可以通过实现`EventProcessor`接口来自定义事件处理逻辑：

```go
type EventProcessor interface {
    ProcessRepositoryEvent(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error
    ProcessBranchEvent(ctx context.Context, event *GitEvent, payload *BranchEvent) error
    ProcessCommitEvent(ctx context.Context, event *GitEvent, payload *CommitEvent) error
    ProcessPushEvent(ctx context.Context, event *GitEvent, payload *PushEvent) error
    ProcessTagEvent(ctx context.Context, event *GitEvent, payload *TagEvent) error
}
```

### 2. 事件路由

支持基于事件类型的智能路由和过滤：

```go
// 支持通配符匹配
EventMask: []string{"repository.*", "push", "tag.created"}
```

## 监控和日志

### 1. 结构化日志

使用zap logger记录详细的事件处理信息：

```go
logger.Info("接收到Git事件",
    zap.String("event_id", gitEvent.EventID),
    zap.String("event_type", gitEvent.EventType),
    zap.String("project_id", gitEvent.ProjectID),
    zap.String("repository_id", gitEvent.RepositoryID))
```

### 2. 指标收集

- 事件处理延迟
- 成功/失败率统计
- 事件类型分布

## 最佳实践

### 1. 安全性

- 始终验证webhook签名
- 使用HTTPS传输
- 定期轮换密钥

### 2. 可靠性

- 实现幂等性处理
- 设置合理的超时时间
- 监控处理性能

### 3. 可维护性

- 清晰的事件结构定义
- 完善的错误处理
- 详细的日志记录

## 总结

Git事件钩子和回调机制的实现为项目服务提供了强大的事件驱动能力，能够实时响应Git操作并触发相应的业务逻辑。该系统具有良好的安全性、可扩展性和可维护性，为构建更加智能和响应式的协作开发平台奠定了基础。