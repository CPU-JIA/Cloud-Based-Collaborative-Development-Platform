package test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/handlers"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// TestWebhookHandling 测试Webhook事件处理
func TestWebhookHandling(t *testing.T) {
	// 设置测试数据库
	db := setupTestDB(t)

	// 创建模拟Git网关客户端
	mockGitClient := new(MockGitGatewayClient)

	// 设置logger
	logger := zap.NewNop()

	// 创建服务和处理器
	projectRepo := repository.NewProjectRepository(db)
	projectService := service.NewProjectService(projectRepo, mockGitClient, logger)

	// 创建webhook系统（不使用secret以简化测试）
	eventProcessor := webhook.NewDefaultEventProcessor(projectRepo, projectService, logger)
	webhookHandler := webhook.NewWebhookHandler(eventProcessor, "", logger)

	projectHandler := handlers.NewProjectHandler(projectService, webhookHandler, logger)

	// 设置路由
	gin.SetMode(gin.TestMode)
	router := gin.New()

	v1 := router.Group("/api/v1")
	{
		webhooks := v1.Group("/webhooks")
		{
			webhooks.GET("/health", projectHandler.GetWebhookHealth)
			webhooks.POST("/git", projectHandler.HandleGitWebhook)
		}
	}

	t.Run("Webhook健康检查", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/webhooks/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "healthy", response["status"])
	})

	t.Run("处理仓库创建事件", func(t *testing.T) {
		// 创建Git事件
		gitEvent := webhook.GitEvent{
			EventType:    "repository",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    uuid.New().String(),
			RepositoryID: uuid.New().String(),
			UserID:       uuid.New().String(),
		}

		// 创建仓库事件负载
		repoEvent := webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:            gitEvent.RepositoryID,
				Name:          "test-webhook-repo",
				ProjectID:     gitEvent.ProjectID,
				Visibility:    "private",
				DefaultBranch: "main",
			},
		}

		// 序列化负载
		payload, err := json.Marshal(repoEvent)
		assert.NoError(t, err)
		gitEvent.Payload = payload

		// 序列化整个事件
		eventData, err := json.Marshal(gitEvent)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/git", bytes.NewBuffer(eventData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// webhook应该接受事件并返回成功
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "事件已接收", response["message"])
		assert.Equal(t, gitEvent.EventID, response["event_id"])
	})

	t.Run("处理推送事件", func(t *testing.T) {
		// 创建Git事件
		gitEvent := webhook.GitEvent{
			EventType:    "push",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    uuid.New().String(),
			RepositoryID: uuid.New().String(),
			UserID:       uuid.New().String(),
		}

		// 创建推送事件负载
		pushEvent := webhook.PushEvent{
			RepositoryID: gitEvent.RepositoryID,
			Branch:       "main",
			Before:       "abc123",
			After:        "def456",
			Commits: []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
				Author  string `json:"author"`
			}{
				{
					SHA:     "def456",
					Message: "Test commit",
					Author:  "test@example.com",
				},
			},
			Pusher: "test-user",
		}

		// 序列化负载
		payload, err := json.Marshal(pushEvent)
		assert.NoError(t, err)
		gitEvent.Payload = payload

		// 序列化整个事件
		eventData, err := json.Marshal(gitEvent)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/git", bytes.NewBuffer(eventData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// webhook应该接受事件并返回成功
		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "事件已接收", response["message"])
		assert.Equal(t, gitEvent.EventID, response["event_id"])
	})

	t.Run("处理无效事件格式", func(t *testing.T) {
		// 发送无效的JSON
		invalidJSON := `{"invalid": json}`

		req := httptest.NewRequest("POST", "/api/v1/webhooks/git", bytes.NewBufferString(invalidJSON))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 应该返回400错误
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "无效的事件格式")
	})

	t.Run("处理缺少必要字段的事件", func(t *testing.T) {
		// 创建缺少必要字段的事件
		invalidEvent := webhook.GitEvent{
			EventType: "", // 缺少事件类型
			EventID:   uuid.New().String(),
			Timestamp: time.Now(),
		}

		eventData, err := json.Marshal(invalidEvent)
		assert.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/webhooks/git", bytes.NewBuffer(eventData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// 应该返回400错误
		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response["error"], "事件类型不能为空")
	})
}

