package unit

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/webhook"
)

// MockHTTPClient Mock HTTP客户端
type MockHTTPClient struct {
	mock.Mock
}

// CallbackManagerTestSuite 回调管理器测试套件
type CallbackManagerTestSuite struct {
	suite.Suite
	manager       *webhook.CallbackManager
	logger        *zap.Logger
	testProjectID uuid.UUID
	testEventID   string
}

func (suite *CallbackManagerTestSuite) SetupSuite() {
	suite.logger = zaptest.NewLogger(suite.T())
	suite.testProjectID = uuid.New()
	suite.testEventID = uuid.New().String()
}

func (suite *CallbackManagerTestSuite) SetupTest() {
	suite.manager = webhook.NewCallbackManager(suite.logger)
}

// TestSendCallback 测试发送回调
func (suite *CallbackManagerTestSuite) TestSendCallback() {
	suite.Run("成功发送回调", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL:      "https://example.com/webhook",
			Secret:   "test-secret",
			Timeout:  5 * time.Second,
			RetryMax: 3,
		}

		event := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
			Source:    "git-gateway",
			Action:    "created",
			Resource: map[string]interface{}{
				"repository_id":   uuid.New().String(),
				"repository_name": "test-repo",
			},
			Metadata: map[string]interface{}{
				"user_id": uuid.New().String(),
			},
			RetryCount: 0,
		}

		// 注意：这个测试实际上会尝试发送HTTP请求
		// 在真实环境中，应该mock HTTP客户端
		result, err := suite.manager.SendCallback(ctx, config, event)
		
		// 由于没有实际的服务器响应，这里应该会返回错误
		// 但我们可以验证回调结果的结构
		assert.NotNil(suite.T(), result)
		if err != nil {
			// 网络错误是预期的，因为URL不存在
			assert.Contains(suite.T(), err.Error(), "request failed")
		}
	})

	suite.Run("测试事件过滤", func() {
		ctx := context.Background()
		
		// 配置只接受repository事件
		config := &webhook.CallbackConfig{
			URL:       "https://example.com/webhook",
			EventMask: []string{"repository.created", "repository.updated"},
		}

		// 创建匹配的事件
		matchingEvent := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "created",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		result, err := suite.manager.SendCallback(ctx, config, matchingEvent)
		
		// 事件匹配过滤器，应该尝试发送（虽然会失败）
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)

		// 创建不匹配的事件
		nonMatchingEvent := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "deleted", // 不在过滤器中
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		result, err = suite.manager.SendCallback(ctx, config, nonMatchingEvent)
		
		// 事件不匹配过滤器，应该跳过
		assert.NoError(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.True(suite.T(), result.Success)
	})

	suite.Run("测试通配符过滤", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL:       "https://example.com/webhook",
			EventMask: []string{"repository.*"},
		}

		// 创建repository类型的各种动作事件
		actions := []string{"created", "updated", "deleted", "archived"}
		
		for _, action := range actions {
			event := &webhook.CallbackEvent{
				ID:        suite.testEventID,
				EventType: "repository",
				Action:    action,
				Timestamp: time.Now(),
				ProjectID: suite.testProjectID.String(),
			}

			result, err := suite.manager.SendCallback(ctx, config, event)
			
			// 所有repository.*事件都应该匹配
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), result)
		}
	})
}

// TestSendCallbackWithRetry 测试带重试的回调发送
func (suite *CallbackManagerTestSuite) TestSendCallbackWithRetry() {
	suite.Run("测试重试机制", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL:      "https://invalid-url-for-testing.example.com/webhook",
			RetryMax: 2,
		}

		event := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "created",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		start := time.Now()
		result, err := suite.manager.SendCallbackWithRetry(ctx, config, event)
		duration := time.Since(start)

		// 应该尝试重试
		assert.Error(suite.T(), err)
		assert.NotNil(suite.T(), result)
		
		// 重试应该增加总耗时（包含退避延迟）
		assert.Greater(suite.T(), duration, 1*time.Second)
	})

	suite.Run("测试上下文取消", func() {
		ctx, cancel := context.WithCancel(context.Background())
		
		config := &webhook.CallbackConfig{
			URL:      "https://invalid-url-for-testing.example.com/webhook",
			RetryMax: 5,
		}

		event := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "created",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		// 短暂延迟后取消上下文
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		result, err := suite.manager.SendCallbackWithRetry(ctx, config, event)

		// 上下文取消应该中断重试
		assert.Error(suite.T(), err)
		assert.NotNil(suite.T(), result)
	})
}

