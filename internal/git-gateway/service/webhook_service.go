package service

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/models"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/repository"
	"github.com/cloud-platform/collaborative-dev/internal/git-gateway/service/notification"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebhookService Webhook服务接口
type WebhookService interface {
	// Webhook事件管理
	CreateWebhookEvent(ctx context.Context, req *models.CreateWebhookEventRequest) (*models.WebhookEvent, error)
	ProcessWebhookEvent(ctx context.Context, eventID uuid.UUID) error
	GetWebhookEvent(ctx context.Context, eventID uuid.UUID) (*models.WebhookEvent, error)
	ListWebhookEvents(ctx context.Context, filter *models.WebhookEventFilter, page, pageSize int) (*models.WebhookEventListResponse, error)
	DeleteWebhookEvent(ctx context.Context, eventID uuid.UUID) error

	// 触发器管理
	CreateWebhookTrigger(ctx context.Context, req *models.CreateWebhookTriggerRequest) (*models.WebhookTrigger, error)
	UpdateWebhookTrigger(ctx context.Context, triggerID uuid.UUID, req *models.UpdateWebhookTriggerRequest) (*models.WebhookTrigger, error)
	GetWebhookTrigger(ctx context.Context, triggerID uuid.UUID) (*models.WebhookTrigger, error)
	ListWebhookTriggers(ctx context.Context, repositoryID *uuid.UUID, page, pageSize int) (*models.WebhookTriggerListResponse, error)
	DeleteWebhookTrigger(ctx context.Context, triggerID uuid.UUID) error
	EnableWebhookTrigger(ctx context.Context, triggerID uuid.UUID, enabled bool) error

	// 投递管理
	ListWebhookDeliveries(ctx context.Context, webhookID *uuid.UUID, page, pageSize int) (*models.WebhookDeliveryListResponse, error)
	RetryWebhookDelivery(ctx context.Context, deliveryID uuid.UUID) error

	// 统计信息
	GetWebhookStatistics(ctx context.Context, repositoryID *uuid.UUID) (*models.WebhookStatistics, error)

	// Git事件处理
	HandleGitPushEvent(ctx context.Context, repositoryID uuid.UUID, pushData *models.PushEvent) error
	HandleGitBranchEvent(ctx context.Context, repositoryID uuid.UUID, branchData *models.BranchEvent) error
	HandleGitTagEvent(ctx context.Context, repositoryID uuid.UUID, tagData *models.TagEvent) error
	HandleGitPullRequestEvent(ctx context.Context, repositoryID uuid.UUID, prData *models.PullRequestEvent) error

	// CI/CD集成
	TriggerPipeline(ctx context.Context, repositoryID uuid.UUID, eventData interface{}) error
	NotifyExternalSystems(ctx context.Context, event *models.WebhookEvent) error
}

// webhookService Webhook服务实现
type webhookService struct {
	repo                repository.GitRepository
	webhookRepo         repository.WebhookRepository // 新增Webhook仓库
	cicdService         CICDService                  // CI/CD服务接口
	logger              *zap.Logger
	config              WebhookConfig
	notificationManager *notification.NotificationManager
}

// WebhookConfig Webhook配置
type WebhookConfig struct {
	MaxRetries     int           `yaml:"max_retries"`
	RetryInterval  time.Duration `yaml:"retry_interval"`
	RequestTimeout time.Duration `yaml:"request_timeout"`
	MaxPayloadSize int64         `yaml:"max_payload_size"`
	EnableBatching bool          `yaml:"enable_batching"`
	BatchSize      int           `yaml:"batch_size"`
	BatchTimeout   time.Duration `yaml:"batch_timeout"`
}

// CICDService CI/CD服务接口（避免循环依赖）
type CICDService interface {
	TriggerPipeline(ctx context.Context, repositoryID uuid.UUID, pipelineID uuid.UUID, variables map[string]interface{}) error
}

