package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/cicd-service/scheduler"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebhookClient CI/CD服务的Webhook客户端接口
type WebhookClient interface {
	// 处理Git钩子事件
	HandleGitPushEvent(ctx context.Context, event *GitPushEvent) error
	HandleBranchEvent(ctx context.Context, event *GitBranchEvent) error
	HandleTagEvent(ctx context.Context, event *GitTagEvent) error
	HandlePullRequestEvent(ctx context.Context, event *GitPullRequestEvent) error
	
	// 触发流水线
	TriggerPipeline(ctx context.Context, repositoryID, pipelineID uuid.UUID, variables map[string]interface{}) error
	
	// 注册事件监听器
	RegisterEventListener(eventType string, listener EventListener) error
	UnregisterEventListener(eventType string, listener EventListener) error
}

// EventListener 事件监听器接口
type EventListener interface {
	HandleEvent(ctx context.Context, event interface{}) error
	GetEventTypes() []string
}

// Git事件结构体

// GitPushEvent Git推送事件
type GitPushEvent struct {
	RepositoryID uuid.UUID              `json:"repository_id"`
	Ref          string                 `json:"ref"`          // refs/heads/main
	Before       string                 `json:"before"`       // 推送前SHA
	After        string                 `json:"after"`        // 推送后SHA  
	Created      bool                   `json:"created"`      // 是否新建分支
	Deleted      bool                   `json:"deleted"`      // 是否删除分支
	Forced       bool                   `json:"forced"`       // 是否强制推送
	Commits      []GitCommit            `json:"commits"`      // 提交列表
	HeadCommit   *GitCommit             `json:"head_commit"`  // 头部提交
	Pusher       GitUser                `json:"pusher"`       // 推送者
	Repository   GitRepository          `json:"repository"`   // 仓库信息
	Variables    map[string]interface{} `json:"variables"`    // 自定义变量
}

// GitBranchEvent Git分支事件
type GitBranchEvent struct {
	RepositoryID uuid.UUID     `json:"repository_id"`
	Action       string        `json:"action"`     // created, deleted
	BranchName   string        `json:"branch_name"`
	SHA          string        `json:"sha"`
	Repository   GitRepository `json:"repository"`
	Sender       GitUser       `json:"sender"`
}

// GitTagEvent Git标签事件  
type GitTagEvent struct {
	RepositoryID uuid.UUID     `json:"repository_id"`
	Action       string        `json:"action"` // created, deleted
	TagName      string        `json:"tag_name"`
	SHA          string        `json:"sha"`
	Repository   GitRepository `json:"repository"`
	Sender       GitUser       `json:"sender"`
}

// GitPullRequestEvent Git拉取请求事件
type GitPullRequestEvent struct {
	RepositoryID  uuid.UUID     `json:"repository_id"`
	Action        string        `json:"action"`        // opened, closed, merged
	Number        int           `json:"number"`        // PR编号
	PullRequest   GitPullRequest `json:"pull_request"` // PR详情
	Repository    GitRepository `json:"repository"`    // 仓库信息
	Sender        GitUser       `json:"sender"`        // 发送者
}

// Git辅助结构体

type GitCommit struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Author    GitUser   `json:"author"`
	Committer GitUser   `json:"committer"`
	Added     []string  `json:"added"`
	Modified  []string  `json:"modified"`
	Removed   []string  `json:"removed"`
}

type GitUser struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Username string `json:"username"`
}

type GitRepository struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	FullName string    `json:"full_name"`
	CloneURL string    `json:"clone_url"`
	SSHURL   string    `json:"ssh_url"`
	Branch   string    `json:"default_branch"`
}

type GitPullRequest struct {
	ID     int    `json:"id"`
	Number int    `json:"number"`
	Title  string `json:"title"`
	Body   string `json:"body"`
	State  string `json:"state"`
	Base   GitRef `json:"base"`
	Head   GitRef `json:"head"`
	User   GitUser `json:"user"`
}

type GitRef struct {
	Ref    string        `json:"ref"`
	SHA    string        `json:"sha"`
	Repo   GitRepository `json:"repo"`
}

