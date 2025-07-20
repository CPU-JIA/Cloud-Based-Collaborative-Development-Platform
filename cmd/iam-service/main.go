package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/handlers"
	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	loggerCfg := cfg.Log.ToLoggerConfig().(struct {
		Level      string `json:"level" yaml:"level"`
		Format     string `json:"format" yaml:"format"`
		Output     string `json:"output" yaml:"output"`
		FilePath   string `json:"file_path" yaml:"file_path"`
		MaxSize    int    `json:"max_size" yaml:"max_size"`
		MaxBackups int    `json:"max_backups" yaml:"max_backups"`
		MaxAge     int    `json:"max_age" yaml:"max_age"`
		Compress   bool   `json:"compress" yaml:"compress"`
	})
	
	appLogger, err := logger.NewZapLogger(struct {
		Level      string `json:"level" yaml:"level"`
		Format     string `json:"format" yaml:"format"`
		Output     string `json:"output" yaml:"output"`
		FilePath   string `json:"file_path" yaml:"file_path"`
		MaxSize    int    `json:"max_size" yaml:"max_size"`
		MaxBackups int    `json:"max_backups" yaml:"max_backups"`
		MaxAge     int    `json:"max_age" yaml:"max_age"`
		Compress   bool   `json:"compress" yaml:"compress"`
	}{
		Level:  loggerCfg.Level,
		Format: loggerCfg.Format,
		Output: loggerCfg.Output,
	})
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 连接数据库
	db, err := database.NewPostgresDB(database.Config{
		Host:            cfg.Database.Host,
		Port:            cfg.Database.Port,
		Name:            cfg.Database.Name,
		User:            cfg.Database.User,
		Password:        cfg.Database.Password,
		SSLMode:         cfg.Database.SSLMode,
		MaxOpenConns:    cfg.Database.MaxOpenConns,
		MaxIdleConns:    cfg.Database.MaxIdleConns,
		ConnMaxLifetime: cfg.Database.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.Database.ConnMaxIdleTime,
		LogLevel:        1, // Silent mode for production
	})
	if err != nil {
		appLogger.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// 初始化JWT服务
	jwtService := auth.NewJWTService(
		cfg.Auth.JWTSecret,
		cfg.Auth.JWTExpiration,
		cfg.Auth.RefreshTokenExpiry,
	)

	// 初始化用户服务
	userService := services.NewUserService(db, jwtService, services.UserServiceConfig{
		PasswordMinLength: cfg.Auth.PasswordMinLength,
		MaxLoginAttempts:  cfg.Auth.MaxLoginAttempts,
		LockoutDuration:   cfg.Auth.LockoutDuration,
	})

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(userService, appLogger)

	// 设置Gin路由
	r := gin.New()
	
	// 全局中间件
	r.Use(middleware.CORS(cfg.Security.CorsAllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.Recovery(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30*time.Second))

	// API路由
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "iam-service",
				"status":  "healthy",
				"version": "1.0.0",
				"time":    time.Now().UTC(),
			})
		})

		// 认证路由（无需认证）
		auth := v1.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/validate", authHandler.ValidateToken)
		}

		// 用户路由（需要认证）
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 用户个人信息
			protected.GET("/auth/profile", authHandler.GetProfile)
			protected.PUT("/auth/profile", authHandler.UpdateProfile)
			protected.POST("/auth/change-password", authHandler.ChangePassword)

			// 用户管理（需要管理员权限）
			users := protected.Group("/users")
			users.Use(middleware.RequireRole("admin", "manager"))
			{
				users.GET("", handleGetUsers)
				users.POST("", handleCreateUser)
				users.GET("/:id", handleGetUser)
				users.PUT("/:id", handleUpdateUser)
				users.DELETE("/:id", handleDeleteUser)
			}

			// 角色管理（需要管理员权限）
			roles := protected.Group("/roles")
			roles.Use(middleware.RequireRole("admin"))
			{
				roles.GET("", handleGetRoles)
				roles.POST("", handleCreateRole)
				roles.GET("/:id", handleGetRole)
				roles.PUT("/:id", handleUpdateRole)
				roles.DELETE("/:id", handleDeleteRole)
			}
		}
	}

	// 启动服务器
	srv := &http.Server{
		Addr:         cfg.Server.Address(),
		Handler:      r,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	go func() {
		appLogger.Infof("Starting IAM service on %s", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server:", err)
		}
	}()

	// 优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown:", err)
	}

	appLogger.Info("Server exited gracefully")
}

// TODO: 实现用户管理和角色管理处理函数

func handleGetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get users endpoint - TODO"})
}

func handleCreateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create user endpoint - TODO"})
}

func handleGetUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get user endpoint - TODO"})
}

func handleUpdateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "update user endpoint - TODO"})
}

func handleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "delete user endpoint - TODO"})
}

func handleGetRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get roles endpoint - TODO"})
}

func handleCreateRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create role endpoint - TODO"})
}

func handleGetRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get role endpoint - TODO"})
}

func handleUpdateRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "update role endpoint - TODO"})
}

func handleDeleteRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "delete role endpoint - TODO"})
}