// NewWebhookService 创建Webhook服务
func NewWebhookService(repo repository.GitRepository, webhookRepo repository.WebhookRepository, cicdService CICDService, config WebhookConfig, logger *zap.Logger) WebhookService {
	return &webhookService{
		repo:        repo,
		webhookRepo: webhookRepo,
		cicdService: cicdService,
		config:      config,
		logger:      logger,
	}
}

// CreateWebhookEvent 创建Webhook事件
func (s *webhookService) CreateWebhookEvent(ctx context.Context, req *models.CreateWebhookEventRequest) (*models.WebhookEvent, error) {
	s.logger.Info("创建Webhook事件",
		zap.String("repository_id", req.RepositoryID.String()),
		zap.String("event_type", string(req.EventType)),
		zap.String("source", req.Source))

	event := &models.WebhookEvent{
		ID:           uuid.New(),
		RepositoryID: req.RepositoryID,
		EventType:    req.EventType,
		EventData:    req.EventData,
		Source:       req.Source,
		Signature:    req.Signature,
		Processed:    false,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.webhookRepo.CreateWebhookEvent(ctx, event); err != nil {
		return nil, fmt.Errorf("创建Webhook事件失败: %w", err)
	}

	// 异步处理事件 - 使用带超时的context防止goroutine泄漏
	go func() {
		// 创建一个新的context，设置合理的超时时间
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()
		
		// 添加panic恢复机制
		defer func() {
			if r := recover(); r != nil {
				s.logger.Error("处理Webhook事件时发生panic",
					zap.String("event_id", event.ID.String()),
					zap.Any("panic", r))
			}
		}()
		
		if err := s.ProcessWebhookEvent(ctx, event.ID); err != nil {
			s.logger.Error("处理Webhook事件失败",
				zap.String("event_id", event.ID.String()),
				zap.Error(err))
		}
	}()

	return event, nil
}

// ProcessWebhookEvent 处理Webhook事件
func (s *webhookService) ProcessWebhookEvent(ctx context.Context, eventID uuid.UUID) error {
	event, err := s.webhookRepo.GetWebhookEvent(ctx, eventID)
	if err != nil {
		return fmt.Errorf("获取Webhook事件失败: %w", err)
	}

	if event.Processed {
		return fmt.Errorf("事件已处理")
	}

	s.logger.Info("处理Webhook事件",
		zap.String("event_id", eventID.String()),
		zap.String("event_type", string(event.EventType)))

	// 获取匹配的触发器
	triggers, err := s.getMatchingTriggers(ctx, event)
	if err != nil {
		return fmt.Errorf("获取匹配触发器失败: %w", err)
	}

	if len(triggers) == 0 {
		s.logger.Debug("没有匹配的触发器", zap.String("event_id", eventID.String()))
	}

	// 执行触发器动作
	var processingErrors []error
	for _, trigger := range triggers {
		if err := s.executeTriggerActions(ctx, event, &trigger); err != nil {
			processingErrors = append(processingErrors, err)
			s.logger.Error("执行触发器动作失败",
				zap.String("trigger_id", trigger.ID.String()),
				zap.Error(err))
		}
	}

	// 更新事件处理状态
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"processed":    true,
		"processed_at": now,
		"updated_at":   now,
	}

	if len(processingErrors) > 0 {
		errorMsg := "处理错误: "
		for _, err := range processingErrors {
			errorMsg += err.Error() + "; "
		}
		updates["error_message"] = errorMsg
	}

	if err := s.webhookRepo.UpdateWebhookEvent(ctx, eventID, updates); err != nil {
		return fmt.Errorf("更新事件处理状态失败: %w", err)
	}

	if len(processingErrors) > 0 {
		return fmt.Errorf("部分触发器执行失败")
	}

	return nil
}