// webhookClient Webhook客户端实现
type webhookClient struct {
	scheduler   scheduler.JobScheduler
	logger      *zap.Logger
	listeners   map[string][]EventListener
	config      WebhookClientConfig
}

// WebhookClientConfig Webhook客户端配置
type WebhookClientConfig struct {
	DefaultPipelineID uuid.UUID         `yaml:"default_pipeline_id"`
	EventMappings     map[string]string `yaml:"event_mappings"`     // 事件类型映射
	DefaultVariables  map[string]interface{} `yaml:"default_variables"` // 默认变量
	AutoTrigger       bool              `yaml:"auto_trigger"`       // 自动触发
	FilterRules       []FilterRule      `yaml:"filter_rules"`       // 过滤规则
}

// FilterRule 过滤规则
type FilterRule struct {
	EventType string   `yaml:"event_type"`
	Branches  []string `yaml:"branches"`  // 匹配的分支
	Paths     []string `yaml:"paths"`     // 匹配的路径
	Authors   []string `yaml:"authors"`   // 匹配的作者
}

// NewWebhookClient 创建Webhook客户端
func NewWebhookClient(scheduler scheduler.JobScheduler, config WebhookClientConfig, logger *zap.Logger) WebhookClient {
	return &webhookClient{
		scheduler: scheduler,
		logger:    logger,
		listeners: make(map[string][]EventListener),
		config:    config,
	}
}

// HandleGitPushEvent 处理Git推送事件
func (c *webhookClient) HandleGitPushEvent(ctx context.Context, event *GitPushEvent) error {
	c.logger.Info("处理Git推送事件", 
		zap.String("repository_id", event.RepositoryID.String()),
		zap.String("ref", event.Ref),
		zap.String("after", event.After))

	// 检查过滤规则
	if !c.matchesFilterRules("push", event) {
		c.logger.Debug("事件不匹配过滤规则，跳过处理")
		return nil
	}

	// 构建流水线变量
	variables := c.buildPushVariables(event)
	
	// 触发监听器
	if err := c.triggerListeners(ctx, "push", event); err != nil {
		c.logger.Error("触发事件监听器失败", zap.Error(err))
	}

	// 自动触发流水线
	if c.config.AutoTrigger {
		pipelineID := c.config.DefaultPipelineID
		if pipelineID == uuid.Nil {
			c.logger.Warn("未配置默认流水线ID，跳过自动触发")
			return nil
		}

		return c.TriggerPipeline(ctx, event.RepositoryID, pipelineID, variables)
	}

	return nil
}

// HandleBranchEvent 处理分支事件
func (c *webhookClient) HandleBranchEvent(ctx context.Context, event *GitBranchEvent) error {
	c.logger.Info("处理Git分支事件", 
		zap.String("repository_id", event.RepositoryID.String()),
		zap.String("action", event.Action),
		zap.String("branch", event.BranchName))

	// 检查过滤规则
	if !c.matchesFilterRules("branch", event) {
		c.logger.Debug("事件不匹配过滤规则，跳过处理")
		return nil
	}

	// 触发监听器
	if err := c.triggerListeners(ctx, "branch", event); err != nil {
		c.logger.Error("触发事件监听器失败", zap.Error(err))
	}

	// 对于分支创建事件，可能需要触发初始化流水线
	if event.Action == "created" && c.config.AutoTrigger {
		variables := map[string]interface{}{
			"EVENT_TYPE":     "branch_created",
			"BRANCH_NAME":    event.BranchName,
			"REPOSITORY_ID":  event.RepositoryID.String(),
			"SHA":           event.SHA,
			"SENDER":        event.Sender.Username,
		}

		// 合并默认变量
		for k, v := range c.config.DefaultVariables {
			variables[k] = v
		}

		if c.config.DefaultPipelineID != uuid.Nil {
			return c.TriggerPipeline(ctx, event.RepositoryID, c.config.DefaultPipelineID, variables)
		}
	}

	return nil
}

