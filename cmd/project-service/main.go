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
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handler"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/transaction"
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

	loggerCfg := cfg.Log.ToLoggerConfig()

	appLogger, err := logger.NewZapLogger(loggerCfg)

	// 直接创建zap logger
	zapLoggerInstance, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
	defer zapLoggerInstance.Sync()

	// 连接数据库 - 直接使用cfg.Database
	databaseConfig := database.Config{
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
		LogLevel:        1, // 设置为logger.Silent
	}
	db, err := database.NewPostgresDB(databaseConfig)
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

	// 初始化补偿管理器和分布式事务管理器
	compensationMgr := compensation.NewCompensationManager(gitGatewayClient, zapLoggerInstance)
	transactionMgr := transaction.NewDistributedTransactionManager(projectRepo, gitGatewayClient, compensationMgr, zapLoggerInstance)

	projectService := service.NewProjectServiceWithTransaction(projectRepo, gitGatewayClient, transactionMgr, zapLoggerInstance)

	// 初始化敏捷服务
	agileService := service.NewAgileService(db.DB, zapLoggerInstance)

	// 初始化Dashboard服务 (暂时注释)
	// dashboardService := service.NewDashboardService(db.DB, agileService, zapLoggerInstance)

	// 初始化webhook系统
	eventProcessor := webhook.NewDefaultEventProcessor(projectRepo, projectService, zapLoggerInstance)
	webhookSecret := os.Getenv("WEBHOOK_SECRET") // 从环境变量获取webhook密钥
	if webhookSecret == "" {
		zapLoggerInstance.Warn("WEBHOOK_SECRET未设置，webhook签名验证将被跳过")
	}
	webhookHandler := webhook.NewWebhookHandler(eventProcessor, webhookSecret, zapLoggerInstance)

	projectHandler := handlers.NewProjectHandler(projectService, webhookHandler, zapLoggerInstance)
	gitHandler := handler.NewGitHandler(projectService, zapLoggerInstance)
	agileHandler := handler.NewAgileHandler(agileService, zapLoggerInstance)
	// dashboardHandler := handler.NewDashboardHandler(dashboardService, zapLoggerInstance) // 暂时注释

	// 创建通用logger用于中间件
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
			projects.GET("/:id/members", projectHandler.GetMembers)               // 获取项目成员
			projects.POST("/:id/members", projectHandler.AddMember)               // 添加项目成员
			projects.DELETE("/:id/members/:user_id", projectHandler.RemoveMember) // 移除项目成员

			// Git仓库管理
			projects.POST("/:id/repositories", projectHandler.CreateRepository) // 创建仓库
			projects.GET("/:id/repositories", projectHandler.ListRepositories)  // 获取项目仓库列表

			// 敏捷管理 - Sprint
			projects.POST("/:id/sprints", agileHandler.CreateSprint)                            // 创建Sprint
			projects.GET("/:id/sprints", agileHandler.ListSprints)                              // 获取Sprint列表
			projects.GET("/:id/sprints/:sprintId", agileHandler.GetSprint)                      // 获取Sprint详情
			projects.PUT("/:id/sprints/:sprintId", agileHandler.UpdateSprint)                   // 更新Sprint
			projects.DELETE("/:id/sprints/:sprintId", agileHandler.DeleteSprint)                // 删除Sprint
			projects.POST("/:id/sprints/:sprintId/start", agileHandler.StartSprint)             // 启动Sprint
			projects.POST("/:id/sprints/:sprintId/close", agileHandler.CloseSprint)             // 关闭Sprint
			projects.GET("/:id/sprints/:sprintId/burndown", agileHandler.GetSprintBurndownData) // 燃尽图数据

			// 敏捷管理 - Epic
			projects.POST("/:id/epics", agileHandler.CreateEpic)     // 创建Epic
			projects.GET("/:id/epics", agileHandler.ListEpics)       // 获取Epic列表
			projects.GET("/:id/epics/:epicId", agileHandler.GetEpic) // 获取Epic详情

			// 敏捷管理 - 任务
			projects.POST("/:id/tasks", agileHandler.CreateTask)               // 创建任务
			projects.GET("/:id/tasks", agileHandler.ListTasks)                 // 获取任务列表
			projects.GET("/:id/statistics", agileHandler.GetProjectStatistics) // 获取项目统计数据

			// 任务排序管理
			projects.POST("/:id/tasks/rebalance", agileHandler.RebalanceTaskRanks)    // 重新平衡任务排名
			projects.GET("/:id/tasks/validate-order", agileHandler.ValidateTaskOrder) // 验证任务排序
		}

		// 任务管理路由
		tasks := v1.Group("/tasks")
		tasks.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			tasks.GET("/:taskId", agileHandler.GetTask)                  // 获取任务详情
			tasks.PUT("/:taskId", agileHandler.UpdateTask)               // 更新任务
			tasks.DELETE("/:taskId", agileHandler.DeleteTask)            // 删除任务
			tasks.POST("/:taskId/status", agileHandler.UpdateTaskStatus) // 更新任务状态
			tasks.POST("/:taskId/assign", agileHandler.AssignTask)       // 分配任务

			// 任务拖拽排序
			tasks.POST("/reorder", agileHandler.ReorderTasks)            // 重新排序任务
			tasks.POST("/move", agileHandler.MoveTask)                   // 精确移动任务
			tasks.POST("/batch-reorder", agileHandler.BatchReorderTasks) // 批量重排序任务

			// 任务评论
			tasks.POST("/:taskId/comments", agileHandler.AddTaskComment)  // 添加评论
			tasks.GET("/:taskId/comments", agileHandler.ListTaskComments) // 获取评论列表

			// 工作日志
			tasks.POST("/:taskId/worklogs", agileHandler.LogWork)     // 记录工作日志
			tasks.GET("/:taskId/worklogs", agileHandler.ListWorkLogs) // 获取工作日志
		}

		// 用户工作负载路由
		users := v1.Group("/users")
		users.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			users.GET("/:userId/workload", agileHandler.GetUserWorkload) // 获取用户工作负载
		}

		/* 暂时注释Dashboard相关路由
		// Dashboard和DORA指标路由
		dashboards := v1.Group("/dashboards")
		dashboards.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 仪表板管理
			dashboards.POST("", dashboardHandler.CreateDashboard)                 // 创建仪表板
			dashboards.PUT("/:dashboard_id", dashboardHandler.UpdateDashboard)    // 更新仪表板
			projects.GET("/:project_id/dashboard", dashboardHandler.GetDashboard)  // 获取项目仪表板

			// 组件管理
			dashboards.POST("/widgets", dashboardHandler.CreateWidget)            // 创建组件
			dashboards.PUT("/widgets/:widget_id", dashboardHandler.UpdateWidget) // 更新组件
			dashboards.DELETE("/widgets/:widget_id", dashboardHandler.DeleteWidget) // 删除组件
		}

		// DORA指标路由
		projects.GET("/:project_id/dora-metrics", dashboardHandler.GetDORAMetrics) // 获取DORA指标
		projects.GET("/:project_id/health", dashboardHandler.GetProjectHealth)     // 获取项目健康度
		projects.GET("/:project_id/metric-trends", dashboardHandler.GetMetricTrends) // 获取指标趋势

		// 告警规则路由
		alertRules := v1.Group("/alert-rules")
		alertRules.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			alertRules.POST("", dashboardHandler.CreateAlertRule)                    // 创建告警规则
			alertRules.PUT("/:rule_id", dashboardHandler.UpdateAlertRule)           // 更新告警规则
			alertRules.DELETE("/:rule_id", dashboardHandler.DeleteAlertRule)        // 删除告警规则
			projects.GET("/:project_id/alert-rules", dashboardHandler.GetAlertRules) // 获取告警规则列表
		}

		// 批量操作路由
		metrics := v1.Group("/metrics")
		metrics.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			metrics.POST("/batch-update", dashboardHandler.BatchUpdateMetrics) // 批量更新指标
		}
		*/

		// 仓库管理路由
		repositories := v1.Group("/repositories")
		repositories.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			repositories.GET("/:repositoryId", gitHandler.GetRepository)       // 获取仓库详情
			repositories.PUT("/:repositoryId", gitHandler.UpdateRepository)    // 更新仓库
			repositories.DELETE("/:repositoryId", gitHandler.DeleteRepository) // 删除仓库

			// 分支管理
			repositories.POST("/:repositoryId/branches", gitHandler.CreateBranch)               // 创建分支
			repositories.GET("/:repositoryId/branches", gitHandler.ListBranches)                // 获取分支列表
			repositories.DELETE("/:repositoryId/branches/:branchName", gitHandler.DeleteBranch) // 删除分支

			// 合并请求管理 (预留路由，等待Git网关实现)
			// repositories.POST("/:repositoryId/pull-requests", gitHandler.CreatePullRequest)     // 创建合并请求
			// repositories.GET("/:repositoryId/pull-requests/:pullRequestId", gitHandler.GetPullRequest) // 获取合并请求
		}

		// Webhook路由 - 无需JWT认证（来自Git网关的内部调用）
		webhooks := v1.Group("/webhooks")
		{
			webhooks.GET("/health", projectHandler.GetWebhookHealth) // Webhook健康检查
			webhooks.POST("/git", projectHandler.HandleGitWebhook)   // 处理Git事件
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