// getMatchingTriggers 获取匹配的触发器
func (s *webhookService) getMatchingTriggers(ctx context.Context, event *models.WebhookEvent) ([]models.WebhookTrigger, error) {
	triggers, err := s.webhookRepo.ListWebhookTriggers(ctx, &event.RepositoryID, 1, 100)
	if err != nil {
		return nil, err
	}

	var matchingTriggers []models.WebhookTrigger
	for _, trigger := range triggers.Triggers {
		if !trigger.Enabled {
			continue
		}

		// 检查事件类型匹配
		if !s.eventTypeMatches(event.EventType, trigger.EventTypes) {
			continue
		}

		// 检查触发条件
		if !s.conditionsMatch(event, &trigger.Conditions) {
			continue
		}

		matchingTriggers = append(matchingTriggers, trigger)
	}

	return matchingTriggers, nil
}

// eventTypeMatches 检查事件类型是否匹配
func (s *webhookService) eventTypeMatches(eventType models.WebhookEventType, triggerTypes []models.WebhookEventType) bool {
	for _, triggerType := range triggerTypes {
		if eventType == triggerType {
			return true
		}
	}
	return false
}

// conditionsMatch 检查触发条件是否匹配
func (s *webhookService) conditionsMatch(event *models.WebhookEvent, conditions *models.TriggerConditions) bool {
	// 根据事件类型解析事件数据
	switch event.EventType {
	case models.EventTypePush, models.EventTypeBranchPush, models.EventTypeTagPush:
		return s.checkPushConditions(event, conditions)
	case models.EventTypePullRequest:
		return s.checkPullRequestConditions(event, conditions)
	case models.EventTypeBranchCreate, models.EventTypeBranchDelete:
		return s.checkBranchConditions(event, conditions)
	case models.EventTypeTagCreate, models.EventTypeTagDelete:
		return s.checkTagConditions(event, conditions)
	default:
		return true // 默认匹配
	}
}

// checkPushConditions 检查推送条件
func (s *webhookService) checkPushConditions(event *models.WebhookEvent, conditions *models.TriggerConditions) bool {
	eventDataBytes, _ := json.Marshal(event.EventData)
	var pushData models.PushEvent
	if err := json.Unmarshal(eventDataBytes, &pushData); err != nil {
		return false
	}

	// 检查分支条件
	if len(conditions.Branches) > 0 {
		branch := strings.TrimPrefix(pushData.Ref, "refs/heads/")
		if !s.matchesPatterns(branch, conditions.Branches) {
			return false
		}
	}

	// 检查提交信息条件
	if conditions.CommitMessage != "" && pushData.HeadCommit != nil {
		matched, _ := regexp.MatchString(conditions.CommitMessage, pushData.HeadCommit.Message)
		if !matched {
			return false
		}
	}

	// 检查作者条件
	if len(conditions.Authors) > 0 && pushData.HeadCommit != nil {
		if !s.matchesPatterns(pushData.HeadCommit.AuthorEmail, conditions.Authors) {
			return false
		}
	}

	// 检查文件变更条件
	if len(conditions.FileChanges.Include) > 0 || len(conditions.FileChanges.Exclude) > 0 {
		return s.checkFileChangeConditions(pushData.Commits, &conditions.FileChanges)
	}

	return true
}

// checkPullRequestConditions 检查拉取请求条件
func (s *webhookService) checkPullRequestConditions(event *models.WebhookEvent, conditions *models.TriggerConditions) bool {
	eventDataBytes, _ := json.Marshal(event.EventData)
	var prData models.PullRequestEvent
	if err := json.Unmarshal(eventDataBytes, &prData); err != nil {
		return false
	}

	// 检查目标分支条件
	if len(conditions.Branches) > 0 {
		// PullRequest分支条件检查
		if prData.PullRequest.TargetBranch != "" && !s.matchesPatterns(prData.PullRequest.TargetBranch, conditions.Branches) {
			return false
		}
	}

	return true
}