// HandleTagEvent 处理标签事件
func (c *webhookClient) HandleTagEvent(ctx context.Context, event *GitTagEvent) error {
	c.logger.Info("处理Git标签事件", 
		zap.String("repository_id", event.RepositoryID.String()),
		zap.String("action", event.Action),
		zap.String("tag", event.TagName))

	// 检查过滤规则
	if !c.matchesFilterRules("tag", event) {
		c.logger.Debug("事件不匹配过滤规则，跳过处理")
		return nil
	}

	// 触发监听器
	if err := c.triggerListeners(ctx, "tag", event); err != nil {
		c.logger.Error("触发事件监听器失败", zap.Error(err))
	}

	// 标签创建通常触发发布流水线
	if event.Action == "created" && c.config.AutoTrigger {
		variables := map[string]interface{}{
			"EVENT_TYPE":    "tag_created",
			"TAG_NAME":      event.TagName,
			"REPOSITORY_ID": event.RepositoryID.String(),
			"SHA":          event.SHA,
			"SENDER":       event.Sender.Username,
			"IS_RELEASE":   c.isReleaseTag(event.TagName),
		}

		// 合并默认变量
		for k, v := range c.config.DefaultVariables {
			variables[k] = v
		}

		if c.config.DefaultPipelineID != uuid.Nil {
			return c.TriggerPipeline(ctx, event.RepositoryID, c.config.DefaultPipelineID, variables)
		}
	}

	return nil
}

// HandlePullRequestEvent 处理拉取请求事件
func (c *webhookClient) HandlePullRequestEvent(ctx context.Context, event *GitPullRequestEvent) error {
	c.logger.Info("处理Git拉取请求事件", 
		zap.String("repository_id", event.RepositoryID.String()),
		zap.String("action", event.Action),
		zap.Int("number", event.Number))

	// 检查过滤规则
	if !c.matchesFilterRules("pull_request", event) {
		c.logger.Debug("事件不匹配过滤规则，跳过处理")
		return nil
	}

	// 触发监听器
	if err := c.triggerListeners(ctx, "pull_request", event); err != nil {
		c.logger.Error("触发事件监听器失败", zap.Error(err))
	}

	// PR事件通常触发CI流水线
	if (event.Action == "opened" || event.Action == "synchronize") && c.config.AutoTrigger {
		variables := map[string]interface{}{
			"EVENT_TYPE":       "pull_request",
			"PR_ACTION":        event.Action,
			"PR_NUMBER":        event.Number,
			"PR_TITLE":         event.PullRequest.Title,
			"PR_BASE_BRANCH":   event.PullRequest.Base.Ref,
			"PR_HEAD_BRANCH":   event.PullRequest.Head.Ref,
			"PR_HEAD_SHA":      event.PullRequest.Head.SHA,
			"REPOSITORY_ID":    event.RepositoryID.String(),
			"SENDER":          event.Sender.Username,
		}

		// 合并默认变量
		for k, v := range c.config.DefaultVariables {
			variables[k] = v
		}

		if c.config.DefaultPipelineID != uuid.Nil {
			return c.TriggerPipeline(ctx, event.RepositoryID, c.config.DefaultPipelineID, variables)
		}
	}

	return nil
}

// TriggerPipeline 触发流水线
func (c *webhookClient) TriggerPipeline(ctx context.Context, repositoryID, pipelineID uuid.UUID, variables map[string]interface{}) error {
	c.logger.Info("触发流水线", 
		zap.String("repository_id", repositoryID.String()),
		zap.String("pipeline_id", pipelineID.String()))

	// 创建流水线运行作业
	job := &scheduler.ScheduleJob{
		JobID:         uuid.New(),
		PipelineRunID: uuid.New(), // 这里需要从Pipeline服务获取实际的RunID
		Name:          fmt.Sprintf("webhook-trigger-%s", repositoryID.String()),
		Stage:         "trigger",
		Priority:      5, // 中等优先级
		RequiredTags:  []string{}, // 可以根据仓库配置设置
		CreatedAt:     time.Now(),
		Config:        variables,
		Dependencies:  []uuid.UUID{}, // 无依赖
		MaxRetries:    3,
		RetryCount:    0,
		EstimatedDuration: 5 * time.Minute, // 预估5分钟
		ResourceRequests: &scheduler.ResourceRequests{
			CPU:    1.0,    // 1核CPU
			Memory: 512 * 1024 * 1024, // 512MB内存
			Disk:   1024 * 1024 * 1024, // 1GB磁盘
		},
	}

	// 提交作业到调度器
	if err := c.scheduler.SubmitJob(job); err != nil {
		return fmt.Errorf("提交流水线触发作业失败: %w", err)
	}

	c.logger.Info("流水线触发作业已提交", 
		zap.String("job_id", job.JobID.String()),
		zap.String("pipeline_run_id", job.PipelineRunID.String()))

	return nil
}

