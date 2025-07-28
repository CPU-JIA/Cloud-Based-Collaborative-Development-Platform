# API 文档

## 概述

云协作开发平台 RESTful API 提供了完整的项目管理、用户认证、文件操作、团队协作等功能。

## 基础信息

- **Base URL**: `https://api.yourplatform.com/api/v1`
- **API版本**: v1
- **认证方式**: JWT Bearer Token
- **数据格式**: JSON
- **字符编码**: UTF-8

## 认证

### JWT Token

所有需要认证的接口都需要在请求头中包含 JWT Token：

```http
Authorization: Bearer <your-jwt-token>
```

### 获取Token

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "expires_in": 3600,
    "user": {
      "id": "user-id",
      "email": "user@example.com",
      "name": "User Name"
    }
  }
}
```

## 统一响应格式

### 成功响应
```json
{
  "success": true,
  "data": {}, // 响应数据
  "meta": {   // 可选的元数据
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100
    }
  }
}
```

### 错误响应
```json
{
  "success": false,
  "error": {
    "code": "VALIDATION_ERROR",
    "message": "输入验证失败",
    "details": [
      {
        "field": "email",
        "message": "邮箱格式不正确"
      }
    ]
  }
}
```

## 用户认证服务 (IAM)

### 用户注册

```http
POST /auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123",
  "name": "用户名",
  "organization": "组织名称"
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "user_id": "user-uuid",
    "email": "user@example.com",
    "status": "pending_verification"
  }
}
```

### 用户登录

```http
POST /auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "password123"
}
```

### 刷新Token

```http
POST /auth/refresh
Authorization: Bearer <refresh-token>
```

### 用户登出

```http
POST /auth/logout
Authorization: Bearer <access-token>
```

### 获取用户信息

```http
GET /auth/me
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": {
    "id": "user-uuid",
    "email": "user@example.com",
    "name": "用户名",
    "organization": "组织名称",
    "role": "developer",
    "avatar": "https://avatar.url",
    "created_at": "2025-01-01T00:00:00Z",
    "last_login": "2025-01-01T12:00:00Z"
  }
}
```

## 项目管理服务

### 创建项目

```http
POST /projects
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "name": "项目名称",
  "description": "项目描述",
  "visibility": "private", // public, private, internal
  "template": "web-app",   // 可选的项目模板
  "technologies": ["go", "react", "postgresql"]
}
```

**响应:**
```json
{
  "success": true,
  "data": {
    "id": "project-uuid",
    "name": "项目名称",
    "description": "项目描述",
    "visibility": "private",
    "owner_id": "user-uuid",
    "status": "active",
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T00:00:00Z"
  }
}
```

### 获取项目列表

```http
GET /projects?page=1&limit=20&status=active&visibility=private
Authorization: Bearer <access-token>
```

**查询参数:**
- `page`: 页码 (默认: 1)
- `limit`: 每页数量 (默认: 20, 最大: 100)
- `status`: 项目状态 (active, archived, deleted)
- `visibility`: 可见性 (public, private, internal)
- `search`: 搜索关键词
- `sort`: 排序字段 (name, created_at, updated_at)
- `order`: 排序方向 (asc, desc)

**响应:**
```json
{
  "success": true,
  "data": [
    {
      "id": "project-uuid",
      "name": "项目名称",
      "description": "项目描述",
      "visibility": "private",
      "owner": {
        "id": "user-uuid",
        "name": "用户名",
        "avatar": "https://avatar.url"
      },
      "stats": {
        "files_count": 150,
        "members_count": 5,
        "commits_count": 245
      },
      "created_at": "2025-01-01T00:00:00Z",
      "updated_at": "2025-01-01T12:00:00Z"
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 100,
      "total_pages": 5
    }
  }
}
```

### 获取项目详情

```http
GET /projects/{project_id}
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": {
    "id": "project-uuid",
    "name": "项目名称",
    "description": "项目描述",
    "visibility": "private",
    "owner": {
      "id": "user-uuid",
      "name": "用户名",
      "email": "user@example.com"
    },
    "technologies": ["go", "react", "postgresql"],
    "repository": {
      "url": "https://github.com/user/repo",
      "branch": "main",
      "last_commit": {
        "hash": "abc123",
        "message": "feat: add new feature",
        "author": "用户名",
        "timestamp": "2025-01-01T12:00:00Z"
      }
    },
    "stats": {
      "files_count": 150,
      "members_count": 5,
      "commits_count": 245,
      "issues_count": 12,
      "prs_count": 3
    },
    "settings": {
      "auto_backup": true,
      "ci_enabled": true,
      "notifications": true
    },
    "created_at": "2025-01-01T00:00:00Z",
    "updated_at": "2025-01-01T12:00:00Z"
  }
}
```

### 更新项目

```http
PUT /projects/{project_id}
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "name": "新项目名称",
  "description": "新项目描述",
  "visibility": "public"
}
```

### 删除项目

```http
DELETE /projects/{project_id}
Authorization: Bearer <access-token>
```

## 文件管理服务

### 上传文件

```http
POST /projects/{project_id}/files
Authorization: Bearer <access-token>
Content-Type: multipart/form-data