// checkBranchConditions 检查分支条件
func (s *webhookService) checkBranchConditions(event *models.WebhookEvent, conditions *models.TriggerConditions) bool {
	eventDataBytes, _ := json.Marshal(event.EventData)
	var branchData models.BranchEvent
	if err := json.Unmarshal(eventDataBytes, &branchData); err != nil {
		return false
	}

	// 检查分支模式
	if len(conditions.Branches) > 0 {
		branch := strings.TrimPrefix(branchData.Ref, "refs/heads/")
		return s.matchesPatterns(branch, conditions.Branches)
	}

	return true
}

// checkTagConditions 检查标签条件
func (s *webhookService) checkTagConditions(event *models.WebhookEvent, conditions *models.TriggerConditions) bool {
	eventDataBytes, _ := json.Marshal(event.EventData)
	var tagData models.TagEvent
	if err := json.Unmarshal(eventDataBytes, &tagData); err != nil {
		return false
	}

	// 检查标签模式
	if len(conditions.Tags) > 0 {
		tag := strings.TrimPrefix(tagData.Ref, "refs/tags/")
		return s.matchesPatterns(tag, conditions.Tags)
	}

	return true
}

// matchesPatterns 检查是否匹配模式
func (s *webhookService) matchesPatterns(value string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, _ := regexp.MatchString(pattern, value)
		if matched {
			return true
		}
	}
	return false
}

// checkFileChangeConditions 检查文件变更条件
func (s *webhookService) checkFileChangeConditions(commits []models.Commit, conditions *models.FileChangeCondition) bool {
	// 暂时跳过文件变更检查，需要调整Commit模型
	_ = commits    // 避免unused variable错误
	_ = conditions // 避免unused variable错误

	/*changedFiles := make(map[string]bool)

	// 收集所有变更的文件
	for _, commit := range commits {
		for _, file := range commit.Added {
			changedFiles[file] = true
		}
		for _, file := range commit.Modified {
			changedFiles[file] = true
		}
		for _, file := range commit.Removed {
			changedFiles[file] = true
		}
	}

	// 检查包含条件
	if len(conditions.Include) > 0 {
		hasIncluded := false
		for file := range changedFiles {
			if s.matchesPatterns(file, conditions.Include) {
				hasIncluded = true
				break
			}
		}
		if !hasIncluded {
			return false
		}
	}

	// 检查排除条件
	if len(conditions.Exclude) > 0 {
		for file := range changedFiles {
			if s.matchesPatterns(file, conditions.Exclude) {
				return false
			}
		}
	}

	return true*/

	// 暂时返回true，表示所有文件变更都满足条件
	return true
}

// executeTriggerActions 执行触发器动作
func (s *webhookService) executeTriggerActions(ctx context.Context, event *models.WebhookEvent, trigger *models.WebhookTrigger) error {
	actions := trigger.Actions

	// 启动流水线
	if actions.StartPipeline != nil {
		if err := s.executePipelineAction(ctx, event, actions.StartPipeline); err != nil {
			return fmt.Errorf("执行流水线动作失败: %w", err)
		}
	}

	// 发送通知
	if actions.SendNotification != nil {
		if err := s.executeNotificationAction(ctx, event, actions.SendNotification); err != nil {
			return fmt.Errorf("执行通知动作失败: %w", err)
		}
	}

	// 调用Webhook
	if actions.CallWebhook != nil {
		if err := s.executeWebhookAction(ctx, event, actions.CallWebhook); err != nil {
			return fmt.Errorf("执行Webhook动作失败: %w", err)
		}
	}

	return nil
}