// RegisterEventListener 注册事件监听器
func (c *webhookClient) RegisterEventListener(eventType string, listener EventListener) error {
	if c.listeners[eventType] == nil {
		c.listeners[eventType] = make([]EventListener, 0)
	}
	
	c.listeners[eventType] = append(c.listeners[eventType], listener)
	
	c.logger.Info("注册事件监听器", 
		zap.String("event_type", eventType),
		zap.Int("listener_count", len(c.listeners[eventType])))
	
	return nil
}

// UnregisterEventListener 取消注册事件监听器
func (c *webhookClient) UnregisterEventListener(eventType string, listener EventListener) error {
	listeners, exists := c.listeners[eventType]
	if !exists {
		return fmt.Errorf("事件类型不存在: %s", eventType)
	}

	// 查找并移除监听器
	for i, l := range listeners {
		if l == listener {
			c.listeners[eventType] = append(listeners[:i], listeners[i+1:]...)
			c.logger.Info("取消注册事件监听器", zap.String("event_type", eventType))
			return nil
		}
	}

	return fmt.Errorf("监听器未找到")
}

// 辅助方法

// matchesFilterRules 检查事件是否匹配过滤规则
func (c *webhookClient) matchesFilterRules(eventType string, event interface{}) bool {
	for _, rule := range c.config.FilterRules {
		if rule.EventType != eventType {
			continue
		}

		// 根据事件类型进行具体的匹配检查
		switch eventType {
		case "push":
			if pushEvent, ok := event.(*GitPushEvent); ok {
				return c.matchesPushRule(rule, pushEvent)
			}
		case "branch":
			if branchEvent, ok := event.(*GitBranchEvent); ok {
				return c.matchesBranchRule(rule, branchEvent)
			}
		case "tag":
			if tagEvent, ok := event.(*GitTagEvent); ok {
				return c.matchesTagRule(rule, tagEvent)
			}
		case "pull_request":
			if prEvent, ok := event.(*GitPullRequestEvent); ok {
				return c.matchesPRRule(rule, prEvent)
			}
		}
	}

	// 如果没有配置过滤规则，默认匹配所有事件
	if len(c.config.FilterRules) == 0 {
		return true
	}

	return false
}

// matchesPushRule 检查推送事件是否匹配规则
func (c *webhookClient) matchesPushRule(rule FilterRule, event *GitPushEvent) bool {
	// 检查分支匹配
	if len(rule.Branches) > 0 {
		branch := event.Ref
		if !c.matchesPatterns(branch, rule.Branches) {
			return false
		}
	}

	// 检查作者匹配
	if len(rule.Authors) > 0 && event.HeadCommit != nil {
		if !c.matchesPatterns(event.HeadCommit.Author.Email, rule.Authors) {
			return false
		}
	}

	// TODO: 检查路径匹配
	
	return true
}

// matchesBranchRule 检查分支事件是否匹配规则
func (c *webhookClient) matchesBranchRule(rule FilterRule, event *GitBranchEvent) bool {
	// 检查分支匹配
	if len(rule.Branches) > 0 {
		if !c.matchesPatterns(event.BranchName, rule.Branches) {
			return false
		}
	}

	// 检查作者匹配
	if len(rule.Authors) > 0 {
		if !c.matchesPatterns(event.Sender.Email, rule.Authors) {
			return false
		}
	}

	return true
}

// matchesTagRule 检查标签事件是否匹配规则
func (c *webhookClient) matchesTagRule(rule FilterRule, event *GitTagEvent) bool {
	// 对于标签事件，使用branches字段作为标签模式匹配
	if len(rule.Branches) > 0 {
		if !c.matchesPatterns(event.TagName, rule.Branches) {
			return false
		}
	}

	// 检查作者匹配
	if len(rule.Authors) > 0 {
		if !c.matchesPatterns(event.Sender.Email, rule.Authors) {
			return false
		}
	}

	return true
}