file: <binary-data>
path: /src/main.go
message: "添加主程序文件"
```

**响应:**
```json
{
  "success": true,
  "data": {
    "id": "file-uuid",
    "name": "main.go",
    "path": "/src/main.go",
    "size": 1024,
    "mime_type": "text/x-go",
    "hash": "sha256:abc123...",
    "url": "https://files.platform.com/projects/project-id/files/file-id",
    "created_at": "2025-01-01T12:00:00Z"
  }
}
```

### 获取文件列表

```http
GET /projects/{project_id}/files?path=/src&recursive=true
Authorization: Bearer <access-token>
```

**查询参数:**
- `path`: 目录路径 (默认: /)
- `recursive`: 是否递归获取子目录 (默认: false)
- `type`: 文件类型过滤 (file, directory)
- `extension`: 文件扩展名过滤 (.go, .js, .md)

**响应:**
```json
{
  "success": true,
  "data": [
    {
      "id": "file-uuid",
      "name": "main.go",
      "path": "/src/main.go",
      "type": "file",
      "size": 1024,
      "mime_type": "text/x-go",
      "last_modified": "2025-01-01T12:00:00Z",
      "author": {
        "id": "user-uuid",
        "name": "用户名"
      }
    },
    {
      "id": "dir-uuid",
      "name": "utils",
      "path": "/src/utils",
      "type": "directory",
      "children_count": 5,
      "last_modified": "2025-01-01T11:00:00Z"
    }
  ]
}
```

### 获取文件内容

```http
GET /projects/{project_id}/files/{file_id}/content
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": {
    "content": "package main\n\nimport \"fmt\"\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}",
    "encoding": "utf-8",
    "lines": 7,
    "size": 87
  }
}
```

### 更新文件

```http
PUT /projects/{project_id}/files/{file_id}
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "content": "更新后的文件内容",
  "message": "修复bug",
  "encoding": "utf-8"
}
```

### 删除文件

```http
DELETE /projects/{project_id}/files/{file_id}
Authorization: Bearer <access-token>
```

## Git集成服务

### 初始化Git仓库

```http
POST /projects/{project_id}/git/init
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "provider": "github", // github, gitlab, bitbucket
  "repository_name": "my-project",
  "private": true,
  "auto_push": true
}
```

### 提交更改

```http
POST /projects/{project_id}/git/commit
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "message": "feat: 添加新功能",
  "files": [
    {
      "path": "/src/main.go",
      "action": "modified"
    },
    {
      "path": "/src/utils/helper.go",
      "action": "added"
    }
  ],
  "auto_push": true
}
```

### 获取提交历史

```http
GET /projects/{project_id}/git/commits?page=1&limit=20
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": [
    {
      "hash": "abc123...",
      "message": "feat: 添加新功能",
      "author": {
        "name": "用户名",
        "email": "user@example.com"
      },
      "timestamp": "2025-01-01T12:00:00Z",
      "files_changed": 3,
      "insertions": 25,
      "deletions": 5
    }
  ],
  "meta": {
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 50
    }
  }
}
```

### 创建分支

```http
POST /projects/{project_id}/git/branches
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "name": "feature/new-feature",
  "from": "main"
}
```

### 合并分支

```http
POST /projects/{project_id}/git/merge
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "source_branch": "feature/new-feature",
  "target_branch": "main",
  "message": "合并新功能分支",
  "delete_source": true
}
```

## 团队协作服务

### 邀请成员

```http
POST /projects/{project_id}/members
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "email": "member@example.com",
  "role": "developer", // owner, maintainer, developer, viewer
  "message": "邀请您加入项目"
}
```

### 获取成员列表

```http
GET /projects/{project_id}/members
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": [
    {
      "id": "member-uuid",
      "user": {
        "id": "user-uuid",
        "name": "用户名",
        "email": "user@example.com",
        "avatar": "https://avatar.url"
      },
      "role": "developer",
      "status": "active", // pending, active, inactive
      "joined_at": "2025-01-01T00:00:00Z",
      "last_activity": "2025-01-01T12:00:00Z"
    }
  ]
}
```

### 更新成员权限

```http
PUT /projects/{project_id}/members/{member_id}
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "role": "maintainer"
}
```

### 移除成员

```http
DELETE /projects/{project_id}/members/{member_id}
Authorization: Bearer <access-token>
```

## 通知服务

### 获取通知列表

```http
GET /notifications?page=1&limit=20&read=false
Authorization: Bearer <access-token>
```

**查询参数:**
- `page`: 页码
- `limit`: 每页数量
- `read`: 是否已读 (true, false)
- `type`: 通知类型 (mention, invitation, system)

**响应:**
```json
{
  "success": true,
  "data": [
    {
      "id": "notification-uuid",
      "type": "mention",
      "title": "您在项目中被提到",
      "message": "用户名 在项目 '项目名称' 中提到了您",
      "data": {
        "project_id": "project-uuid",
        "user_id": "user-uuid",
        "comment_id": "comment-uuid"
      },
      "read": false,
      "created_at": "2025-01-01T12:00:00Z"
    }
  ],
  "meta": {
    "unread_count": 5,
    "pagination": {
      "page": 1,
      "limit": 20,
      "total": 25
    }
  }
}
```

### 标记为已读

```http
PUT /notifications/{notification_id}/read
Authorization: Bearer <access-token>
```

### 标记全部为已读

```http
PUT /notifications/read-all
Authorization: Bearer <access-token>
```

## CI/CD 服务

### 创建流水线

```http
POST /projects/{project_id}/pipelines
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "name": "CI Pipeline",
  "trigger": "push", // push, pull_request, manual, schedule
  "branch": "main",
  "config": {
    "build": {
      "image": "golang:1.23",
      "commands": [
        "go mod download",
        "go build -o app .",
        "go test ./..."
      ]
    },
    "deploy": {
      "environment": "staging",
      "target": "kubernetes"
    }
  }
}
```

### 触发流水线

```http
POST /projects/{project_id}/pipelines/{pipeline_id}/run
Authorization: Bearer <access-token>
Content-Type: application/json

