package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
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

	// 直接创建zap logger
	zapLoggerInstance, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer zapLoggerInstance.Sync()

	// 连接数据库
	dbConfig := cfg.Database.ToDBConfig().(database.Config)
	db, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		zapLoggerInstance.Fatal("Failed to connect to database", zap.Error(err))
	}

	// 初始化Git网关客户端
	gitGatewayClient := client.NewGitGatewayClient(&client.GitGatewayClientConfig{
		BaseURL: "http://localhost:8083", // 从配置文件读取Git网关地址
		Timeout: 30 * time.Second,
		APIKey:  "", // 如需要API Key，从配置文件读取
		Logger:  zapLoggerInstance,
	})

	// 初始化依赖
	projectRepo := repository.NewProjectRepository(db.DB)
	projectService := service.NewProjectService(projectRepo, gitGatewayClient, zapLoggerInstance)
	
	// 初始化webhook系统
	eventProcessor := webhook.NewDefaultEventProcessor(projectRepo, projectService, zapLoggerInstance)
	webhookSecret := os.Getenv("WEBHOOK_SECRET") // 从环境变量获取webhook密钥
	if webhookSecret == "" {
		zapLoggerInstance.Warn("WEBHOOK_SECRET未设置，webhook签名验证将被跳过")
	}
	webhookHandler := webhook.NewWebhookHandler(eventProcessor, webhookSecret, zapLoggerInstance)
	
	projectHandler := handlers.NewProjectHandler(projectService, webhookHandler, zapLoggerInstance)

	// 创建通用logger用于中间件
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
		log.Fatalf("Failed to initialize app logger: %v", err)
	}

	r := gin.New()

	r.Use(middleware.CORS(cfg.Security.CorsAllowedOrigins))
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30 * time.Second))

	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "project-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		// 项目管理路由 - 需要JWT认证
		projects := v1.Group("/projects")
		projects.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 项目CRUD
			projects.POST("", projectHandler.CreateProject)           // 创建项目
			projects.GET("", projectHandler.ListProjects)             // 获取项目列表
			projects.GET("/my", projectHandler.GetUserProjects)       // 获取用户项目
			projects.GET("/key/:key", projectHandler.GetProjectByKey) // 根据key获取项目
			projects.GET("/:id", projectHandler.GetProject)           // 获取项目详情
			projects.PUT("/:id", projectHandler.UpdateProject)        // 更新项目
			projects.DELETE("/:id", projectHandler.DeleteProject)     // 删除项目

			// 项目成员管理
			projects.GET("/:id/members", projectHandler.GetMembers)         // 获取项目成员
			projects.POST("/:id/members", projectHandler.AddMember)         // 添加项目成员
			projects.DELETE("/:id/members/:user_id", projectHandler.RemoveMember) // 移除项目成员

			// Git仓库管理
			projects.POST("/:id/repositories", projectHandler.CreateRepository)    // 创建仓库
			projects.GET("/:id/repositories", projectHandler.ListRepositories)     // 获取项目仓库列表
		}

		// 仓库管理路由
		repositories := v1.Group("/repositories")
		repositories.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			repositories.GET("/:repository_id", projectHandler.GetRepository)       // 获取仓库详情
			repositories.PUT("/:repository_id", projectHandler.UpdateRepository)   // 更新仓库
			repositories.DELETE("/:repository_id", projectHandler.DeleteRepository) // 删除仓库
		}

		// Webhook路由 - 无需JWT认证（来自Git网关的内部调用）
		webhooks := v1.Group("/webhooks")
		{
			webhooks.GET("/health", projectHandler.GetWebhookHealth)   // Webhook健康检查
			webhooks.POST("/git", projectHandler.HandleGitWebhook)     // 处理Git事件
		}
	}

	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting Project service on", cfg.Server.Address())
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