// matchesPRRule 检查拉取请求事件是否匹配规则
func (c *webhookClient) matchesPRRule(rule FilterRule, event *GitPullRequestEvent) bool {
	// 检查目标分支匹配
	if len(rule.Branches) > 0 {
		if !c.matchesPatterns(event.PullRequest.Base.Ref, rule.Branches) {
			return false
		}
	}

	// 检查作者匹配
	if len(rule.Authors) > 0 {
		if !c.matchesPatterns(event.Sender.Email, rule.Authors) {
			return false
		}
	}

	return true
}

// matchesPatterns 检查值是否匹配模式列表
func (c *webhookClient) matchesPatterns(value string, patterns []string) bool {
	for _, pattern := range patterns {
		// 简单的通配符匹配实现
		if pattern == "*" || pattern == value {
			return true
		}
		// TODO: 实现更复杂的模式匹配
	}
	return false
}

// triggerListeners 触发事件监听器
func (c *webhookClient) triggerListeners(ctx context.Context, eventType string, event interface{}) error {
	listeners, exists := c.listeners[eventType]
	if !exists || len(listeners) == 0 {
		return nil
	}

	var errors []error
	for _, listener := range listeners {
		if err := listener.HandleEvent(ctx, event); err != nil {
			errors = append(errors, err)
			c.logger.Error("事件监听器处理失败", 
				zap.String("event_type", eventType),
				zap.Error(err))
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("部分监听器处理失败: %v", errors)
	}

	return nil
}

// buildPushVariables 构建推送事件的变量
func (c *webhookClient) buildPushVariables(event *GitPushEvent) map[string]interface{} {
	variables := map[string]interface{}{
		"EVENT_TYPE":     "push",
		"GIT_REF":        event.Ref,
		"GIT_BEFORE":     event.Before,
		"GIT_AFTER":      event.After,
		"GIT_CREATED":    event.Created,
		"GIT_DELETED":    event.Deleted,
		"GIT_FORCED":     event.Forced,
		"REPOSITORY_ID":  event.RepositoryID.String(),
		"REPOSITORY_NAME": event.Repository.Name,
		"PUSHER_NAME":    event.Pusher.Name,
		"PUSHER_EMAIL":   event.Pusher.Email,
	}

	// 提取分支名
	if event.Ref != "" {
		if branch := extractBranchName(event.Ref); branch != "" {
			variables["GIT_BRANCH"] = branch
		}
	}

	// 添加头部提交信息
	if event.HeadCommit != nil {
		variables["GIT_COMMIT_SHA"] = event.HeadCommit.ID
		variables["GIT_COMMIT_MESSAGE"] = event.HeadCommit.Message
		variables["GIT_AUTHOR_NAME"] = event.HeadCommit.Author.Name
		variables["GIT_AUTHOR_EMAIL"] = event.HeadCommit.Author.Email
	}

	// 添加提交数量
	variables["COMMIT_COUNT"] = len(event.Commits)

	// 合并自定义变量
	for k, v := range event.Variables {
		variables[k] = v
	}

	// 合并默认变量
	for k, v := range c.config.DefaultVariables {
		variables[k] = v
	}

	return variables
}

// isReleaseTag 判断是否为发布标签
func (c *webhookClient) isReleaseTag(tagName string) bool {
	// 简单的发布标签判断逻辑
	return len(tagName) > 0 && (tagName[0] == 'v' || 
		contains(tagName, "release") || 
		contains(tagName, "stable"))
}

// extractBranchName 从引用中提取分支名
func extractBranchName(ref string) string {
	const branchPrefix = "refs/heads/"
	if len(ref) > len(branchPrefix) && ref[:len(branchPrefix)] == branchPrefix {
		return ref[len(branchPrefix):]
	}
	return ""
}

// contains 检查字符串是否包含子字符串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || 
		(len(s) > len(substr) && findSubstring(s, substr)))
}

// findSubstring 查找子字符串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}