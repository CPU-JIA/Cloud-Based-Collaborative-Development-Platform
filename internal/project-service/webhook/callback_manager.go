package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CallbackManager 回调管理器
type CallbackManager struct {
	httpClient *http.Client
	logger     *zap.Logger
}

// CallbackConfig 回调配置
type CallbackConfig struct {
	URL       string            `json:"url"`
	Secret    string            `json:"secret,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Timeout   time.Duration     `json:"timeout"`
	RetryMax  int               `json:"retry_max"`
	EventMask []string          `json:"event_mask,omitempty"` // 事件过滤器
}

// CallbackEvent 回调事件
type CallbackEvent struct {
	ID         string                 `json:"id"`
	EventType  string                 `json:"event_type"`
	Timestamp  time.Time              `json:"timestamp"`
	ProjectID  string                 `json:"project_id"`
	Source     string                 `json:"source"`      // 事件来源：git-gateway, project-service等
	Action     string                 `json:"action"`      // 具体动作：created, updated, deleted等
	Resource   map[string]interface{} `json:"resource"`    // 资源详情
	Metadata   map[string]interface{} `json:"metadata"`    // 额外元数据
	RetryCount int                    `json:"retry_count"` // 重试次数
}

// CallbackResult 回调结果
type CallbackResult struct {
	Success      bool          `json:"success"`
	StatusCode   int           `json:"status_code"`
	ResponseBody string        `json:"response_body"`
	Duration     time.Duration `json:"duration"`
	Error        string        `json:"error,omitempty"`
}

// NewCallbackManager 创建回调管理器
func NewCallbackManager(logger *zap.Logger) *CallbackManager {
	return &CallbackManager{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		logger: logger,
	}
}

// SendCallback 发送回调事件
func (cm *CallbackManager) SendCallback(ctx context.Context, config *CallbackConfig, event *CallbackEvent) (*CallbackResult, error) {
	startTime := time.Now()

	// 检查事件是否匹配过滤器
	if !cm.shouldSendEvent(config, event) {
		cm.logger.Debug("事件不匹配过滤器，跳过回调",
			zap.String("event_type", event.EventType),
			zap.String("action", event.Action),
			zap.Strings("event_mask", config.EventMask))
		return &CallbackResult{
			Success:  true,
			Duration: time.Since(startTime),
		}, nil
	}

	// 序列化事件
	eventData, err := json.Marshal(event)
	if err != nil {
		return nil, fmt.Errorf("序列化回调事件失败: %w", err)
	}

	// 创建HTTP请求
	req, err := http.NewRequestWithContext(ctx, "POST", config.URL, bytes.NewBuffer(eventData))
	if err != nil {
		return nil, fmt.Errorf("创建回调请求失败: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CloudPlatform-Webhook/1.0")
	req.Header.Set("X-Event-ID", event.ID)
	req.Header.Set("X-Event-Type", event.EventType)
	req.Header.Set("X-Event-Timestamp", event.Timestamp.Format(time.RFC3339))

	// 设置自定义请求头
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// 计算并设置签名
	if config.Secret != "" {
		signature := cm.calculateSignature(eventData, config.Secret)
		req.Header.Set("X-Hub-Signature-256", signature)
	}

	// 设置超时
	if config.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, config.Timeout)
		defer cancel()
		req = req.WithContext(ctx)
	}

	// 发送请求
	resp, err := cm.httpClient.Do(req)
	if err != nil {
		return &CallbackResult{
			Success:  false,
			Duration: time.Since(startTime),
			Error:    err.Error(),
		}, err
	}
	defer resp.Body.Close()

	// 读取响应
	var responseBody bytes.Buffer
	_, err = responseBody.ReadFrom(resp.Body)
	if err != nil {
		cm.logger.Warn("读取回调响应失败", zap.Error(err))
	}

	result := &CallbackResult{
		Success:      resp.StatusCode >= 200 && resp.StatusCode < 300,
		StatusCode:   resp.StatusCode,
		ResponseBody: responseBody.String(),
		Duration:     time.Since(startTime),
	}

	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d: %s", resp.StatusCode, responseBody.String())
	}

	cm.logger.Info("回调请求完成",
		zap.String("url", config.URL),
		zap.String("event_id", event.ID),
		zap.String("event_type", event.EventType),
		zap.Int("status_code", resp.StatusCode),
		zap.Duration("duration", result.Duration),
		zap.Bool("success", result.Success))

	return result, nil
}

// SendCallbackWithRetry 带重试的回调发送
func (cm *CallbackManager) SendCallbackWithRetry(ctx context.Context, config *CallbackConfig, event *CallbackEvent) (*CallbackResult, error) {
	maxRetries := config.RetryMax
	if maxRetries <= 0 {
		maxRetries = 3 // 默认重试3次
	}

	var lastResult *CallbackResult
	var lastError error

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 计算退避延迟（指数退避）
			delay := time.Duration(attempt*attempt) * time.Second
			if delay > 30*time.Second {
				delay = 30 * time.Second
			}

			cm.logger.Info("重试回调发送",
				zap.String("event_id", event.ID),
				zap.Int("attempt", attempt),
				zap.Duration("delay", delay))

			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				return lastResult, ctx.Err()
			case <-timer.C:
				timer.Stop()
			}

			// 更新重试次数
			event.RetryCount = attempt
		}

		result, err := cm.SendCallback(ctx, config, event)
		lastResult = result
		lastError = err

		// 如果成功或者不可重试的错误，直接返回
		if err == nil && result.Success {
			return result, nil
		}

		// 检查是否应该重试
		if !cm.shouldRetry(result, err) {
			break
		}

		cm.logger.Warn("回调发送失败，准备重试",
			zap.String("event_id", event.ID),
			zap.Int("attempt", attempt),
			zap.Error(err),
			zap.Any("result", result))
	}

	cm.logger.Error("回调发送最终失败",
		zap.String("event_id", event.ID),
		zap.Int("max_retries", maxRetries),
		zap.Error(lastError))

	return lastResult, lastError
}

// shouldSendEvent 检查是否应该发送事件
func (cm *CallbackManager) shouldSendEvent(config *CallbackConfig, event *CallbackEvent) bool {
	// 如果没有配置过滤器，发送所有事件
	if len(config.EventMask) == 0 {
		return true
	}

	// 检查事件类型是否在过滤器中
	eventPattern := fmt.Sprintf("%s.%s", event.EventType, event.Action)

	for _, mask := range config.EventMask {
		if mask == "*" || mask == event.EventType || mask == eventPattern {
			return true
		}

		// 支持通配符匹配，例如 "repository.*"
		if cm.matchPattern(mask, eventPattern) {
			return true
		}
	}

	return false
}

// matchPattern 简单的通配符匹配
func (cm *CallbackManager) matchPattern(pattern, text string) bool {
	if pattern == "*" {
		return true
	}

	// 简单的前缀匹配，支持 "repository.*" 这样的模式
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(text) >= len(prefix) && text[:len(prefix)] == prefix
	}

	return pattern == text
}

// shouldRetry 判断是否应该重试
func (cm *CallbackManager) shouldRetry(result *CallbackResult, err error) bool {
	// 网络错误或超时，应该重试
	if err != nil {
		return true
	}

	// HTTP状态码判断
	if result != nil {
		switch result.StatusCode {
		case 408, 429: // Request Timeout, Too Many Requests
			return true
		case 500, 502, 503, 504: // Server errors
			return true
		case 400, 401, 403, 404: // Client errors - 不重试
			return false
		}
	}

	return false
}

// calculateSignature 计算HMAC签名
func (cm *CallbackManager) calculateSignature(data []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(data)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// CreateProjectEvent 创建项目相关事件
func (cm *CallbackManager) CreateProjectEvent(eventType, action string, projectID uuid.UUID, resource interface{}, metadata map[string]interface{}) *CallbackEvent {
	return &CallbackEvent{
		ID:        uuid.New().String(),
		EventType: eventType,
		Timestamp: time.Now().UTC(),
		ProjectID: projectID.String(),
		Source:    "project-service",
		Action:    action,
		Resource:  cm.interfaceToMap(resource),
		Metadata:  metadata,
	}
}

// CreateRepositoryEvent 创建仓库相关事件
func (cm *CallbackManager) CreateRepositoryEvent(action string, projectID, repositoryID uuid.UUID, repository interface{}, metadata map[string]interface{}) *CallbackEvent {
	return &CallbackEvent{
		ID:        uuid.New().String(),
		EventType: "repository",
		Timestamp: time.Now().UTC(),
		ProjectID: projectID.String(),
		Source:    "git-gateway",
		Action:    action,
		Resource:  cm.interfaceToMap(repository),
		Metadata:  metadata,
	}
}

// interfaceToMap 将interface{}转换为map[string]interface{}
func (cm *CallbackManager) interfaceToMap(data interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	// 尝试类型断言
	if m, ok := data.(map[string]interface{}); ok {
		return m
	}

	// 通过JSON序列化/反序列化转换
	jsonData, err := json.Marshal(data)
	if err != nil {
		cm.logger.Warn("转换数据为map失败", zap.Error(err))
		return map[string]interface{}{"raw": data}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(jsonData, &result); err != nil {
		cm.logger.Warn("JSON反序列化失败", zap.Error(err))
		return map[string]interface{}{"raw": string(jsonData)}
	}

	return result
}
