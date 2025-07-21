package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/repository"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/service"
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

	// 初始化依赖
	gitRepo := repository.NewGitRepository(db.DB)
	gitService := service.NewGitService(gitRepo, zapLoggerInstance, "/var/git/repositories")
	gitHandler := handlers.NewGitHandler(gitService, zapLoggerInstance)

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
				"service": "git-gateway-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		// 仓库管理路由 - 需要JWT认证
		repositories := v1.Group("/repositories")
		repositories.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 仓库CRUD
			repositories.POST("", gitHandler.CreateRepository)                    // 创建仓库
			repositories.GET("", gitHandler.ListRepositories)                     // 获取仓库列表  
			repositories.GET("/search", gitHandler.SearchRepositories)            // 搜索仓库
			repositories.GET("/:id", gitHandler.GetRepository)                    // 获取仓库详情
			repositories.PUT("/:id", gitHandler.UpdateRepository)                 // 更新仓库
			repositories.DELETE("/:id", gitHandler.DeleteRepository)              // 删除仓库
			repositories.GET("/:id/stats", gitHandler.GetRepositoryStats)         // 获取仓库统计

			// 分支管理
			repositories.POST("/:id/branches", gitHandler.CreateBranch)           // 创建分支
			repositories.GET("/:id/branches", gitHandler.ListBranches)            // 获取分支列表
			repositories.GET("/:id/branches/:branch", gitHandler.GetBranch)       // 获取分支详情
			repositories.DELETE("/:id/branches/:branch", gitHandler.DeleteBranch) // 删除分支
			repositories.PUT("/:id/default-branch", gitHandler.SetDefaultBranch)  // 设置默认分支
			repositories.POST("/:id/merge", gitHandler.MergeBranch)               // 合并分支

			// 提交管理
			repositories.POST("/:id/commits", gitHandler.CreateCommit)            // 创建提交
			repositories.GET("/:id/commits", gitHandler.ListCommits)              // 获取提交列表
			repositories.GET("/:id/commits/:sha", gitHandler.GetCommit)           // 获取提交详情
			repositories.GET("/:id/commits/:sha/diff", gitHandler.GetCommitDiff)  // 获取提交差异
			repositories.GET("/:id/compare", gitHandler.CompareBranches)          // 比较分支

			// 标签管理
			repositories.POST("/:id/tags", gitHandler.CreateTag)                  // 创建标签
			repositories.GET("/:id/tags", gitHandler.ListTags)                    // 获取标签列表
			repositories.GET("/:id/tags/:tag", gitHandler.GetTag)                 // 获取标签详情
			repositories.DELETE("/:id/tags/:tag", gitHandler.DeleteTag)           // 删除标签

			// 文件操作
			repositories.GET("/:id/files", gitHandler.GetFileContent)             // 获取文件内容
			repositories.GET("/:id/tree", gitHandler.GetDirectoryContent)         // 获取目录内容
		}
	}

	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting Git Gateway service on", cfg.Server.Address())
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
