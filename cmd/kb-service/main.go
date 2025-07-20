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
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

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
	
	// 简化logger初始化，直接使用NewZapLogger
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

	// 暂时跳过数据库连接以专注修复编译问题
	// db, err := database.NewPostgresDB(cfg.Database.ToDBConfig())
	// if err != nil {
	//     appLogger.Fatal("Failed to connect to database:", err)
	// }

	r := gin.New()
	
	r.Use(middleware.CORS())
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30*time.Second))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "kb-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		auth := v1.Group("/auth")
		{
			auth.POST("/login", handleLogin)
			auth.POST("/logout", handleLogout)
			auth.POST("/refresh", handleRefresh)
		}

		users := v1.Group("/users")
		users.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			users.GET("", handleGetUsers)
			users.POST("", handleCreateUser)
			users.GET("/:id", handleGetUser)
			users.PUT("/:id", handleUpdateUser)
			users.DELETE("/:id", handleDeleteUser)
		}

		roles := v1.Group("/roles")
		roles.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			roles.GET("", handleGetRoles)
			roles.POST("", handleCreateRole)
			roles.GET("/:id", handleGetRole)
			roles.PUT("/:id", handleUpdateRole)
			roles.DELETE("/:id", handleDeleteRole)
		}
	}

	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting Knowledge Base service on", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown:", err)
	}

	appLogger.Info("Server exited")
}

func handleLogin(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "login endpoint"})
}

func handleLogout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logout endpoint"})
}

func handleRefresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "refresh endpoint"})
}

func handleGetUsers(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get users endpoint"})
}

func handleCreateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create user endpoint"})
}

func handleGetUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get user endpoint"})
}

func handleUpdateUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "update user endpoint"})
}

func handleDeleteUser(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "delete user endpoint"})
}

func handleGetRoles(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get roles endpoint"})
}

func handleCreateRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "create role endpoint"})
}

func handleGetRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "get role endpoint"})
}

func handleUpdateRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "update role endpoint"})
}

func handleDeleteRole(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "delete role endpoint"})
}