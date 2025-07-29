package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/consumer"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/notification-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
	// 加载配置
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 初始化日志
	loggerCfg := cfg.Log.ToLoggerConfig()

	appLogger, err := logger.NewZapLogger(loggerCfg)

	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 连接数据库
	dbConfig := cfg.Database.ToDBConfig()
	postgresDB, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		appLogger.Fatal("Failed to connect to database:", err)
	}
	db := postgresDB.DB

	// 初始化存储库
	notificationRepo := repository.NewNotificationRepository(db)
	templateRepo := repository.NewTemplateRepository(db)
	ruleRepo := repository.NewRuleRepository(db)
	deliveryLogRepo := repository.NewDeliveryLogRepository(db)

	// 初始化服务
	templateEngine := services.NewTemplateEngine()

	// TODO: 实现实际的邮件、Webhook、应用内通知服务
	var emailService services.EmailService = &services.MockEmailService{}
	var webhookService services.WebhookService = &services.MockWebhookService{}
	var inAppService services.InAppService = &services.MockInAppService{}

	deliveryService := services.NewDeliveryService(
		notificationRepo,
		deliveryLogRepo,
		emailService,
		webhookService,
		inAppService,
		appLogger,
	)

	notificationService := services.NewNotificationService(
		notificationRepo,
		templateRepo,
		ruleRepo,
		templateEngine,
		deliveryService,
		appLogger,
	)

	// 初始化HTTP处理器
	notificationHandler := handlers.NewNotificationHandler(notificationService, appLogger)

	// 初始化Kafka消费者
	kafkaConfig := consumer.KafkaConfig{
		Brokers: []string{"localhost:9092"}, // TODO: 从配置文件读取
		GroupID: "notification-service",
		Topic:   "platform-events",
	}

	eventConsumer := consumer.NewEventConsumer(kafkaConfig, notificationService, appLogger)

	// 启动Kafka消费者
	if err := eventConsumer.Start(); err != nil {
		appLogger.Fatal("Failed to start event consumer:", err)
	}

	// 初始化HTTP路由
	r := gin.New()

	// 中间件
	r.Use(middleware.CORS(cfg.Security.CorsAllowedOrigins))
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30 * time.Second))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service": "notification-service",
			"status":  "healthy",
			"version": "1.0.0",
		})
	})

	// API路由
	v1 := r.Group("/api/v1")
	{
		// 通知相关接口
		notifications := v1.Group("/notifications")
		notifications.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			notifications.GET("", notificationHandler.GetNotifications)
			notifications.POST("", notificationHandler.CreateNotification)
			notifications.GET("/unread/count", notificationHandler.GetUnreadCount)
			notifications.POST("/:id/read", notificationHandler.MarkAsRead)
			notifications.DELETE("/:id", notificationHandler.DeleteNotification)
			notifications.POST("/:id/retry", notificationHandler.RetryNotification)
			notifications.GET("/correlation/:correlation_id", notificationHandler.GetNotificationsByCorrelationID)
		}

		// TODO: 添加模板管理接口
		// templates := v1.Group("/templates")
		// templates.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		// {
		//     templates.GET("", templateHandler.GetTemplates)
		//     templates.POST("", templateHandler.CreateTemplate)
		//     templates.GET("/:id", templateHandler.GetTemplate)
		//     templates.PUT("/:id", templateHandler.UpdateTemplate)
		//     templates.DELETE("/:id", templateHandler.DeleteTemplate)
		// }

		// TODO: 添加规则管理接口
		// rules := v1.Group("/rules")
		// rules.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		// {
		//     rules.GET("", ruleHandler.GetRules)
		//     rules.POST("", ruleHandler.CreateRule)
		//     rules.GET("/:id", ruleHandler.GetRule)
		//     rules.PUT("/:id", ruleHandler.UpdateRule)
		//     rules.DELETE("/:id", ruleHandler.DeleteRule)
		// }
	}

	// 启动HTTP服务器
	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting Notification service on", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server:", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	appLogger.Info("Shutting down notification service...")

	// 停止Kafka消费者
	if err := eventConsumer.Stop(); err != nil {
		appLogger.Error("Failed to stop event consumer:", err)
	}

	// 关闭HTTP服务器
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		appLogger.Fatal("Server forced to shutdown:", err)
	}

	appLogger.Info("Notification service exited")
}