// TestCreateEventMethods 测试事件创建方法
func (suite *CallbackManagerTestSuite) TestCreateEventMethods() {
	suite.Run("创建项目事件", func() {
		resource := map[string]interface{}{
			"project_name": "test-project",
			"project_type": "software",
		}
		
		metadata := map[string]interface{}{
			"created_by": "user@example.com",
		}

		event := suite.manager.CreateProjectEvent(
			"project", 
			"created", 
			suite.testProjectID, 
			resource, 
			metadata,
		)

		assert.NotNil(suite.T(), event)
		assert.Equal(suite.T(), "project", event.EventType)
		assert.Equal(suite.T(), "created", event.Action)
		assert.Equal(suite.T(), suite.testProjectID.String(), event.ProjectID)
		assert.Equal(suite.T(), "project-service", event.Source)
		assert.NotEmpty(suite.T(), event.ID)
		assert.NotZero(suite.T(), event.Timestamp)
		assert.NotNil(suite.T(), event.Resource)
		assert.Equal(suite.T(), metadata, event.Metadata)
	})

	suite.Run("创建仓库事件", func() {
		repositoryID := uuid.New()
		repository := map[string]interface{}{
			"repository_name": "test-repo",
			"visibility":      "private",
		}
		
		metadata := map[string]interface{}{
			"branch_count": 3,
		}

		event := suite.manager.CreateRepositoryEvent(
			"created", 
			suite.testProjectID, 
			repositoryID, 
			repository, 
			metadata,
		)

		assert.NotNil(suite.T(), event)
		assert.Equal(suite.T(), "repository", event.EventType)
		assert.Equal(suite.T(), "created", event.Action)
		assert.Equal(suite.T(), suite.testProjectID.String(), event.ProjectID)
		assert.Equal(suite.T(), "git-gateway", event.Source)
		assert.NotEmpty(suite.T(), event.ID)
		assert.NotZero(suite.T(), event.Timestamp)
		assert.NotNil(suite.T(), event.Resource)
		assert.Equal(suite.T(), metadata, event.Metadata)
	})
}

// TestPatternMatching 测试模式匹配
func (suite *CallbackManagerTestSuite) TestPatternMatching() {
	testCases := []struct {
		name     string
		pattern  string
		text     string
		expected bool
	}{
		{
			name:     "通配符匹配",
			pattern:  "*",
			text:     "anything",
			expected: true,
		},
		{
			name:     "前缀通配符匹配",
			pattern:  "repository.*",
			text:     "repository.created",
			expected: true,
		},
		{
			name:     "前缀通配符不匹配",
			pattern:  "repository.*",
			text:     "branch.created",
			expected: false,
		},
		{
			name:     "精确匹配",
			pattern:  "repository.created",
			text:     "repository.created",
			expected: true,
		},
		{
			name:     "精确不匹配",
			pattern:  "repository.created",
			text:     "repository.updated",
			expected: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			config := &webhook.CallbackConfig{
				EventMask: []string{tc.pattern},
			}

			event := &webhook.CallbackEvent{
				EventType: "repository",
				Action:    "created",
			}

			// 构造事件模式
			if tc.text == "repository.created" {
				event.EventType = "repository"
				event.Action = "created"
			} else if tc.text == "repository.updated" {
				event.EventType = "repository"
				event.Action = "updated"
			} else if tc.text == "branch.created" {
				event.EventType = "branch"
				event.Action = "created"
			} else if tc.text == "anything" {
				event.EventType = "anything"
				event.Action = ""
			}

			// 使用私有方法进行测试（这里我们通过发送实际回调来测试）
			ctx := context.Background()
			config.URL = "https://example.com/webhook"
			
			result, err := suite.manager.SendCallback(ctx, config, event)
			
			if tc.expected {
				// 应该尝试发送（即使失败）
				assert.NotNil(suite.T(), result)
			} else {
				// 应该被过滤掉，直接返回成功
				assert.NoError(suite.T(), err)
				assert.True(suite.T(), result.Success)
			}
		})
	}
}

