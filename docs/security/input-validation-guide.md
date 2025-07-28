# 输入验证中间件使用指南

## 概述

本指南介绍如何使用增强的输入验证中间件来保护您的应用程序免受常见的安全威胁。

## 支持的安全防护

1. **SQL注入防护** - 检测并阻止SQL注入攻击
2. **XSS防护** - 防止跨站脚本攻击
3. **路径遍历防护** - 阻止文件系统访问攻击
4. **命令注入防护** - 防止系统命令执行攻击
5. **LDAP注入防护** - 阻止LDAP查询注入攻击

## 快速开始

### 基本使用

```go
package main

import (
    "github.com/gin-gonic/gin"
    "github.com/cloud-platform/collaborative-dev/shared/middleware"
)

func main() {
    router := gin.Default()
    
    // 使用默认配置的输入验证中间件
    router.Use(middleware.InputValidation())
    
    // 使用JSON请求体深度验证中间件
    router.Use(middleware.RequestBodyValidation())
    
    // 定义路由
    router.POST("/api/users", createUser)
    router.GET("/api/files/:filename", getFile)
    
    router.Run(":8080")
}
```

### 自定义配置

```go
// 创建自定义配置
config := middleware.ValidatorConfig{
    // 启用所有安全防护
    EnableSQLInjectionProtection:     true,
    EnableXSSProtection:              true,
    EnablePathTraversalProtection:    true,
    EnableCommandInjectionProtection: true,
    EnableLDAPInjectionProtection:    true,
    EnableDeepJSONValidation:         true,
    
    // 设置限制
    MaxRequestSize: 5 * 1024 * 1024, // 5MB
    MaxJSONDepth:   10,               // 最大嵌套10层
    
    // 允许的内容类型
    AllowedContentTypes: []string{
        "application/json",
        "application/x-www-form-urlencoded",
        "multipart/form-data",
    },
    
    // 自定义错误处理
    ErrorHandler: func(c *gin.Context, err error) {
        c.JSON(400, gin.H{
            "error": err.Error(),
            "code":  "VALIDATION_FAILED",
        })
    },
}

// 使用自定义配置
router.Use(middleware.InputValidationWithConfig(config))
router.Use(middleware.RequestBodyValidationWithConfig(config))
```

## 结构体验证

### 使用验证标签

```go
type CreateUserRequest struct {
    Username string `json:"username" validate:"required,min=3,max=20,nosqlinjection,noxss"`
    Email    string `json:"email" validate:"required,email"`
    Password string `json:"password" validate:"required,min=8,strongpassword"`
    Bio      string `json:"bio" validate:"max=500,noxss"`
    Avatar   string `json:"avatar" validate:"omitempty,safeurl"`
}

type FileRequest struct {
    Path     string `json:"path" validate:"required,safepath"`
    Command  string `json:"command" validate:"nocommandinjection"`
    LDAPUser string `json:"ldap_user" validate:"noldapinjection"`
}

func createUser(c *gin.Context) {
    var req CreateUserRequest
    
    // 绑定并验证
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 创建验证器并验证结构体
    validator := middleware.NewInputValidatorMiddleware(nil)
    if err := validator.ValidateStruct(req); err != nil {
        c.JSON(400, gin.H{"error": err.Error()})
        return
    }
    
    // 处理请求...
}
```

### 可用的验证标签

- `nosqlinjection` - 防止SQL注入
- `noxss` - 防止XSS攻击
- `safepath` - 验证安全的文件路径
- `safefilename` - 验证安全的文件名
- `nocommandinjection` - 防止命令注入
- `noldapinjection` - 防止LDAP注入
- `strongpassword` - 要求强密码（至少8位，包含大小写字母、数字和特殊字符）
- `safeurl` - 验证安全的URL（防止javascript:等危险协议）
- `safejson` - 综合验证（防止SQL注入、XSS和命令注入）

## 高级功能

### 深度JSON验证

对于嵌套的JSON数据，使用RequestBodyValidator进行深度扫描：

```go
// 这种嵌套数据会被深度验证
{
    "user": {
        "name": "John Doe",
        "profile": {
            "bio": "<script>alert('xss')</script>",  // 会被检测到
            "interests": [
                "coding",
                "'; DROP TABLE users;--"              // 会被检测到
            ]
        }
    }
}
```

