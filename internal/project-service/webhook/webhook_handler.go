package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebhookHandler Git事件webhook处理器
type WebhookHandler struct {
	eventProcessor EventProcessor
	secret         string
	logger         *zap.Logger
}

// GitEvent Git事件基础结构
type GitEvent struct {
	EventType    string    `json:"event_type"`
	EventID      string    `json:"event_id"`
	Timestamp    time.Time `json:"timestamp"`
	ProjectID    string    `json:"project_id"`
	RepositoryID string    `json:"repository_id"`
	UserID       string    `json:"user_id,omitempty"`
	Payload      json.RawMessage `json:"payload"`
}

// RepositoryEvent 仓库事件
type RepositoryEvent struct {
	Action     string `json:"action"`
	Repository struct {
		ID            string `json:"id"`
		Name          string `json:"name"`
		ProjectID     string `json:"project_id"`
		Visibility    string `json:"visibility"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
}

// BranchEvent 分支事件
type BranchEvent struct {
	Action string `json:"action"`
	Branch struct {
		Name         string `json:"name"`
		RepositoryID string `json:"repository_id"`
		Commit       string `json:"commit"`
	} `json:"branch"`
}

// CommitEvent 提交事件
type CommitEvent struct {
	Action string `json:"action"`
	Commit struct {
		SHA          string    `json:"sha"`
		Message      string    `json:"message"`
		Author       string    `json:"author"`
		RepositoryID string    `json:"repository_id"`
		Branch       string    `json:"branch"`
		Timestamp    time.Time `json:"timestamp"`
	} `json:"commit"`
}

// PushEvent 推送事件
type PushEvent struct {
	RepositoryID string `json:"repository_id"`
	Branch       string `json:"branch"`
	Before       string `json:"before"`
	After        string `json:"after"`
	Commits      []struct {
		SHA     string `json:"sha"`
		Message string `json:"message"`
		Author  string `json:"author"`
	} `json:"commits"`
	Pusher string `json:"pusher"`
}

// TagEvent 标签事件
type TagEvent struct {
	Action string `json:"action"`
	Tag    struct {
		Name         string `json:"name"`
		RepositoryID string `json:"repository_id"`
		Target       string `json:"target"`
		Message      string `json:"message,omitempty"`
	} `json:"tag"`
}

// EventProcessor 事件处理器接口
type EventProcessor interface {
	ProcessRepositoryEvent(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error
	ProcessBranchEvent(ctx context.Context, event *GitEvent, payload *BranchEvent) error
	ProcessCommitEvent(ctx context.Context, event *GitEvent, payload *CommitEvent) error
	ProcessPushEvent(ctx context.Context, event *GitEvent, payload *PushEvent) error
	ProcessTagEvent(ctx context.Context, event *GitEvent, payload *TagEvent) error
}

// NewWebhookHandler 创建webhook处理器
func NewWebhookHandler(processor EventProcessor, secret string, logger *zap.Logger) *WebhookHandler {
	return &WebhookHandler{
		eventProcessor: processor,
		secret:         secret,
		logger:         logger,
	}
}

// HandleWebhook 处理Git webhook事件
func (h *WebhookHandler) HandleWebhook(c *gin.Context) {
	// 验证webhook签名
	if !h.verifySignature(c) {
		h.logger.Warn("无效的webhook签名",
			zap.String("remote_addr", c.ClientIP()),
			zap.String("user_agent", c.GetHeader("User-Agent")))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "无效的签名"})
		return
	}

	// 读取请求体
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("读取webhook请求体失败", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无法读取请求体"})
		return
	}

	// 解析Git事件
	var gitEvent GitEvent
	if err := json.Unmarshal(body, &gitEvent); err != nil {
		h.logger.Error("解析Git事件失败", 
			zap.Error(err),
			zap.String("body", string(body)))
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的事件格式"})
		return
	}

	// 验证事件基本信息
	if err := h.validateEvent(&gitEvent); err != nil {
		h.logger.Error("事件验证失败", 
			zap.Error(err),
			zap.String("event_id", gitEvent.EventID),
			zap.String("event_type", gitEvent.EventType))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("接收到Git事件",
		zap.String("event_id", gitEvent.EventID),
		zap.String("event_type", gitEvent.EventType),
		zap.String("project_id", gitEvent.ProjectID),
		zap.String("repository_id", gitEvent.RepositoryID))

	// 异步处理事件
	go h.processEventAsync(&gitEvent)

	// 立即返回成功响应
	c.JSON(http.StatusOK, gin.H{
		"message":  "事件已接收",
		"event_id": gitEvent.EventID,
	})
}

// verifySignature 验证webhook签名
func (h *WebhookHandler) verifySignature(c *gin.Context) bool {
	if h.secret == "" {
		return true // 如果未配置secret，跳过验证
	}

	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		return false
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return false
	}

	// 重置request body以便后续读取
	c.Request.Body = io.NopCloser(strings.NewReader(string(body)))

	// 计算期望的签名
	mac := hmac.New(sha256.New, []byte(h.secret))
	mac.Write(body)
	expectedSignature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// validateEvent 验证事件基本信息
func (h *WebhookHandler) validateEvent(event *GitEvent) error {
	if event.EventType == "" {
		return fmt.Errorf("事件类型不能为空")
	}

	if event.EventID == "" {
		return fmt.Errorf("事件ID不能为空")
	}

	if event.ProjectID == "" {
		return fmt.Errorf("项目ID不能为空")
	}

	// 验证UUID格式
	if _, err := uuid.Parse(event.ProjectID); err != nil {
		return fmt.Errorf("无效的项目ID格式: %w", err)
	}

	if event.RepositoryID != "" {
		if _, err := uuid.Parse(event.RepositoryID); err != nil {
			return fmt.Errorf("无效的仓库ID格式: %w", err)
		}
	}

	return nil
}

// processEventAsync 异步处理事件
func (h *WebhookHandler) processEventAsync(event *GitEvent) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	defer func() {
		if r := recover(); r != nil {
			h.logger.Error("处理Git事件时发生panic",
				zap.String("event_id", event.EventID),
				zap.String("event_type", event.EventType),
				zap.Any("panic", r))
		}
	}()

	if err := h.processEvent(ctx, event); err != nil {
		h.logger.Error("处理Git事件失败",
			zap.Error(err),
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
			zap.String("project_id", event.ProjectID))
	} else {
		h.logger.Info("Git事件处理成功",
			zap.String("event_id", event.EventID),
			zap.String("event_type", event.EventType),
			zap.String("project_id", event.ProjectID))
	}
}

// processEvent 处理具体事件
func (h *WebhookHandler) processEvent(ctx context.Context, event *GitEvent) error {
	switch event.EventType {
	case "repository":
		return h.processRepositoryEvent(ctx, event)
	case "branch":
		return h.processBranchEvent(ctx, event)
	case "commit":
		return h.processCommitEvent(ctx, event)
	case "push":
		return h.processPushEvent(ctx, event)
	case "tag":
		return h.processTagEvent(ctx, event)
	default:
		h.logger.Warn("未知的事件类型",
			zap.String("event_type", event.EventType),
			zap.String("event_id", event.EventID))
		return nil // 忽略未知事件类型
	}
}

// processRepositoryEvent 处理仓库事件
func (h *WebhookHandler) processRepositoryEvent(ctx context.Context, event *GitEvent) error {
	var payload RepositoryEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("解析仓库事件负载失败: %w", err)
	}

	return h.eventProcessor.ProcessRepositoryEvent(ctx, event, &payload)
}

// processBranchEvent 处理分支事件
func (h *WebhookHandler) processBranchEvent(ctx context.Context, event *GitEvent) error {
	var payload BranchEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("解析分支事件负载失败: %w", err)
	}

	return h.eventProcessor.ProcessBranchEvent(ctx, event, &payload)
}

// processCommitEvent 处理提交事件
func (h *WebhookHandler) processCommitEvent(ctx context.Context, event *GitEvent) error {
	var payload CommitEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("解析提交事件负载失败: %w", err)
	}

	return h.eventProcessor.ProcessCommitEvent(ctx, event, &payload)
}

// processPushEvent 处理推送事件
func (h *WebhookHandler) processPushEvent(ctx context.Context, event *GitEvent) error {
	var payload PushEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("解析推送事件负载失败: %w", err)
	}

	return h.eventProcessor.ProcessPushEvent(ctx, event, &payload)
}

// processTagEvent 处理标签事件
func (h *WebhookHandler) processTagEvent(ctx context.Context, event *GitEvent) error {
	var payload TagEvent
	if err := json.Unmarshal(event.Payload, &payload); err != nil {
		return fmt.Errorf("解析标签事件负载失败: %w", err)
	}

	return h.eventProcessor.ProcessTagEvent(ctx, event, &payload)
}

// GetHealthCheck 健康检查端点
func (h *WebhookHandler) GetHealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "git-webhook-handler",
		"timestamp": time.Now().UTC(),
	})
}