package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/handlers"
	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/config"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/middleware"
	"github.com/gin-gonic/gin"
)

func main() {
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

	// 初始化用户管理服务
	userMgmtService := services.NewUserManagementService(db)

	// 初始化角色管理服务
	roleMgmtService := services.NewRoleManagementService(db)

	// 初始化MFA服务
	mfaService := auth.NewMFAService(auth.MFAConfig{
		Issuer: "Collaborative Platform",
	})
	mfaMgmtService := services.NewMFAManagementService(db, mfaService)

	// 初始化会话管理服务
	sessionMgmtService := services.NewSessionManagementService(db)

	// 初始化SSO服务
	ssoService := services.NewSSOService(db.DB)

	// 初始化API令牌服务
	apiTokenService := services.NewAPITokenService(db.DB)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(userService, appLogger)
	userHandler := handlers.NewUserHandler(userService, userMgmtService, appLogger)
	roleHandler := handlers.NewRoleHandler(roleMgmtService, appLogger)
	mfaHandler := handlers.NewMFAHandler(mfaMgmtService, appLogger)
	sessionHandler := handlers.NewSessionHandler(sessionMgmtService, appLogger)
	ssoHandler := handlers.NewSSOHandler(ssoService, jwtService, appLogger)
	apiTokenHandler := handlers.NewAPITokenHandler(apiTokenService, appLogger)

	// 设置Gin路由
	r := gin.New()

	// 全局中间件
	r.Use(middleware.CORS(cfg.Security.CorsAllowedOrigins))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(appLogger))
	r.Use(middleware.Recovery(appLogger))
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.Timeout(30 * time.Second))

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
			auth.POST("/mfa/verify", mfaHandler.VerifyMFA)
		}

		// SSO路由（部分需要认证）
		sso := v1.Group("/sso")
		{
			// 公开SSO路由
			sso.POST("/initiate", ssoHandler.InitiateSSO)
			sso.POST("/complete", ssoHandler.CompleteSSO)
			sso.GET("/providers/public", ssoHandler.GetSSOProviders) // 公开的提供商列表
		}

		// 用户路由（需要认证）
		protected := v1.Group("")
		protected.Use(middleware.JWTAuth(cfg.Auth.JWTSecret))
		{
			// 用户个人信息
			protected.GET("/auth/profile", authHandler.GetProfile)
			protected.PUT("/auth/profile", authHandler.UpdateProfile)
			protected.POST("/auth/change-password", authHandler.ChangePassword)

			// MFA多因子认证
			mfa := protected.Group("/auth/mfa")
			{
				mfa.POST("/enable", mfaHandler.EnableMFA)
				mfa.POST("/verify-setup", mfaHandler.VerifyMFASetup)
				mfa.GET("/devices", mfaHandler.GetMFADevices)
				mfa.DELETE("/disable", mfaHandler.DisableMFA)
			}

			// 会话管理
			sessions := protected.Group("/auth/sessions")
			{
				sessions.GET("", sessionHandler.GetSessions)
				sessions.DELETE("/:id", sessionHandler.RevokeSession)
				sessions.DELETE("", sessionHandler.RevokeAllSessions)
			}

			// 用户管理（需要管理员权限）
			users := protected.Group("/users")
			users.Use(middleware.RequireRole("admin", "manager"))
			{
				users.GET("", userHandler.GetUsers)
				users.POST("", userHandler.CreateUser)
				users.GET("/:id", userHandler.GetUser)
				users.PUT("/:id", userHandler.UpdateUser)
				users.DELETE("/:id", userHandler.DeleteUser)

				// 管理员会话管理
				users.DELETE("/:user_id/sessions", sessionHandler.AdminRevokeUserSessions)
			}

			// 管理员统计功能
			admin := protected.Group("/admin")
			admin.Use(middleware.RequireRole("admin"))
			{
				admin.GET("/sessions/stats", sessionHandler.GetSessionStats)
			}

			// 角色管理（需要管理员权限）
			roles := protected.Group("/roles")
			roles.Use(middleware.RequireRole("admin"))
			{
				roles.GET("", roleHandler.GetRoles)
				roles.POST("", roleHandler.CreateRole)
				roles.GET("/:id", roleHandler.GetRole)
				roles.PUT("/:id", roleHandler.UpdateRole)
				roles.DELETE("/:id", roleHandler.DeleteRole)

				// 获取可用权限列表
				roles.GET("/permissions", func(c *gin.Context) {
					respHandler := api.NewResponseHandler()
					permissions, err := roleMgmtService.GetAvailablePermissions(c.Request.Context())
					if err != nil {
						respHandler.InternalServerError(c, err.Error())
						return
					}
					respHandler.OK(c, "获取权限列表成功", permissions)
				})
			}

			// SSO管理（需要管理员权限）
			ssoAdmin := protected.Group("/sso")
			ssoAdmin.Use(middleware.RequireRole("admin"))
			{
				ssoAdmin.GET("/providers", ssoHandler.GetSSOProviders)
				ssoAdmin.POST("/providers", ssoHandler.CreateSSOProvider)
				ssoAdmin.GET("/providers/:id", ssoHandler.GetSSOProviderByID)
				ssoAdmin.PUT("/providers/:id", ssoHandler.UpdateSSOProvider)
				ssoAdmin.DELETE("/providers/:id", ssoHandler.DeleteSSOProvider)
			}

			// API令牌管理
			tokens := protected.Group("/tokens")
			{
				tokens.GET("", apiTokenHandler.GetAPITokens)
				tokens.POST("", apiTokenHandler.CreateAPIToken)
				tokens.GET("/scopes", apiTokenHandler.GetAvailableScopes)
				tokens.GET("/:id", apiTokenHandler.GetAPITokenByID)
				tokens.PUT("/:id", apiTokenHandler.UpdateAPIToken)
				tokens.POST("/:id/revoke", apiTokenHandler.RevokeAPIToken)
				tokens.GET("/:id/stats", apiTokenHandler.GetTokenUsageStats)
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