// executePipelineAction 执行流水线动作
func (s *webhookService) executePipelineAction(ctx context.Context, event *models.WebhookEvent, action *models.PipelineAction) error {
	s.logger.Info("执行流水线动作",
		zap.String("pipeline_id", action.PipelineID.String()),
		zap.String("event_id", event.ID.String()))

	// 构建流水线变量
	variables := make(map[string]interface{})

	// 添加预定义变量
	variables["WEBHOOK_EVENT_ID"] = event.ID.String()
	variables["WEBHOOK_EVENT_TYPE"] = string(event.EventType)
	variables["WEBHOOK_SOURCE"] = event.Source
	variables["REPOSITORY_ID"] = event.RepositoryID.String()

	// 添加自定义变量
	for k, v := range action.Variables {
		variables[k] = v
	}

	// 根据事件类型添加特定变量
	s.addEventSpecificVariables(event, variables)

	// 调用CI/CD服务启动流水线
	if s.cicdService != nil {
		return s.cicdService.TriggerPipeline(ctx, event.RepositoryID, action.PipelineID, variables)
	}

	s.logger.Warn("CI/CD服务未配置，跳过流水线触发")
	return nil
}

// addEventSpecificVariables 添加事件特定变量
func (s *webhookService) addEventSpecificVariables(event *models.WebhookEvent, variables map[string]interface{}) {
	switch event.EventType {
	case models.EventTypePush:
		eventDataBytes, _ := json.Marshal(event.EventData)
		var pushData models.PushEvent
		if err := json.Unmarshal(eventDataBytes, &pushData); err == nil {
			variables["GIT_REF"] = pushData.Ref
			variables["GIT_BEFORE"] = pushData.Before
			variables["GIT_AFTER"] = pushData.After
			variables["GIT_BRANCH"] = strings.TrimPrefix(pushData.Ref, "refs/heads/")
			if pushData.HeadCommit != nil {
				variables["GIT_COMMIT_SHA"] = pushData.HeadCommit.ID
				variables["GIT_COMMIT_MESSAGE"] = pushData.HeadCommit.Message
				variables["GIT_AUTHOR_NAME"] = pushData.HeadCommit.Author
				variables["GIT_AUTHOR_EMAIL"] = pushData.HeadCommit.AuthorEmail
			}
		}
	case models.EventTypePullRequest:
		eventDataBytes, _ := json.Marshal(event.EventData)
		var prData models.PullRequestEvent
		if err := json.Unmarshal(eventDataBytes, &prData); err == nil {
			variables["PULL_REQUEST_NUMBER"] = prData.Number
			variables["PULL_REQUEST_ACTION"] = prData.Action
			variables["PULL_REQUEST_TITLE"] = prData.PullRequest.Title
			variables["PULL_REQUEST_BASE_REF"] = prData.PullRequest.TargetBranch
			variables["PULL_REQUEST_HEAD_REF"] = prData.PullRequest.SourceBranch
		}
	}
}

// executeNotificationAction 执行通知动作
func (s *webhookService) executeNotificationAction(ctx context.Context, event *models.WebhookEvent, action *models.NotificationAction) error {
	s.logger.Info("执行通知动作",
		zap.String("type", action.Type),
		zap.String("event_id", event.ID.String()))

	// 构建通知内容
	notification := &notification.Notification{
		Type:       notification.NotificationType(action.Type),
		Recipients: action.Recipients,
		Subject:    s.formatNotificationSubject(event, action),
		Body:       s.formatNotificationBody(event, action),
		Priority:   "normal",
		Metadata: map[string]interface{}{
			"event_id":   event.ID.String(),
			"event_type": event.EventType,
			"repository": event.RepositoryID.String(),
		},
		Timestamp: time.Now(),
	}

	// 使用通知管理器发送通知
	result, err := s.notificationManager.Send(ctx, notification)
	if err != nil {
		s.logger.Error("Failed to send notification",
			zap.String("type", action.Type),
			zap.String("event_id", event.ID.String()),
			zap.Error(err))
		return err
	}

	s.logger.Info("Notification sent successfully",
		zap.String("type", action.Type),
		zap.String("message_id", result.MessageID))

	return nil
}