{
  "branch": "main",
  "commit": "abc123...",
  "environment": "staging"
}
```

### 获取流水线状态

```http
GET /projects/{project_id}/pipelines/{pipeline_id}/runs/{run_id}
Authorization: Bearer <access-token>
```

**响应:**
```json
{
  "success": true,
  "data": {
    "id": "run-uuid",
    "pipeline_id": "pipeline-uuid",
    "status": "running", // pending, running, success, failed, cancelled
    "branch": "main",
    "commit": "abc123...",
    "stages": [
      {
        "name": "build",
        "status": "success",
        "started_at": "2025-01-01T12:00:00Z",
        "finished_at": "2025-01-01T12:05:00Z",
        "duration": 300
      },
      {
        "name": "test",
        "status": "running",
        "started_at": "2025-01-01T12:05:00Z"
      }
    ],
    "created_at": "2025-01-01T12:00:00Z"
  }
}
```

## 错误代码

| 状态码 | 错误代码 | 描述 |
|--------|----------|------|
| 400 | VALIDATION_ERROR | 请求参数验证失败 |
| 401 | UNAUTHORIZED | 未认证或Token无效 |
| 403 | FORBIDDEN | 权限不足 |
| 404 | NOT_FOUND | 资源不存在 |
| 409 | CONFLICT | 资源冲突 |
| 422 | UNPROCESSABLE_ENTITY | 业务逻辑错误 |
| 429 | RATE_LIMIT_EXCEEDED | 请求频率超限 |
| 500 | INTERNAL_SERVER_ERROR | 服务器内部错误 |
| 502 | BAD_GATEWAY | 网关错误 |
| 503 | SERVICE_UNAVAILABLE | 服务不可用 |

## 速率限制

| 端点类型 | 限制 | 时间窗口 |
|----------|------|----------|
| 认证相关 | 5次 | 1分钟 |
| 文件上传 | 10次 | 1分钟 |
| API调用 | 1000次 | 1小时 |
| WebSocket | 100连接 | 每用户 |

## WebSocket API

### 连接

```javascript
const ws = new WebSocket('wss://api.yourplatform.com/ws?token=your-jwt-token');

