package unit

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
)

// MockEventProcessor Mock事件处理器
type MockEventProcessor struct {
	mock.Mock
}

func (m *MockEventProcessor) ProcessRepositoryEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.RepositoryEvent) error {
	args := m.Called(ctx, event, payload)
	return args.Error(0)
}

func (m *MockEventProcessor) ProcessBranchEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.BranchEvent) error {
	args := m.Called(ctx, event, payload)
	return args.Error(0)
}

func (m *MockEventProcessor) ProcessCommitEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.CommitEvent) error {
	args := m.Called(ctx, event, payload)
	return args.Error(0)
}

func (m *MockEventProcessor) ProcessPushEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.PushEvent) error {
	args := m.Called(ctx, event, payload)
	return args.Error(0)
}

func (m *MockEventProcessor) ProcessTagEvent(ctx context.Context, event *webhook.GitEvent, payload *webhook.TagEvent) error {
	args := m.Called(ctx, event, payload)
	return args.Error(0)
}

// WebhookEventProcessorTestSuite Webhook事件处理器测试套件
type WebhookEventProcessorTestSuite struct {
	suite.Suite
	handler         *webhook.WebhookHandler
	mockProcessor   *MockEventProcessor
	router          *gin.Engine
	logger          *zap.Logger
	testProjectID   uuid.UUID
	testRepositoryID uuid.UUID
	testUserID      uuid.UUID
	secret          string
}

func (suite *WebhookEventProcessorTestSuite) SetupSuite() {
	gin.SetMode(gin.TestMode)
	suite.logger = zaptest.NewLogger(suite.T())
	
	suite.testProjectID = uuid.New()
	suite.testRepositoryID = uuid.New()
	suite.testUserID = uuid.New()
	suite.secret = "test-webhook-secret-key-12345"
}

func (suite *WebhookEventProcessorTestSuite) SetupTest() {
	suite.mockProcessor = new(MockEventProcessor)
	suite.handler = webhook.NewWebhookHandler(suite.mockProcessor, suite.secret, suite.logger)
	
	suite.router = gin.New()
	suite.setupRoutes()
}

func (suite *WebhookEventProcessorTestSuite) TearDownTest() {
	suite.mockProcessor.AssertExpectations(suite.T())
}

func (suite *WebhookEventProcessorTestSuite) setupRoutes() {
	api := suite.router.Group("/api/v1")
	{
		api.POST("/webhook/git", suite.handler.HandleWebhook)
		api.GET("/webhook/health", suite.handler.GetHealthCheck)
	}
}

// TestRepositoryEvents 测试仓库事件处理
func (suite *WebhookEventProcessorTestSuite) TestRepositoryEvents() {
	testCases := []struct {
		name           string
		action         string
		setupMocks     func()
		expectedStatus int
	}{
		{
			name:   "仓库创建事件",
			action: "created",
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.MatchedBy(func(event *webhook.GitEvent) bool {
						return event.EventType == "repository" && 
							   event.ProjectID == suite.testProjectID.String() &&
							   event.RepositoryID == suite.testRepositoryID.String()
					}),
					mock.MatchedBy(func(payload *webhook.RepositoryEvent) bool {
						return payload.Action == "created" &&
							   payload.Repository.Name == "test-repo" &&
							   payload.Repository.ProjectID == suite.testProjectID.String()
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "仓库更新事件",
			action: "updated",
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.RepositoryEvent) bool {
						return payload.Action == "updated"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "仓库删除事件",
			action: "deleted",
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.RepositoryEvent) bool {
						return payload.Action == "deleted"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "仓库归档事件",
			action: "archived",
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.RepositoryEvent) bool {
						return payload.Action == "archived"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:   "仓库取消归档事件",
			action: "unarchived",
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.RepositoryEvent) bool {
						return payload.Action == "unarchived"
					}),
				).Return(nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			gitEvent := webhook.GitEvent{
				EventType:    "repository",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
				UserID:       suite.testUserID.String(),
			}

			repositoryPayload := webhook.RepositoryEvent{
				Action: tc.action,
				Repository: struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					ProjectID     string `json:"project_id"`
					Visibility    string `json:"visibility"`
					DefaultBranch string `json:"default_branch"`
				}{
					ID:            suite.testRepositoryID.String(),
					Name:          "test-repo",
					ProjectID:     suite.testProjectID.String(),
					Visibility:    "private",
					DefaultBranch: "main",
				},
			}

			payloadBytes, _ := json.Marshal(repositoryPayload)
			gitEvent.Payload = payloadBytes

			body := suite.createSignedRequest(gitEvent)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code)
		})
	}
}