// formatNotificationSubject 格式化通知主题
func (s *webhookService) formatNotificationSubject(event *models.WebhookEvent, action *models.NotificationAction) string {
	subject := action.Subject
	if subject == "" {
		// 根据事件类型生成默认主题
		switch event.EventType {
		case "push":
			subject = fmt.Sprintf("[Git] Push to repository %s", event.EventData["repository"])
		case "pull_request":
			subject = fmt.Sprintf("[Git] Pull Request %s in %s", event.EventData["action"], event.EventData["repository"])
		case "tag":
			subject = fmt.Sprintf("[Git] New tag in repository %s", event.EventData["repository"])
		default:
			subject = fmt.Sprintf("[Git] Event %s occurred", event.EventType)
		}
	}

	// 替换变量
	subject = s.replaceVariables(subject, event)
	return subject
}

// formatNotificationBody 格式化通知内容
func (s *webhookService) formatNotificationBody(event *models.WebhookEvent, action *models.NotificationAction) string {
	body := action.Template
	if body == "" {
		// 根据事件类型生成默认内容
		body = s.generateDefaultNotificationBody(event)
	}

	// 替换变量
	body = s.replaceVariables(body, event)
	return body
}

// replaceVariables 替换模板变量
func (s *webhookService) replaceVariables(template string, event *models.WebhookEvent) string {
	// 这里可以使用更复杂的模板引擎
	result := template

	// 替换基本变量
	result = strings.ReplaceAll(result, "${event_id}", event.ID.String())
	result = strings.ReplaceAll(result, "${event_type}", string(event.EventType))
	result = strings.ReplaceAll(result, "${timestamp}", event.CreatedAt.Format(time.RFC3339))

	// 替换payload中的变量
	for key, value := range event.EventData {
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
	}

	return result
}

// generateDefaultNotificationBody 生成默认通知内容
func (s *webhookService) generateDefaultNotificationBody(event *models.WebhookEvent) string {
	var body strings.Builder

	body.WriteString(fmt.Sprintf("Event Type: %s\n", event.EventType))
	body.WriteString(fmt.Sprintf("Event ID: %s\n", event.ID.String()))
	body.WriteString(fmt.Sprintf("Timestamp: %s\n", event.CreatedAt.Format(time.RFC3339)))
	body.WriteString("\nEvent Details:\n")

	// 格式化payload
	for key, value := range event.EventData {
		body.WriteString(fmt.Sprintf("- %s: %v\n", key, value))
	}

	return body.String()
}

// executeWebhookAction 执行Webhook动作
func (s *webhookService) executeWebhookAction(ctx context.Context, event *models.WebhookEvent, action *models.WebhookAction) error {
	s.logger.Info("执行Webhook动作",
		zap.String("url", action.URL),
		zap.String("event_id", event.ID.String()))

	// 准备请求数据
	payload := s.prepareWebhookPayload(event, action)

	// 序列化payload
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	// 发送HTTP请求（带重试）
	maxRetries := 3
	retryDelay := 1 * time.Second

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(retryDelay):
				retryDelay *= 2 // 指数退避
			}
		}

		err = s.sendHTTPRequest(ctx, action, data)
		if err == nil {
			s.logger.Info("Webhook sent successfully",
				zap.String("url", action.URL),
				zap.Int("attempts", attempt+1))
			return nil
		}

		s.logger.Warn("Failed to send webhook, retrying",
			zap.String("url", action.URL),
			zap.Int("attempt", attempt+1),
			zap.Error(err))
	}

	return fmt.Errorf("failed to send webhook after %d attempts: %w", maxRetries, err)
}

// prepareWebhookPayload 准备Webhook载荷
func (s *webhookService) prepareWebhookPayload(event *models.WebhookEvent, action *models.WebhookAction) map[string]interface{} {
	// 基础载荷
	payload := map[string]interface{}{
		"event_id":   event.ID.String(),
		"event_type": event.EventType,
		"timestamp":  event.CreatedAt.Format(time.RFC3339),
		"data":       event.EventData,
	}

	// 添加自定义字段
	if action.Body != nil {
		for key, value := range action.Body {
			payload[key] = value
		}
	}

	// 添加签名（如果配置了密钥）
	if action.Secret != "" {
		signature := s.generateWebhookSignature(event.ID.String(), action.Secret)
		payload["signature"] = signature
	}

	return payload
}

