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

	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/handlers"
	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/repository"
	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/services"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
)

func main() {
	// 1. 配置加载
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 2. Logger初始化
	loggerCfg := cfg.Log.ToLoggerConfig()

	appLogger, err := logger.NewZapLogger(loggerCfg)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	// 3. 数据库连接
	dbConfig := cfg.Database.ToDBConfig()
	db, err := database.NewPostgresDB(dbConfig)
	if err != nil {
		appLogger.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// 4. Repository层初始化
	tenantRepo := repository.NewTenantRepository(db)
	configRepo := repository.NewConfigRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	brandingRepo := repository.NewBrandingRepository(db)

	// 5. Service层初始化
	tenantService := services.NewTenantService(
		tenantRepo,
		configRepo,
		subscriptionRepo,
		brandingRepo,
	)

	// 6. Handler层初始化
	tenantHandler := handlers.NewTenantHandler(tenantService, appLogger)

	// 7. 路由设置
	r := gin.New()

	// 全局中间件
	r.Use(middleware.CORS(cfg.Security.CorsAllowedOrigins))
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30 * time.Second))

	// API路由组
	v1 := r.Group("/api/v1")
	{
		// 健康检查
		v1.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"service": "tenant-service",
				"status":  "healthy",
				"version": "1.0.0",
			})
		})

		// 租户管理路由
		tenants := v1.Group("/tenants")
		{
			// 公开接口
			tenants.GET("/by-domain", tenantHandler.GetTenantByDomain)
			tenants.GET("/stats", tenantHandler.GetTenantStats)
			tenants.GET("/search", tenantHandler.SearchTenants)

			// 需要认证的接口
			authenticated := tenants.Group("")
			authenticated.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
			{
				// CRUD操作
				authenticated.POST("", tenantHandler.CreateTenant)
				authenticated.GET("", tenantHandler.ListTenants)
				authenticated.GET("/:id", tenantHandler.GetTenant)
				authenticated.PUT("/:id", tenantHandler.UpdateTenant)
				authenticated.DELETE("/:id", tenantHandler.DeleteTenant)

				// 租户状态管理
				authenticated.POST("/:id/activate", tenantHandler.ActivateTenant)
				authenticated.POST("/:id/suspend", tenantHandler.SuspendTenant)

				// 租户完整信息
				authenticated.GET("/:id/with-config", tenantHandler.GetTenantWithConfig)
				authenticated.GET("/:id/complete", tenantHandler.GetTenantWithAll)
			}
		}
	}

	// 8. 服务器配置和启动
	srv := &http.Server{
		Addr:    cfg.Server.Address(),
		Handler: r,
	}

	go func() {
		appLogger.Info("Starting Tenant service", "address", cfg.Server.Address())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			appLogger.Fatal("Failed to start server", "error", err)
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