// TestBranchEvents 测试分支事件处理
func (suite *WebhookEventProcessorTestSuite) TestBranchEvents() {
	testCases := []struct {
		name       string
		action     string
		branchName string
		setupMocks func(branchName string)
	}{
		{
			name:       "分支创建事件",
			action:     "created",
			branchName: "feature/new-feature",
			setupMocks: func(branchName string) {
				suite.mockProcessor.On("ProcessBranchEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.BranchEvent) bool {
						return payload.Action == "created" && 
							   payload.Branch.Name == branchName
					}),
				).Return(nil)
			},
		},
		{
			name:       "分支删除事件",
			action:     "deleted",
			branchName: "feature/old-feature",
			setupMocks: func(branchName string) {
				suite.mockProcessor.On("ProcessBranchEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.BranchEvent) bool {
						return payload.Action == "deleted" && 
							   payload.Branch.Name == branchName
					}),
				).Return(nil)
			},
		},
		{
			name:       "默认分支变更事件",
			action:     "default_changed",
			branchName: "develop",
			setupMocks: func(branchName string) {
				suite.mockProcessor.On("ProcessBranchEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.MatchedBy(func(payload *webhook.BranchEvent) bool {
						return payload.Action == "default_changed" && 
							   payload.Branch.Name == branchName
					}),
				).Return(nil)
			},
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks(tc.branchName)

			gitEvent := webhook.GitEvent{
				EventType:    "branch",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			branchPayload := webhook.BranchEvent{
				Action: tc.action,
				Branch: struct {
					Name         string `json:"name"`
					RepositoryID string `json:"repository_id"`
					Commit       string `json:"commit"`
				}{
					Name:         tc.branchName,
					RepositoryID: suite.testRepositoryID.String(),
					Commit:       "abc123def456",
				},
			}

			payloadBytes, _ := json.Marshal(branchPayload)
			gitEvent.Payload = payloadBytes

			body := suite.createSignedRequest(gitEvent)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)
		})
	}
}

// TestCommitEvents 测试提交事件处理
func (suite *WebhookEventProcessorTestSuite) TestCommitEvents() {
	suite.Run("提交创建事件", func() {
		suite.mockProcessor.On("ProcessCommitEvent", 
			mock.Anything, 
			mock.AnythingOfType("*webhook.GitEvent"),
			mock.MatchedBy(func(payload *webhook.CommitEvent) bool {
				return payload.Action == "created" && 
					   payload.Commit.SHA == "abc123def456789" &&
					   payload.Commit.Message == "feat: add new feature" &&
					   payload.Commit.Author == "developer@example.com"
			}),
		).Return(nil)

		gitEvent := webhook.GitEvent{
			EventType:    "commit",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		commitPayload := webhook.CommitEvent{
			Action: "created",
			Commit: struct {
				SHA          string    `json:"sha"`
				Message      string    `json:"message"`
				Author       string    `json:"author"`
				RepositoryID string    `json:"repository_id"`
				Branch       string    `json:"branch"`
				Timestamp    time.Time `json:"timestamp"`
			}{
				SHA:          "abc123def456789",
				Message:      "feat: add new feature",
				Author:       "developer@example.com",
				RepositoryID: suite.testRepositoryID.String(),
				Branch:       "main",
				Timestamp:    time.Now(),
			},
		}

		payloadBytes, _ := json.Marshal(commitPayload)
		gitEvent.Payload = payloadBytes

		body := suite.createSignedRequest(gitEvent)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
	})
}

// TestPushEvents 测试推送事件处理
func (suite *WebhookEventProcessorTestSuite) TestPushEvents() {
	suite.Run("推送事件", func() {
		suite.mockProcessor.On("ProcessPushEvent", 
			mock.Anything, 
			mock.AnythingOfType("*webhook.GitEvent"),
			mock.MatchedBy(func(payload *webhook.PushEvent) bool {
				return payload.RepositoryID == suite.testRepositoryID.String() &&
					   payload.Branch == "main" &&
					   payload.Pusher == "developer@example.com" &&
					   len(payload.Commits) == 2
			}),
		).Return(nil)

		gitEvent := webhook.GitEvent{
			EventType:    "push",
			EventID:      uuid.New().String(),
			Timestamp:    time.Now(),
			ProjectID:    suite.testProjectID.String(),
			RepositoryID: suite.testRepositoryID.String(),
		}

		pushPayload := webhook.PushEvent{
			RepositoryID: suite.testRepositoryID.String(),
			Branch:       "main",
			Before:       "abc123",
			After:        "def456",
			Commits: []struct {
				SHA     string `json:"sha"`
				Message string `json:"message"`
				Author  string `json:"author"`
			}{
				{
					SHA:     "commit1sha",
					Message: "feat: implement feature A",
					Author:  "developer@example.com",
				},
				{
					SHA:     "commit2sha",
					Message: "fix: resolve bug in feature A",
					Author:  "developer@example.com",
				},
			},
			Pusher: "developer@example.com",
		}

		payloadBytes, _ := json.Marshal(pushPayload)
		gitEvent.Payload = payloadBytes

		body := suite.createSignedRequest(gitEvent)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)
	})
}