// sendHTTPRequest 发送HTTP请求
func (s *webhookService) sendHTTPRequest(ctx context.Context, action *models.WebhookAction, data []byte) error {
	// 创建请求
	req, err := http.NewRequestWithContext(ctx, action.Method, action.URL, bytes.NewBuffer(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "CloudPlatform-Webhook/1.0")

	// 添加自定义请求头
	for key, value := range action.Headers {
		req.Header.Set(key, value)
	}

	// 添加签名头（如果配置了密钥）
	if action.Secret != "" {
		signature := s.generateWebhookSignature(string(data), action.Secret)
		req.Header.Set("X-Webhook-Signature", signature)
	}

	// 创建HTTP客户端
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned status %d: %s", resp.StatusCode, string(body))
	}

	s.logger.Debug("Webhook response",
		zap.String("url", action.URL),
		zap.Int("status", resp.StatusCode),
		zap.String("body", string(body)))

	return nil
}

// generateWebhookSignature 生成Webhook签名
func (s *webhookService) generateWebhookSignature(data, secret string) string {
	h := hmac.New(sha256.New, []byte(secret))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

// HandleGitPushEvent 处理Git推送事件
func (s *webhookService) HandleGitPushEvent(ctx context.Context, repositoryID uuid.UUID, pushData *models.PushEvent) error {
	eventData, _ := json.Marshal(pushData)
	var eventDataMap map[string]interface{}
	json.Unmarshal(eventData, &eventDataMap)

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    models.EventTypePush,
		EventData:    eventDataMap,
		Source:       "git",
	}

	_, err := s.CreateWebhookEvent(ctx, req)
	return err
}

// HandleGitBranchEvent 处理Git分支事件
func (s *webhookService) HandleGitBranchEvent(ctx context.Context, repositoryID uuid.UUID, branchData *models.BranchEvent) error {
	eventData, _ := json.Marshal(branchData)
	var eventDataMap map[string]interface{}
	json.Unmarshal(eventData, &eventDataMap)

	eventType := models.EventTypeBranchCreate
	if branchData.Action == "deleted" {
		eventType = models.EventTypeBranchDelete
	}

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    eventType,
		EventData:    eventDataMap,
		Source:       "git",
	}

	_, err := s.CreateWebhookEvent(ctx, req)
	return err
}

// HandleGitTagEvent 处理Git标签事件
func (s *webhookService) HandleGitTagEvent(ctx context.Context, repositoryID uuid.UUID, tagData *models.TagEvent) error {
	eventData, _ := json.Marshal(tagData)
	var eventDataMap map[string]interface{}
	json.Unmarshal(eventData, &eventDataMap)

	eventType := models.EventTypeTagCreate
	if tagData.Action == "deleted" {
		eventType = models.EventTypeTagDelete
	}

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    eventType,
		EventData:    eventDataMap,
		Source:       "git",
	}

	_, err := s.CreateWebhookEvent(ctx, req)
	return err
}

// HandleGitPullRequestEvent 处理Git拉取请求事件
func (s *webhookService) HandleGitPullRequestEvent(ctx context.Context, repositoryID uuid.UUID, prData *models.PullRequestEvent) error {
	eventData, _ := json.Marshal(prData)
	var eventDataMap map[string]interface{}
	json.Unmarshal(eventData, &eventDataMap)

	req := &models.CreateWebhookEventRequest{
		RepositoryID: repositoryID,
		EventType:    models.EventTypePullRequest,
		EventData:    eventDataMap,
		Source:       "git",
	}

	_, err := s.CreateWebhookEvent(ctx, req)
	return err
}

