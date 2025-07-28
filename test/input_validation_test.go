package test

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestInputValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("SQL注入防护测试", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.InputValidation())
		router.Use(middleware.RequestBodyValidation())

		router.GET("/users/:id", func(c *gin.Context) {
			id := c.Param("id")
			c.JSON(200, gin.H{"user_id": id})
		})

		router.POST("/search", func(c *gin.Context) {
			var req struct {
				Query string `json:"query"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"query": req.Query})
		})

		// 测试路径参数SQL注入
		testCases := []struct {
			name       string
			path       string
			expectCode int
		}{
			{"正常ID", "/users/123", 200},
			{"SQL注入-UNION", "/users/1%20UNION%20SELECT%20*%20FROM%20users", 400},
			{"SQL注入-OR", "/users/1%20OR%201=1", 400},
			{"SQL注入-注释", "/users/1;--", 400},
			{"SQL注入-DROP", "/users/1;DROP%20TABLE%20users", 400},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.path, nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, tc.expectCode, resp.Code)
			})
		}

		// 测试请求体SQL注入
		sqlInjectionPayloads := []string{
			`{"query": "'; DROP TABLE users; --"}`,
			`{"query": "1' OR '1'='1"}`,
			`{"query": "1 UNION SELECT * FROM passwords"}`,
			`{"query": "admin'--"}`,
		}

		for _, payload := range sqlInjectionPayloads {
			req := httptest.NewRequest("POST", "/search", bytes.NewBuffer([]byte(payload)))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, 400, resp.Code, "应该拦截SQL注入: %s", payload)
		}

		t.Logf("✅ SQL注入防护测试通过")
	})

	t.Run("XSS防护测试", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.InputValidation())

		router.POST("/comment", func(c *gin.Context) {
			var req struct {
				Content string `json:"content"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, gin.H{"error": err.Error()})
				return
			}
			c.JSON(200, gin.H{"content": req.Content})
		})

		xssPayloads := []string{
			`{"content": "<script>alert('XSS')</script>"}`,
			`{"content": "<img src=x onerror=alert('XSS')>"}`,
			`{"content": "<iframe src='javascript:alert(1)'></iframe>"}`,
			`{"content": "<a href='javascript:void(0)'>Click</a>"}`,
			`{"content": "<svg onload=alert('XSS')>"}`,
		}

		for _, payload := range xssPayloads {
			req := httptest.NewRequest("POST", "/comment", bytes.NewBuffer([]byte(payload)))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, 400, resp.Code, "应该拦截XSS攻击: %s", payload)
		}

		// 测试正常内容
		normalContent := `{"content": "这是一条正常的评论，包含一些特殊字符: <, >, &, ', \""}`
		req := httptest.NewRequest("POST", "/comment", bytes.NewBuffer([]byte(normalContent)))
		req.Header.Set("Content-Type", "application/json")
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)
		assert.Equal(t, 200, resp.Code, "正常内容应该通过")

		t.Logf("✅ XSS防护测试通过")
	})

	t.Run("路径遍历防护测试", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.InputValidation())

		router.GET("/file/*path", func(c *gin.Context) {
			path := c.Param("path")
			c.JSON(200, gin.H{"path": path})
		})

		pathTraversalPayloads := []struct {
			name       string
			path       string
			expectCode int
		}{
			{"正常路径", "/file/documents/report.pdf", 200},
			{"路径遍历-点点斜杠", "/file/../etc/passwd", 400},
			{"路径遍历-编码", "/file/%2e%2e%2f%2e%2e%2fetc%2fpasswd", 400},
			{"路径遍历-双编码", "/file/%252e%252e%252f", 400},
			{"路径遍历-反斜杠", "/file/..\\..\\windows\\system32", 400},
		}

		for _, tc := range pathTraversalPayloads {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("GET", tc.path, nil)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, tc.expectCode, resp.Code)
			})
		}

		t.Logf("✅ 路径遍历防护测试通过")
	})

	t.Run("请求大小限制测试", func(t *testing.T) {
		config := middleware.DefaultValidatorConfig
		config.MaxRequestSize = 1024 // 1KB

		router := gin.New()
		router.Use(middleware.InputValidationWithConfig(config))

		router.POST("/upload", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		// 测试超大请求
		largeData := make([]byte, 2048) // 2KB
		for i := range largeData {
			largeData[i] = 'A'
		}

		req := httptest.NewRequest("POST", "/upload", bytes.NewBuffer(largeData))
		req.Header.Set("Content-Type", "application/octet-stream")
		req.ContentLength = int64(len(largeData))
		resp := httptest.NewRecorder()
		router.ServeHTTP(resp, req)

		assert.Equal(t, 400, resp.Code)

		var result map[string]interface{}
		err := json.Unmarshal(resp.Body.Bytes(), &result)
		assert.NoError(t, err)
		assert.Contains(t, result["details"], "请求大小超过限制")

		t.Logf("✅ 请求大小限制测试通过")
	})

	t.Run("内容类型验证测试", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.InputValidation())

		router.POST("/api/data", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		testCases := []struct {
			name        string
			contentType string
			expectCode  int
		}{
			{"JSON内容", "application/json", 200},
			{"表单内容", "application/x-www-form-urlencoded", 200},
			{"多部分表单", "multipart/form-data", 200},
			{"文本内容", "text/plain", 200},
			{"不支持的类型", "application/xml", 400},
			{"危险类型", "text/html", 400},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				req := httptest.NewRequest("POST", "/api/data", bytes.NewBuffer([]byte("test")))
				req.Header.Set("Content-Type", tc.contentType)
				resp := httptest.NewRecorder()
				router.ServeHTTP(resp, req)
				assert.Equal(t, tc.expectCode, resp.Code)
			})
		}

		t.Logf("✅ 内容类型验证测试通过")
	})
}

func TestStructValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// 定义测试结构体
	type UserRegistration struct {
		Username string `json:"username" validate:"required,min=3,max=20,nosqlinjection,noxss"`
		Password string `json:"password" validate:"required,strongpassword"`
		Email    string `json:"email" validate:"required,email"`
		Website  string `json:"website" validate:"omitempty,url,safeurl"`
	}

	router := gin.New()
	validator := middleware.NewInputValidatorMiddleware(nil)

	router.POST("/register", func(c *gin.Context) {
		var req UserRegistration
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		// 使用自定义验证器
		if err := validator.ValidateStruct(req); err != nil {
			c.JSON(400, gin.H{"error": err.Error()})
			return
		}

		c.JSON(200, gin.H{"status": "registered"})
	})

	testCases := []struct {
		name       string
		payload    interface{}
		expectCode int
	}{
		{
			name: "正常注册",
			payload: UserRegistration{
				Username: "testuser",
				Password: "P@ssw0rd123!",
				Email:    "test@example.com",
				Website:  "https://example.com",
			},
			expectCode: 200,
		},
		{
			name: "用户名包含SQL注入",
			payload: UserRegistration{
				Username: "admin'; DROP TABLE users;--",
				Password: "P@ssw0rd123!",
				Email:    "test@example.com",
			},
			expectCode: 400,
		},
		{
			name: "弱密码",
			payload: UserRegistration{
				Username: "testuser",
				Password: "123456",
				Email:    "test@example.com",
			},
			expectCode: 400,
		},
		{
			name: "危险URL",
			payload: UserRegistration{
				Username: "testuser",
				Password: "P@ssw0rd123!",
				Email:    "test@example.com",
				Website:  "javascript:alert('XSS')",
			},
			expectCode: 400,
		},
		{
			name: "用户名包含XSS",
			payload: UserRegistration{
				Username: "<script>alert('XSS')</script>",
				Password: "P@ssw0rd123!",
				Email:    "test@example.com",
			},
			expectCode: 400,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			jsonData, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest("POST", "/register", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, tc.expectCode, resp.Code)
		})
	}

	t.Logf("✅ 结构体验证测试通过")
}

func TestFormValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.InputValidation())

	router.POST("/form", func(c *gin.Context) {
		name := c.PostForm("name")
		comment := c.PostForm("comment")
		c.JSON(200, gin.H{
			"name":    name,
			"comment": comment,
		})
	})

	// 测试表单SQL注入
	formData := "name=admin'--&comment=test"
	req := httptest.NewRequest("POST", "/form", bytes.NewBufferString(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, 400, resp.Code)

	// 测试表单XSS
	formData = "name=user&comment=<script>alert('XSS')</script>"
	req = httptest.NewRequest("POST", "/form", bytes.NewBufferString(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, 400, resp.Code)

	// 测试正常表单
	formData = "name=testuser&comment=This is a normal comment"
	req = httptest.NewRequest("POST", "/form", bytes.NewBufferString(formData))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)
	assert.Equal(t, 200, resp.Code)

	t.Logf("✅ 表单验证测试通过")
}

func TestQueryParameterValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(middleware.InputValidation())

	router.GET("/search", func(c *gin.Context) {
		query := c.Query("q")
		sort := c.Query("sort")
		c.JSON(200, gin.H{
			"query": query,
			"sort":  sort,
		})
	})

	testCases := []struct {
		name       string
		query      string
		expectCode int
	}{
		{"正常查询", "/search?q=golang&sort=date", 200},
		{"SQL注入查询", "/search?q=test'%20OR%201=1--", 400},
		{"XSS查询", "/search?q=<script>alert('XSS')</script>", 400},
		{"UNION注入", "/search?q=test%20UNION%20SELECT%20*%20FROM%20users", 400},
		{"多个参数注入", "/search?q=test&sort=date;DROP%20TABLE%20users", 400},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tc.query, nil)
			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)
			assert.Equal(t, tc.expectCode, resp.Code)
		})
	}

	t.Logf("✅ 查询参数验证测试通过")
}
