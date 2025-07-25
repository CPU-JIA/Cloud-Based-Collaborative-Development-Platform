package main

import (
	"fmt"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

type User struct {
	ID          int    `json:"id"`
	Email       string `json:"email"`
	Name        string `json:"name"`
	DisplayName string `json:"display_name"`
	Username    string `json:"username"`
	Avatar      string `json:"avatar"`
	CreatedAt   string `json:"created_at"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Success     bool   `json:"success"`
	AccessToken string `json:"access_token"`
	User        User   `json:"user"`
	Message     string `json:"message"`
}

type ApiResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Error   string `json:"error"`
}

var mockUser = User{
	ID:          1,
	Email:       "demo@clouddev.com",
	Name:        "演示用户",
	DisplayName: "演示用户",
	Username:    "demo",
	Avatar:      "https://api.dicebear.com/7.x/avataaars/svg?seed=demo",
	CreatedAt:   "2024-01-01T00:00:00Z",
}

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	// CORS设置
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3003", "http://localhost:3000"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service":   "test-login-api",
			"status":    "ok",
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	r.POST("/auth/login", func(c *gin.Context) {
		var req LoginRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			fmt.Printf("❌ JSON绑定错误: %v\n", err)
			c.JSON(400, ApiResponse{
				Success: false,
				Error:   "请求数据格式错误",
			})
			return
		}

		fmt.Printf("🔐 登录请求 - Email: %s, Password: %s\n", req.Email, req.Password)
		fmt.Printf("🔍 预期 - Email: demo@clouddev.com, Password: demo123\n")

		if req.Email == "demo@clouddev.com" && req.Password == "demo123" {
			token := "mock-jwt-token-" + fmt.Sprintf("%d", time.Now().Unix())
			response := LoginResponse{
				Success:     true,
				AccessToken: token,
				User:        mockUser,
				Message:     "登录成功",
			}
			fmt.Printf("✅ 登录成功: %s\n", req.Email)
			c.JSON(200, response)
		} else {
			fmt.Printf("❌ 登录失败: Email=%s, Password=%s\n", req.Email, req.Password)
			c.JSON(401, ApiResponse{
				Success: false,
				Error:   "邮箱或密码错误",
			})
		}
	})

	fmt.Println("🚀 测试登录服务启动在 :8083")
	r.Run(":8083")
}