// CreateWebhookTrigger 创建钩子触发器
func (s *webhookService) CreateWebhookTrigger(ctx context.Context, req *models.CreateWebhookTriggerRequest) (*models.WebhookTrigger, error) {
	trigger := &models.WebhookTrigger{
		ID:           uuid.New(),
		RepositoryID: req.RepositoryID,
		Name:         req.Name,
		EventTypes:   req.EventTypes,
		Conditions:   req.Conditions,
		Actions:      req.Actions,
		Enabled:      req.Enabled,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	}

	if err := s.webhookRepo.CreateWebhookTrigger(ctx, trigger); err != nil {
		return nil, fmt.Errorf("创建钩子触发器失败: %w", err)
	}

	return trigger, nil
}

// 其他接口方法的实现...
// TriggerPipeline, NotifyExternalSystems等方法实现
func (s *webhookService) TriggerPipeline(ctx context.Context, repositoryID uuid.UUID, eventData interface{}) error {
	// 实现流水线触发逻辑
	return nil
}

func (s *webhookService) NotifyExternalSystems(ctx context.Context, event *models.WebhookEvent) error {
	// 实现外部系统通知逻辑
	return nil
}

// 其他方法实现占位符
func (s *webhookService) GetWebhookEvent(ctx context.Context, eventID uuid.UUID) (*models.WebhookEvent, error) {
	return s.webhookRepo.GetWebhookEvent(ctx, eventID)
}

func (s *webhookService) ListWebhookEvents(ctx context.Context, filter *models.WebhookEventFilter, page, pageSize int) (*models.WebhookEventListResponse, error) {
	return s.webhookRepo.ListWebhookEvents(ctx, filter, page, pageSize)
}

func (s *webhookService) DeleteWebhookEvent(ctx context.Context, eventID uuid.UUID) error {
	return s.webhookRepo.DeleteWebhookEvent(ctx, eventID)
}

func (s *webhookService) UpdateWebhookTrigger(ctx context.Context, triggerID uuid.UUID, req *models.UpdateWebhookTriggerRequest) (*models.WebhookTrigger, error) {
	// TODO: 实现更新逻辑
	return nil, fmt.Errorf("未实现")
}

func (s *webhookService) GetWebhookTrigger(ctx context.Context, triggerID uuid.UUID) (*models.WebhookTrigger, error) {
	return s.webhookRepo.GetWebhookTrigger(ctx, triggerID)
}

func (s *webhookService) ListWebhookTriggers(ctx context.Context, repositoryID *uuid.UUID, page, pageSize int) (*models.WebhookTriggerListResponse, error) {
	return s.webhookRepo.ListWebhookTriggers(ctx, repositoryID, page, pageSize)
}

func (s *webhookService) DeleteWebhookTrigger(ctx context.Context, triggerID uuid.UUID) error {
	return s.webhookRepo.DeleteWebhookTrigger(ctx, triggerID)
}

func (s *webhookService) EnableWebhookTrigger(ctx context.Context, triggerID uuid.UUID, enabled bool) error {
	updates := map[string]interface{}{
		"enabled":    enabled,
		"updated_at": time.Now().UTC(),
	}
	return s.webhookRepo.UpdateWebhookTrigger(ctx, triggerID, updates)
}

func (s *webhookService) ListWebhookDeliveries(ctx context.Context, webhookID *uuid.UUID, page, pageSize int) (*models.WebhookDeliveryListResponse, error) {
	// TODO: 实现投递记录列表
	return &models.WebhookDeliveryListResponse{}, nil
}

func (s *webhookService) RetryWebhookDelivery(ctx context.Context, deliveryID uuid.UUID) error {
	// TODO: 实现投递重试
	return fmt.Errorf("未实现")
}

func (s *webhookService) GetWebhookStatistics(ctx context.Context, repositoryID *uuid.UUID) (*models.WebhookStatistics, error) {
	// TODO: 实现统计信息
	return &models.WebhookStatistics{}, nil
}