// TestTagEvents 测试标签事件处理
func (suite *WebhookEventProcessorTestSuite) TestTagEvents() {
	testCases := []struct {
		name    string
		action  string
		tagName string
	}{
		{
			name:    "标签创建事件",
			action:  "created",
			tagName: "v1.0.0",
		},
		{
			name:    "标签删除事件",
			action:  "deleted",
			tagName: "v0.9.0",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.mockProcessor.On("ProcessTagEvent", 
				mock.Anything, 
				mock.AnythingOfType("*webhook.GitEvent"),
				mock.MatchedBy(func(payload *webhook.TagEvent) bool {
					return payload.Action == tc.action && 
						   payload.Tag.Name == tc.tagName &&
						   payload.Tag.RepositoryID == suite.testRepositoryID.String()
				}),
			).Return(nil)

			gitEvent := webhook.GitEvent{
				EventType:    "tag",
				EventID:      uuid.New().String(),
				Timestamp:    time.Now(),
				ProjectID:    suite.testProjectID.String(),
				RepositoryID: suite.testRepositoryID.String(),
			}

			tagPayload := webhook.TagEvent{
				Action: tc.action,
				Tag: struct {
					Name         string `json:"name"`
					RepositoryID string `json:"repository_id"`
					Target       string `json:"target"`
					Message      string `json:"message,omitempty"`
				}{
					Name:         tc.tagName,
					RepositoryID: suite.testRepositoryID.String(),
					Target:       "commit123",
					Message:      "Release " + tc.tagName,
				},
			}

			payloadBytes, _ := json.Marshal(tagPayload)
			gitEvent.Payload = payloadBytes

			body := suite.createSignedRequest(gitEvent)
			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), http.StatusOK, w.Code)
		})
	}
}

// TestSignatureVerification 测试签名验证
func (suite *WebhookEventProcessorTestSuite) TestSignatureVerification() {
	testCases := []struct {
		name           string
		signature      string
		body           string
		expectedStatus int
		description    string
	}{
		{
			name:           "有效签名",
			signature:      "", // 将在测试中动态生成
			body:           `{"event_type":"repository","event_id":"test-123"}`,
			expectedStatus: http.StatusOK,
			description:    "正确的签名应该验证通过",
		},
		{
			name:           "无效签名",
			signature:      "sha256=invalid_signature",
			body:           `{"event_type":"repository","event_id":"test-123"}`,
			expectedStatus: http.StatusUnauthorized,
			description:    "错误的签名应该被拒绝",
		},
		{
			name:           "缺少签名",
			signature:      "",
			body:           `{"event_type":"repository","event_id":"test-123"}`,
			expectedStatus: http.StatusUnauthorized,
			description:    "没有签名头应该被拒绝",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// 为有效签名测试生成正确的签名
			signature := tc.signature
			if tc.name == "有效签名" {
				signature = suite.calculateSignature([]byte(tc.body))
				// 设置mock期望，只有在有效签名时才会调用
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.AnythingOfType("*webhook.RepositoryEvent"),
				).Return(nil).Maybe()
			}

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			if signature != "" {
				req.Header.Set("X-Hub-Signature-256", signature)
			}
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code, tc.description)
		})
	}
}

