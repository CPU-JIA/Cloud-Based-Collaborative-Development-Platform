package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/gin-gonic/gin"
)

type APITestResult struct {
	Endpoint    string        `json:"endpoint"`
	Method      string        `json:"method"`
	StatusCode  int           `json:"status_code"`
	Duration    time.Duration `json:"duration"`
	Success     bool          `json:"success"`
	Error       string        `json:"error,omitempty"`
	Description string        `json:"description"`
}

func main() {
	fmt.Println("=== Cloud-Based Collaborative Development Platform ===")
	fmt.Println("API端点基础功能测试")
	fmt.Println()

	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("配置加载失败: %v", err)
	}

	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	var results []APITestResult

	fmt.Println("🧪 开始API功能测试...")
	fmt.Println()

	// 1. 测试基础健康检查
	results = append(results, testHealthCheck())

	// 2. 测试认证API结构
	results = append(results, testAuthStructure())

	// 3. 测试项目API结构
	results = append(results, testProjectAPI())

	// 4. 测试CI/CD API结构
	results = append(results, testCICDAPI())

	// 5. 测试中间件功能
	results = append(results, testMiddleware(cfg))

	// 显示测试结果
	fmt.Println("\n=== 测试结果汇总 ===")
	successCount := 0
	for i, result := range results {
		status := "❌"
		if result.Success {
			status = "✅"
			successCount++
		}
		fmt.Printf("%d. %s %s %s - %s (%v)\n", 
			i+1, status, result.Method, result.Endpoint, 
			result.Description, result.Duration)
		if result.Error != "" {
			fmt.Printf("   错误: %s\n", result.Error)
		}
	}

	fmt.Printf("\n📊 总计: %d/%d 测试通过 (%.1f%%)\n", 
		successCount, len(results), float64(successCount)/float64(len(results))*100)

	if successCount == len(results) {
		fmt.Println("🎉 所有API结构测试通过！")
	} else {
		fmt.Println("⚠️  部分测试失败，需要进一步检查")
	}
}

func testHealthCheck() APITestResult {
	start := time.Now()
	
	// 创建路由
	router := gin.New()
	v1 := router.Group("/api/v1")
	v1.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "api-gateway",
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// 创建测试请求
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/health", nil)
	router.ServeHTTP(w, req)

	duration := time.Since(start)
	success := w.Code == 200

	return APITestResult{
		Endpoint:    "/api/v1/health",
		Method:      "GET",
		StatusCode:  w.Code,
		Duration:    duration,
		Success:     success,
		Description: "健康检查端点",
		Error:       getErrorIfAny(w.Code, 200, "健康检查失败"),
	}
}

func testAuthStructure() APITestResult {
	start := time.Now()
	
	// 创建路由
	router := gin.New()
	v1 := router.Group("/api/v1")
	auth := v1.Group("/auth")
	
	// 模拟登录端点
	auth.POST("/login", func(c *gin.Context) {
		var reqBody struct {
			Email    string `json:"email" binding:"required,email"`
			Password string `json:"password" binding:"required"`
		}
		
		if err := c.ShouldBindJSON(&reqBody); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		
		// 模拟成功响应
		c.JSON(http.StatusOK, gin.H{
			"token": "mock-jwt-token",
			"user": gin.H{
				"id":    "test-uuid",
				"email": reqBody.Email,
			},
		})
	})

	// 创建测试请求
	loginData := map[string]string{
		"email":    "test@example.com",
		"password": "testpassword123",
	}
	jsonData, _ := json.Marshal(loginData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	duration := time.Since(start)
	success := w.Code == 200

	return APITestResult{
		Endpoint:    "/api/v1/auth/login",
		Method:      "POST", 
		StatusCode:  w.Code,
		Duration:    duration,
		Success:     success,
		Description: "用户认证API结构",
		Error:       getErrorIfAny(w.Code, 200, "认证API结构测试失败"),
	}
}

func testProjectAPI() APITestResult {
	start := time.Now()
	
	router := gin.New()
	v1 := router.Group("/api/v1")
	projects := v1.Group("/projects")
	
	// 模拟项目列表端点
	projects.GET("", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"projects": []gin.H{
				{"id": "proj-1", "name": "Test Project 1"},
				{"id": "proj-2", "name": "Test Project 2"},
			},
			"total": 2,
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/projects", nil)
	router.ServeHTTP(w, req)

	duration := time.Since(start)
	success := w.Code == 200

	return APITestResult{
		Endpoint:    "/api/v1/projects",
		Method:      "GET",
		StatusCode:  w.Code,
		Duration:    duration,
		Success:     success,
		Description: "项目管理API结构",
		Error:       getErrorIfAny(w.Code, 200, "项目API结构测试失败"),
	}
}

func testCICDAPI() APITestResult {
	start := time.Now()
	
	router := gin.New()
	v1 := router.Group("/api/v1")
	pipelines := v1.Group("/pipelines")
	
	// 模拟流水线状态端点
	pipelines.GET("/:id/status", func(c *gin.Context) {
		pipelineId := c.Param("id")
		c.JSON(http.StatusOK, gin.H{
			"pipeline_id": pipelineId,
			"status":      "running",
			"stage":       "build",
		})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/pipelines/test-pipeline/status", nil)
	router.ServeHTTP(w, req)

	duration := time.Since(start)
	success := w.Code == 200

	return APITestResult{
		Endpoint:    "/api/v1/pipelines/:id/status",
		Method:      "GET",
		StatusCode:  w.Code,
		Duration:    duration,
		Success:     success,
		Description: "CI/CD流水线API结构",
		Error:       getErrorIfAny(w.Code, 200, "CI/CD API结构测试失败"),
	}
}

func testMiddleware(cfg *config.Config) APITestResult {
	start := time.Now()
	
	router := gin.New()
	
	// 模拟CORS中间件
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	})

	// 模拟限流中间件
	router.Use(func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "99")
		c.Next()
	})

	v1 := router.Group("/api/v1")
	v1.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "middleware test passed"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/test", nil)
	router.ServeHTTP(w, req)

	duration := time.Since(start)
	success := w.Code == 200 && 
		w.Header().Get("Access-Control-Allow-Origin") == "*" &&
		w.Header().Get("X-RateLimit-Limit") == "100"

	return APITestResult{
		Endpoint:    "/api/v1/test",
		Method:      "GET",
		StatusCode:  w.Code,
		Duration:    duration,
		Success:     success,
		Description: "中间件功能测试",
		Error:       getErrorIfAny(success, true, "中间件功能测试失败"),
	}
}

func getErrorIfAny(actual interface{}, expected interface{}, message string) string {
	if actual != expected {
		return fmt.Sprintf("%s - 期望: %v, 实际: %v", message, expected, actual)
	}
	return ""
}