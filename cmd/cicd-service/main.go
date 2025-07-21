package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/engine"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/runner"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/scheduler"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/storage"
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

	// 初始化存储管理器
	storageConfig := storage.DefaultConfig()
	
	// 从全局配置更新存储配置
	if cfg.Storage.Local.BasePath != "" {
		storageConfig.Local.BasePath = cfg.Storage.Local.BasePath
	}
	if cfg.Storage.Local.MaxFileSize > 0 {
		storageConfig.Local.MaxFileSize = cfg.Storage.Local.MaxFileSize
	}
	if cfg.Storage.Cache.MaxSize > 0 {
		storageConfig.Cache.MaxSize = cfg.Storage.Cache.MaxSize
	}
	if cfg.Storage.Artifact.RetentionDays > 0 {
		storageConfig.Artifact.RetentionDays = cfg.Storage.Artifact.RetentionDays
	}
	
	storageManager, err := storage.NewStorageManager(storageConfig, zapLoggerInstance)
	if err != nil {
		zapLoggerInstance.Fatal("Failed to initialize storage manager", zap.Error(err))
	}
	
	// 初始化存储管理器
	ctx := context.Background()
	if err := storageManager.Initialize(ctx); err != nil {
		zapLoggerInstance.Fatal("Failed to initialize storage components", zap.Error(err))
	}

	// 初始化依赖
	pipelineRepo := repository.NewPipelineRepository(db.DB)
	
	// 创建执行引擎
	pipelineEngine := engine.NewPipelineEngine(pipelineRepo, storageManager, zapLoggerInstance)
	
	// 创建作业调度器
	schedulerConfig := scheduler.DefaultSchedulerConfig()
	
	// 从全局配置更新调度器配置
	if cfg.CICD.Scheduler.WorkerCount > 0 {
		schedulerConfig.WorkerCount = cfg.CICD.Scheduler.WorkerCount
	}
	if cfg.CICD.Scheduler.QueueSize > 0 {
		schedulerConfig.QueueSize = cfg.CICD.Scheduler.QueueSize
	}
	if cfg.CICD.Scheduler.PollInterval > 0 {
		schedulerConfig.PollInterval = cfg.CICD.Scheduler.PollInterval
	}
	if cfg.CICD.Scheduler.JobTimeout > 0 {
		schedulerConfig.JobTimeout = cfg.CICD.Scheduler.JobTimeout
	}
	schedulerConfig.EnablePriority = cfg.CICD.Scheduler.EnablePriority
	schedulerConfig.EnableLoadBalance = cfg.CICD.Scheduler.EnableLoadBalance
	
	jobScheduler := scheduler.NewJobScheduler(pipelineRepo, pipelineEngine, schedulerConfig, zapLoggerInstance)
	
	// 启动作业调度器
	if err := jobScheduler.Start(ctx); err != nil {
		zapLoggerInstance.Fatal("Failed to start job scheduler", zap.Error(err))
	}
	
	// 创建Runner通信管理器
	runnerCommManager := runner.NewRunnerCommunicationManager(pipelineRepo, pipelineEngine, zapLoggerInstance)
	
	// 启动Runner通信管理器
	if err := runnerCommManager.Start(ctx); err != nil {
		zapLoggerInstance.Fatal("Failed to start runner communication manager", zap.Error(err))
	}
	
	// 注意：暂时跳过调度器和Runner通信的直接连接
	// 这是为了避免循环依赖。作业调度器可以独立工作。
	// 未来可以通过事件系统或消息队列来实现解耦
	
	pipelineService := service.NewPipelineService(pipelineRepo, storageManager, zapLoggerInstance)
	pipelineHandler := handlers.NewPipelineHandler(pipelineService, zapLoggerInstance)

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
				"service": "cicd-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		// 流水线管理路由 - 需要JWT认证
		pipelines := v1.Group("/pipelines")
		pipelines.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 流水线CRUD
			pipelines.POST("", pipelineHandler.CreatePipeline)                    // 创建流水线
			pipelines.GET("", pipelineHandler.ListPipelines)                      // 获取流水线列表
			pipelines.GET("/:id", pipelineHandler.GetPipeline)                    // 获取流水线详情
			pipelines.PUT("/:id", pipelineHandler.UpdatePipeline)                 // 更新流水线
			pipelines.DELETE("/:id", pipelineHandler.DeletePipeline)              // 删除流水线

			// 流水线操作
			pipelines.POST("/:id/trigger", pipelineHandler.TriggerPipeline)       // 触发流水线
			pipelines.GET("/:id/runs", pipelineHandler.GetPipelineRuns)           // 获取流水线运行列表
			pipelines.GET("/:id/stats", pipelineHandler.GetPipelineStats)         // 获取流水线统计
		}

		// 流水线运行管理路由
		pipelineRuns := v1.Group("/pipeline-runs")
		pipelineRuns.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			pipelineRuns.GET("/:id", pipelineHandler.GetPipelineRun)              // 获取运行详情
			pipelineRuns.POST("/:id/cancel", pipelineHandler.CancelPipelineRun)   // 取消运行
			pipelineRuns.POST("/:id/retry", pipelineHandler.RetryPipelineRun)     // 重试运行
			pipelineRuns.GET("/:run_id/jobs", pipelineHandler.GetJobs)            // 获取作业列表
		}

		// 作业管理路由
		jobs := v1.Group("/jobs")
		jobs.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			jobs.GET("/:id", pipelineHandler.GetJob)                              // 获取作业详情
		}

		// 执行器管理路由
		runners := v1.Group("/runners")
		runners.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			runners.POST("", pipelineHandler.RegisterRunner)                      // 注册执行器
			runners.GET("", pipelineHandler.ListRunners)                          // 获取执行器列表
			runners.GET("/:id", pipelineHandler.GetRunner)                        // 获取执行器详情
			runners.PUT("/:id", pipelineHandler.UpdateRunner)                     // 更新执行器
			runners.DELETE("/:id", pipelineHandler.UnregisterRunner)              // 注销执行器
			runners.POST("/:id/heartbeat", pipelineHandler.HeartbeatRunner)       // 执行器心跳
			runners.GET("/:id/stats", pipelineHandler.GetRunnerStats)             // 获取执行器统计
		}
		
		// WebSocket路由 - Runner连接
		v1.GET("/ws/runner", func(c *gin.Context) {
			runnerCommManager.HandleRunnerConnection(c.Writer, c.Request)
		})
	}

	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting CI/CD service on", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server:", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down server...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止作业调度器
	if err := jobScheduler.Stop(); err != nil {
		zapLoggerInstance.Error("Failed to stop job scheduler", zap.Error(err))
	}
	
	// 停止Runner通信管理器
	if err := runnerCommManager.Stop(); err != nil {
		zapLoggerInstance.Error("Failed to stop runner communication manager", zap.Error(err))
	}

	// 关闭存储管理器
	if err := storageManager.Shutdown(shutdownCtx); err != nil {
		zapLoggerInstance.Error("Failed to shutdown storage manager", zap.Error(err))
	}

	if err := srv.Shutdown(shutdownCtx); err != nil {
		appLogger.Fatal("Server forced to shutdown:", err)
	}

	appLogger.Info("Server exited")
}