### 性能优化建议

1. **选择性启用防护**：根据实际需求启用相应的防护，避免不必要的检查
   ```go
   config := middleware.ValidatorConfig{
       EnableSQLInjectionProtection: true,  // 只启用需要的
       EnableXSSProtection:          true,
       // 其他保持默认false
   }
   ```

2. **路径特定验证**：对文件相关的端点才启用路径遍历检测
   ```go
   fileRouter := router.Group("/files")
   fileRouter.Use(middleware.InputValidationWithConfig(middleware.ValidatorConfig{
       EnablePathTraversalProtection: true,
   }))
   ```

3. **缓存验证结果**：对于重复的验证，考虑缓存结果

### 自定义验证器

```go
config := middleware.ValidatorConfig{
    CustomValidators: map[string]validator.Func{
        "alphanumeric": func(fl validator.FieldLevel) bool {
            return regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(fl.Field().String())
        },
        "nospaces": func(fl validator.FieldLevel) bool {
            return !strings.Contains(fl.Field().String(), " ")
        },
    },
}

type Request struct {
    Code string `json:"code" validate:"required,alphanumeric"`
    Key  string `json:"key" validate:"required,nospaces"`
}
```

## 错误处理

### 默认错误响应

```json
{
    "error": "输入验证失败",
    "code": "VALIDATION_FAILED",
    "details": "检测到SQL注入攻击：查询参数 'id'"
}
```

### 自定义错误处理

```go
config.ErrorHandler = func(c *gin.Context, err error) {
    // 记录日志
    logger.Warn("验证失败", 
        "ip", c.ClientIP(),
        "path", c.Request.URL.Path,
        "error", err.Error(),
    )
    
    // 根据错误类型返回不同响应
    switch {
    case strings.Contains(err.Error(), "SQL注入"):
        c.JSON(400, gin.H{
            "error": "Invalid input detected",
            "code": "SQL_INJECTION_DETECTED",
        })
    case strings.Contains(err.Error(), "XSS"):
        c.JSON(400, gin.H{
            "error": "Unsafe content detected",
            "code": "XSS_DETECTED",
        })
    default:
        c.JSON(400, gin.H{
            "error": "Validation failed",
            "code": "VALIDATION_ERROR",
        })
    }
}
```

## 最佳实践

1. **分层防御**：输入验证只是第一道防线，还应配合：
   - 参数化查询防止SQL注入
   - 输出编码防止XSS
   - 最小权限原则
   - 安全的会话管理

2. **日志记录**：记录所有验证失败的尝试，用于安全监控
   ```go
   config.Logger = logger.GetLogger()
   ```

3. **定期更新**：保持验证规则的更新，应对新的攻击模式

4. **测试覆盖**：确保所有端点都有适当的验证测试

5. **性能监控**：监控验证中间件的性能影响
   ```go
   // 使用性能测试
   go test -bench=. -benchmem
   ```

## 故障排除

### 常见问题

1. **误报问题**
   - 某些合法输入可能被误判（如包含SQL关键字的文本）
   - 解决方案：使用白名单或调整检测规则

2. **性能影响**
   - 深度JSON验证可能影响性能
   - 解决方案：限制JSON深度，优化正则表达式

3. **编码问题**
   - URL编码的参数可能绕过检测
   - 解决方案：中间件已处理常见编码情况

### 调试技巧

启用详细日志：
```go
config.Logger = logger.NewLogger(logger.Config{
    Level: "debug",
})
```

## 示例项目结构

```
project/
├── main.go
├── middleware/
│   └── security.go         # 安全中间件配置
├── handlers/
│   ├── user.go            # 用户处理器
│   └── file.go            # 文件处理器
├── models/
│   └── requests.go        # 请求结构体定义
└── tests/
    └── security_test.go   # 安全测试
```

## 总结

通过正确使用输入验证中间件，可以有效防御大多数常见的Web应用安全威胁。记住，安全是一个持续的过程，需要：

- 定期审查和更新安全规则
- 进行安全测试和渗透测试
- 保持对新威胁的警觉
- 培训开发团队的安全意识

配合其他安全措施，构建真正安全的应用程序。