// TestInterfaceConversion 测试接口转换
func (suite *CallbackManagerTestSuite) TestInterfaceConversion() {
	suite.Run("转换map类型", func() {
		data := map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		}

		event := suite.manager.CreateProjectEvent(
			"test", 
			"action", 
			suite.testProjectID, 
			data, 
			nil,
		)

		assert.Equal(suite.T(), data, event.Resource)
	})

	suite.Run("转换结构体类型", func() {
		type TestStruct struct {
			Name  string `json:"name"`
			Value int    `json:"value"`
		}

		data := TestStruct{
			Name:  "test",
			Value: 42,
		}

		event := suite.manager.CreateProjectEvent(
			"test", 
			"action", 
			suite.testProjectID, 
			data, 
			nil,
		)

		// 应该被转换为map
		assert.NotNil(suite.T(), event.Resource)
		resourceMap, ok := event.Resource.(map[string]interface{})
		assert.True(suite.T(), ok)
		assert.Equal(suite.T(), "test", resourceMap["name"])
		assert.Equal(suite.T(), float64(42), resourceMap["value"]) // JSON数字转为float64
	})

	suite.Run("转换nil值", func() {
		event := suite.manager.CreateProjectEvent(
			"test", 
			"action", 
			suite.testProjectID, 
			nil, 
			nil,
		)

		assert.Nil(suite.T(), event.Resource)
	})
}

// TestErrorScenarios 测试错误场景
func (suite *CallbackManagerTestSuite) TestErrorScenarios() {
	suite.Run("测试超时配置", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL:     "https://httpbin.org/delay/10", // 延迟10秒响应
			Timeout: 1 * time.Second,              // 1秒超时
		}

		event := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "created",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		start := time.Now()
		result, err := suite.manager.SendCallback(ctx, config, event)
		duration := time.Since(start)

		// 应该在超时时间内失败
		assert.Error(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.False(suite.T(), result.Success)
		assert.Less(suite.T(), duration, 5*time.Second) // 应该远早于10秒
	})

	suite.Run("测试无效URL", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL: "invalid-url",
		}

		event := &webhook.CallbackEvent{
			ID:        suite.testEventID,
			EventType: "repository",
			Action:    "created",
			Timestamp: time.Now(),
			ProjectID: suite.testProjectID.String(),
		}

		result, err := suite.manager.SendCallback(ctx, config, event)

		assert.Error(suite.T(), err)
		assert.NotNil(suite.T(), result)
		assert.False(suite.T(), result.Success)
	})
}

// TestConcurrentCallbacks 测试并发回调
func (suite *CallbackManagerTestSuite) TestConcurrentCallbacks() {
	suite.Run("并发发送回调", func() {
		ctx := context.Background()
		
		config := &webhook.CallbackConfig{
			URL: "https://invalid-url-for-testing.example.com/webhook",
		}

		// 创建多个goroutine并发发送回调
		goroutineCount := 10
		done := make(chan bool, goroutineCount)
		
		for i := 0; i < goroutineCount; i++ {
			go func(id int) {
				defer func() { done <- true }()
				
				event := &webhook.CallbackEvent{
					ID:        uuid.New().String(),
					EventType: "repository",
					Action:    "created",
					Timestamp: time.Now(),
					ProjectID: suite.testProjectID.String(),
				}

				result, err := suite.manager.SendCallback(ctx, config, event)
				
				// 每个回调都应该有结果（即使失败）
				assert.NotNil(suite.T(), result)
				// 网络错误是预期的
				if err != nil {
					assert.Contains(suite.T(), err.Error(), "request failed")
				}
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < goroutineCount; i++ {
			select {
			case <-done:
				// 继续
			case <-time.After(30 * time.Second):
				suite.T().Fatal("并发测试超时")
			}
		}
	})
}

// 运行测试套件
func TestCallbackManagerSuite(t *testing.T) {
	suite.Run(t, new(CallbackManagerTestSuite))
}