ws.onopen = function() {
  console.log('WebSocket连接已建立');
};

ws.onmessage = function(event) {
  const data = JSON.parse(event.data);
  console.log('收到消息:', data);
};
```

### 消息格式

```json
{
  "type": "notification",
  "event": "project.member.added",
  "data": {
    "project_id": "project-uuid",
    "member_id": "user-uuid",
    "member_name": "新成员"
  },
  "timestamp": "2025-01-01T12:00:00Z"
}
```

### 支持的事件

- `project.created` - 项目创建
- `project.updated` - 项目更新
- `project.member.added` - 成员加入
- `project.member.removed` - 成员移除
- `file.uploaded` - 文件上传
- `file.updated` - 文件更新
- `commit.pushed` - 代码提交
- `pipeline.started` - 流水线开始
- `pipeline.completed` - 流水线完成

## SDK 和客户端库

### JavaScript/TypeScript

```bash
npm install @yourplatform/api-client
```

```javascript
import { PlatformClient } from '@yourplatform/api-client';

const client = new PlatformClient({
  baseURL: 'https://api.yourplatform.com',
  token: 'your-jwt-token'
});

// 获取项目列表
const projects = await client.projects.list();

// 创建项目
const newProject = await client.projects.create({
  name: '新项目',
  description: '项目描述'
});
```

### Go

```bash
go get github.com/yourplatform/go-client
```

```go
import "github.com/yourplatform/go-client"

client := platform.NewClient("https://api.yourplatform.com", "your-jwt-token")

projects, err := client.Projects.List(ctx, &platform.ProjectListOptions{
    Page:  1,
    Limit: 20,
})
```

### Python

```bash
pip install yourplatform-client
```

```python
from yourplatform import PlatformClient

client = PlatformClient(
    base_url="https://api.yourplatform.com",
    token="your-jwt-token"
)

projects = client.projects.list(page=1, limit=20)
```

## 变更日志

### v1.0.0 (2025-01-01)
- 初始API版本发布
- 支持用户认证、项目管理、文件操作
- 集成Git功能
- 团队协作功能
- CI/CD流水线
- WebSocket实时通信

---

*最后更新: 2025-07-27*
*API版本: v1.0.0*