// TestCallbackManager 测试回调管理器
func TestCallbackManager(t *testing.T) {
	logger := zap.NewNop()
	callbackManager := webhook.NewCallbackManager(logger)

	t.Run("创建项目事件", func(t *testing.T) {
		projectID := uuid.New()

		event := callbackManager.CreateProjectEvent(
			"project",
			"created",
			projectID,
			map[string]interface{}{
				"name":        "Test Project",
				"description": "A test project",
			},
			map[string]interface{}{
				"created_by": "test-user",
			},
		)

		assert.Equal(t, "project", event.EventType)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, projectID.String(), event.ProjectID)
		assert.Equal(t, "project-service", event.Source)
		assert.NotEmpty(t, event.ID)
		assert.NotEmpty(t, event.Resource)
		assert.NotEmpty(t, event.Metadata)
	})

	t.Run("创建仓库事件", func(t *testing.T) {
		projectID := uuid.New()
		repositoryID := uuid.New()

		event := callbackManager.CreateRepositoryEvent(
			"created",
			projectID,
			repositoryID,
			map[string]interface{}{
				"name":       "test-repo",
				"visibility": "private",
			},
			map[string]interface{}{
				"branch": "main",
			},
		)

		assert.Equal(t, "repository", event.EventType)
		assert.Equal(t, "created", event.Action)
		assert.Equal(t, projectID.String(), event.ProjectID)
		assert.Equal(t, "git-gateway", event.Source)
		assert.NotEmpty(t, event.ID)
		assert.NotEmpty(t, event.Resource)
	})

	t.Run("测试事件过滤器", func(t *testing.T) {
		config := &webhook.CallbackConfig{
			URL:       "http://test.example.com/webhook",
			EventMask: []string{"repository.created", "push"},
		}

		// 应该匹配的事件
		matchingEvent := &webhook.CallbackEvent{
			EventType: "repository",
			Action:    "created",
		}

		// 不应该匹配的事件
		nonMatchingEvent := &webhook.CallbackEvent{
			EventType: "repository",
			Action:    "deleted",
		}

		// 测试内部方法 - 这里我们需要测试逻辑
		// 由于shouldSendEvent是私有方法，我们通过公共接口测试

		// 创建一个测试服务器来验证回调
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer testServer.Close()

		config.URL = testServer.URL

		ctx := context.Background()

		// 测试匹配的事件 - 应该发送
		result, err := callbackManager.SendCallback(ctx, config, matchingEvent)
		assert.NoError(t, err)
		assert.True(t, result.Success)

		// 测试不匹配的事件 - 应该跳过（仍然返回成功）
		result, err = callbackManager.SendCallback(ctx, config, nonMatchingEvent)
		assert.NoError(t, err)
		assert.True(t, result.Success)
	})
}

// TestEventProcessor 测试事件处理器
func TestEventProcessor(t *testing.T) {
	// 设置测试数据库
	db := setupTestDB(t)

	logger := zap.NewNop()
	projectRepo := repository.NewProjectRepository(db)

	// 创建模拟Git网关客户端
	mockGitClient := new(MockGitGatewayClient)
	projectService := service.NewProjectService(projectRepo, mockGitClient, logger)

	eventProcessor := webhook.NewDefaultEventProcessor(projectRepo, projectService, logger)

	t.Run("处理仓库创建事件", func(t *testing.T) {
		projectID := uuid.New()
		gitEvent := &webhook.GitEvent{
			EventID:      uuid.New().String(),
			EventType:    "repository",
			ProjectID:    projectID.String(),
			RepositoryID: uuid.New().String(),
		}

		repoEvent := &webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:            gitEvent.RepositoryID,
				Name:          "test-repo",
				ProjectID:     projectID.String(),
				Visibility:    "private",
				DefaultBranch: "main",
			},
		}

		ctx := context.Background()
		err := eventProcessor.ProcessRepositoryEvent(ctx, gitEvent, repoEvent)

		// 处理应该成功（即使没有实际的项目存在，也应该记录活动）
		assert.NoError(t, err)
	})

	t.Run("处理推送事件", func(t *testing.T) {
		gitEvent := &webhook.GitEvent{
			EventID:      uuid.New().String(),
			EventType:    "push",
			ProjectID:    uuid.New().String(),
			RepositoryID: uuid.New().String(),
		}

		pushEvent := &webhook.PushEvent{
			RepositoryID: gitEvent.RepositoryID,
			Branch:       "main",
			Before:       "abc123",
			After:        "def456",
			Commits: []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
				Author  string `json:"author"`
			}{
				{
					SHA:     "def456",
					Message: "feat: add new feature",
					Author:  "developer@example.com",
				},
			},
			Pusher: "developer",
		}

		ctx := context.Background()
		err := eventProcessor.ProcessPushEvent(ctx, gitEvent, pushEvent)

		// 处理应该成功
		assert.NoError(t, err)
	})
}

// 辅助函数定义在 test_helpers.go 中