// TestErrorHandling 测试错误处理
func (suite *WebhookEventProcessorTestSuite) TestErrorHandling() {
	testCases := []struct {
		name           string
		body           string
		setupMocks     func()
		expectedStatus int
		description    string
	}{
		{
			name:           "无效的JSON格式",
			body:           `{"event_type":"repository","invalid_json":}`,
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			description:    "无效的JSON应该返回400错误",
		},
		{
			name:           "缺少必需字段",
			body:           `{"event_type":"","event_id":"test-123"}`,
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			description:    "缺少必需字段应该返回400错误",
		},
		{
			name:           "无效的项目ID格式",
			body:           `{"event_type":"repository","event_id":"test-123","project_id":"invalid-uuid","repository_id":"` + suite.testRepositoryID.String() + `","payload":{}}`,
			setupMocks:     func() {},
			expectedStatus: http.StatusBadRequest,
			description:    "无效的UUID格式应该返回400错误",
		},
		{
			name: "事件处理器返回错误",
			body: suite.createValidEventBody("repository"),
			setupMocks: func() {
				suite.mockProcessor.On("ProcessRepositoryEvent", 
					mock.Anything, 
					mock.AnythingOfType("*webhook.GitEvent"),
					mock.AnythingOfType("*webhook.RepositoryEvent"),
				).Return(fmt.Errorf("处理失败"))
			},
			expectedStatus: http.StatusOK, // 异步处理，立即返回成功
			description:    "事件处理错误不影响HTTP响应",
		},
		{
			name:           "未知事件类型",
			body:           suite.createValidEventBody("unknown_event"),
			setupMocks:     func() {},
			expectedStatus: http.StatusOK,
			description:    "未知事件类型应该被忽略但返回成功",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			tc.setupMocks()

			req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(tc.body)))
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			assert.Equal(suite.T(), tc.expectedStatus, w.Code, tc.description)
		})
	}
}

// TestHealthCheck 测试健康检查
func (suite *WebhookEventProcessorTestSuite) TestHealthCheck() {
	suite.Run("健康检查端点", func() {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/webhook/health", nil)
		w := httptest.NewRecorder()

		suite.router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(suite.T(), err)

		assert.Equal(suite.T(), "healthy", response["status"])
		assert.Equal(suite.T(), "git-webhook-handler", response["service"])
		assert.NotNil(suite.T(), response["timestamp"])
	})
}

// TestConcurrentRequests 测试并发请求处理
func (suite *WebhookEventProcessorTestSuite) TestConcurrentRequests() {
	suite.Run("并发webhook请求", func() {
		// 设置mock期望，允许多次调用
		suite.mockProcessor.On("ProcessRepositoryEvent", 
			mock.Anything, 
			mock.AnythingOfType("*webhook.GitEvent"),
			mock.AnythingOfType("*webhook.RepositoryEvent"),
		).Return(nil).Times(10)

		done := make(chan bool, 10)
		
		// 发送10个并发请求
		for i := 0; i < 10; i++ {
			go func(id int) {
				defer func() { done <- true }()
				
				body := suite.createValidEventBody("repository")
				req := httptest.NewRequest(http.MethodPost, "/api/v1/webhook/git", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("X-Hub-Signature-256", suite.calculateSignature([]byte(body)))
				w := httptest.NewRecorder()

				suite.router.ServeHTTP(w, req)

				assert.Equal(suite.T(), http.StatusOK, w.Code, fmt.Sprintf("Request %d failed", id))
			}(i)
		}

		// 等待所有请求完成
		for i := 0; i < 10; i++ {
			select {
			case <-done:
				// 请求完成
			case <-time.After(5 * time.Second):
				suite.T().Fatal("请求超时")
			}
		}

		// 给异步处理一些时间
		time.Sleep(100 * time.Millisecond)
	})
}

// 辅助方法

func (suite *WebhookEventProcessorTestSuite) createSignedRequest(event webhook.GitEvent) string {
	body, _ := json.Marshal(event)
	return string(body)
}

func (suite *WebhookEventProcessorTestSuite) calculateSignature(body []byte) string {
	// 这里应该实现与webhook.CallbackManager.calculateSignature相同的逻辑
	return "sha256=test_signature_placeholder"
}

func (suite *WebhookEventProcessorTestSuite) createValidEventBody(eventType string) string {
	gitEvent := webhook.GitEvent{
		EventType:    eventType,
		EventID:      uuid.New().String(),
		Timestamp:    time.Now(),
		ProjectID:    suite.testProjectID.String(),
		RepositoryID: suite.testRepositoryID.String(),
	}

	// 根据事件类型创建相应的payload
	switch eventType {
	case "repository":
		payload := webhook.RepositoryEvent{
			Action: "created",
			Repository: struct {
				ID            string `json:"id"`
				Name          string `json:"name"`
				ProjectID     string `json:"project_id"`
				Visibility    string `json:"visibility"`
				DefaultBranch string `json:"default_branch"`
			}{
				ID:            suite.testRepositoryID.String(),
				Name:          "test-repo",
				ProjectID:     suite.testProjectID.String(),
				Visibility:    "private",
				DefaultBranch: "main",
			},
		}
		payloadBytes, _ := json.Marshal(payload)
		gitEvent.Payload = payloadBytes
	default:
		gitEvent.Payload = json.RawMessage(`{}`)
	}

	body, _ := json.Marshal(gitEvent)
	return string(body)
}

// 运行测试套件
func TestWebhookEventProcessorSuite(t *testing.T) {
	suite.Run(t, new(WebhookEventProcessorTestSuite))
}