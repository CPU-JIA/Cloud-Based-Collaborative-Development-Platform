package webhook

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/service"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DefaultEventProcessor 默认事件处理器实现
type DefaultEventProcessor struct {
	projectRepo    repository.ProjectRepository
	projectService service.ProjectService
	logger         *zap.Logger
}

// EventMetrics 事件处理指标
type EventMetrics struct {
	EventType       string    `json:"event_type"`
	ProcessedCount  int64     `json:"processed_count"`
	FailedCount     int64     `json:"failed_count"`
	LastProcessedAt time.Time `json:"last_processed_at"`
	AverageLatency  float64   `json:"average_latency_ms"`
}

// NewDefaultEventProcessor 创建默认事件处理器
func NewDefaultEventProcessor(
	projectRepo repository.ProjectRepository,
	projectService service.ProjectService,
	logger *zap.Logger,
) *DefaultEventProcessor {
	return &DefaultEventProcessor{
		projectRepo:    projectRepo,
		projectService: projectService,
		logger:         logger,
	}
}

// ProcessRepositoryEvent 处理仓库事件
func (p *DefaultEventProcessor) ProcessRepositoryEvent(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	startTime := time.Now()
	defer func() {
		p.logger.Info("仓库事件处理完成",
			zap.String("event_id", event.EventID),
			zap.String("action", payload.Action),
			zap.String("repository_id", payload.Repository.ID),
			zap.Duration("duration", time.Since(startTime)))
	}()

	switch payload.Action {
	case "created":
		return p.handleRepositoryCreated(ctx, event, payload)
	case "updated":
		return p.handleRepositoryUpdated(ctx, event, payload)
	case "deleted":
		return p.handleRepositoryDeleted(ctx, event, payload)
	case "archived":
		return p.handleRepositoryArchived(ctx, event, payload)
	case "unarchived":
		return p.handleRepositoryUnarchived(ctx, event, payload)
	default:
		p.logger.Info("忽略未处理的仓库事件",
			zap.String("action", payload.Action),
			zap.String("event_id", event.EventID))
		return nil
	}
}

// ProcessBranchEvent 处理分支事件
func (p *DefaultEventProcessor) ProcessBranchEvent(ctx context.Context, event *GitEvent, payload *BranchEvent) error {
	startTime := time.Now()
	defer func() {
		p.logger.Info("分支事件处理完成",
			zap.String("event_id", event.EventID),
			zap.String("action", payload.Action),
			zap.String("branch_name", payload.Branch.Name),
			zap.Duration("duration", time.Since(startTime)))
	}()

	switch payload.Action {
	case "created":
		return p.handleBranchCreated(ctx, event, payload)
	case "deleted":
		return p.handleBranchDeleted(ctx, event, payload)
	case "default_changed":
		return p.handleDefaultBranchChanged(ctx, event, payload)
	default:
		p.logger.Info("忽略未处理的分支事件",
			zap.String("action", payload.Action),
			zap.String("event_id", event.EventID))
		return nil
	}
}

// ProcessCommitEvent 处理提交事件
func (p *DefaultEventProcessor) ProcessCommitEvent(ctx context.Context, event *GitEvent, payload *CommitEvent) error {
	startTime := time.Now()
	defer func() {
		p.logger.Info("提交事件处理完成",
			zap.String("event_id", event.EventID),
			zap.String("action", payload.Action),
			zap.String("commit_sha", payload.Commit.SHA),
			zap.Duration("duration", time.Since(startTime)))
	}()

	switch payload.Action {
	case "created":
		return p.handleCommitCreated(ctx, event, payload)
	default:
		p.logger.Info("忽略未处理的提交事件",
			zap.String("action", payload.Action),
			zap.String("event_id", event.EventID))
		return nil
	}
}

// ProcessPushEvent 处理推送事件
func (p *DefaultEventProcessor) ProcessPushEvent(ctx context.Context, event *GitEvent, payload *PushEvent) error {
	startTime := time.Now()
	defer func() {
		p.logger.Info("推送事件处理完成",
			zap.String("event_id", event.EventID),
			zap.String("repository_id", payload.RepositoryID),
			zap.String("branch", payload.Branch),
			zap.Int("commits_count", len(payload.Commits)),
			zap.Duration("duration", time.Since(startTime)))
	}()

	return p.handlePushEvent(ctx, event, payload)
}

// ProcessTagEvent 处理标签事件
func (p *DefaultEventProcessor) ProcessTagEvent(ctx context.Context, event *GitEvent, payload *TagEvent) error {
	startTime := time.Now()
	defer func() {
		p.logger.Info("标签事件处理完成",
			zap.String("event_id", event.EventID),
			zap.String("action", payload.Action),
			zap.String("tag_name", payload.Tag.Name),
			zap.Duration("duration", time.Since(startTime)))
	}()

	switch payload.Action {
	case "created":
		return p.handleTagCreated(ctx, event, payload)
	case "deleted":
		return p.handleTagDeleted(ctx, event, payload)
	default:
		p.logger.Info("忽略未处理的标签事件",
			zap.String("action", payload.Action),
			zap.String("event_id", event.EventID))
		return nil
	}
}

// 仓库事件处理方法

func (p *DefaultEventProcessor) handleRepositoryCreated(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	p.logger.Info("处理仓库创建事件",
		zap.String("repository_id", payload.Repository.ID),
		zap.String("repository_name", payload.Repository.Name),
		zap.String("project_id", payload.Repository.ProjectID))

	// 验证项目存在
	projectID, err := uuid.Parse(payload.Repository.ProjectID)
	if err != nil {
		return fmt.Errorf("无效的项目ID: %w", err)
	}

	// 这里可以添加项目活动记录、通知等逻辑
	// 例如：记录项目活动日志
	return p.recordProjectActivity(ctx, projectID, "repository_created", map[string]interface{}{
		"repository_id":   payload.Repository.ID,
		"repository_name": payload.Repository.Name,
		"event_id":        event.EventID,
	})
}

func (p *DefaultEventProcessor) handleRepositoryUpdated(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	p.logger.Info("处理仓库更新事件",
		zap.String("repository_id", payload.Repository.ID),
		zap.String("repository_name", payload.Repository.Name))

	projectID, err := uuid.Parse(payload.Repository.ProjectID)
	if err != nil {
		return fmt.Errorf("无效的项目ID: %w", err)
	}

	return p.recordProjectActivity(ctx, projectID, "repository_updated", map[string]interface{}{
		"repository_id":   payload.Repository.ID,
		"repository_name": payload.Repository.Name,
		"event_id":        event.EventID,
	})
}

func (p *DefaultEventProcessor) handleRepositoryDeleted(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	p.logger.Info("处理仓库删除事件",
		zap.String("repository_id", payload.Repository.ID),
		zap.String("repository_name", payload.Repository.Name))

	projectID, err := uuid.Parse(payload.Repository.ProjectID)
	if err != nil {
		return fmt.Errorf("无效的项目ID: %w", err)
	}

	return p.recordProjectActivity(ctx, projectID, "repository_deleted", map[string]interface{}{
		"repository_id":   payload.Repository.ID,
		"repository_name": payload.Repository.Name,
		"event_id":        event.EventID,
	})
}

func (p *DefaultEventProcessor) handleRepositoryArchived(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	p.logger.Info("处理仓库归档事件",
		zap.String("repository_id", payload.Repository.ID))

	projectID, err := uuid.Parse(payload.Repository.ProjectID)
	if err != nil {
		return fmt.Errorf("无效的项目ID: %w", err)
	}

	return p.recordProjectActivity(ctx, projectID, "repository_archived", map[string]interface{}{
		"repository_id": payload.Repository.ID,
		"event_id":      event.EventID,
	})
}

func (p *DefaultEventProcessor) handleRepositoryUnarchived(ctx context.Context, event *GitEvent, payload *RepositoryEvent) error {
	p.logger.Info("处理仓库取消归档事件",
		zap.String("repository_id", payload.Repository.ID))

	projectID, err := uuid.Parse(payload.Repository.ProjectID)
	if err != nil {
		return fmt.Errorf("无效的项目ID: %w", err)
	}

	return p.recordProjectActivity(ctx, projectID, "repository_unarchived", map[string]interface{}{
		"repository_id": payload.Repository.ID,
		"event_id":      event.EventID,
	})
}

// 分支事件处理方法

func (p *DefaultEventProcessor) handleBranchCreated(ctx context.Context, event *GitEvent, payload *BranchEvent) error {
	p.logger.Info("处理分支创建事件",
		zap.String("branch_name", payload.Branch.Name),
		zap.String("repository_id", payload.Branch.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Branch.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	// 通过仓库ID获取项目ID（这里需要从Git网关查询）
	// 暂时记录到仓库级别的活动
	return p.recordRepositoryActivity(ctx, repositoryID, "branch_created", map[string]interface{}{
		"branch_name": payload.Branch.Name,
		"commit":      payload.Branch.Commit,
		"event_id":    event.EventID,
	})
}

func (p *DefaultEventProcessor) handleBranchDeleted(ctx context.Context, event *GitEvent, payload *BranchEvent) error {
	p.logger.Info("处理分支删除事件",
		zap.String("branch_name", payload.Branch.Name),
		zap.String("repository_id", payload.Branch.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Branch.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "branch_deleted", map[string]interface{}{
		"branch_name": payload.Branch.Name,
		"event_id":    event.EventID,
	})
}

func (p *DefaultEventProcessor) handleDefaultBranchChanged(ctx context.Context, event *GitEvent, payload *BranchEvent) error {
	p.logger.Info("处理默认分支变更事件",
		zap.String("new_default_branch", payload.Branch.Name),
		zap.String("repository_id", payload.Branch.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Branch.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "default_branch_changed", map[string]interface{}{
		"new_default_branch": payload.Branch.Name,
		"event_id":           event.EventID,
	})
}

// 提交事件处理方法

func (p *DefaultEventProcessor) handleCommitCreated(ctx context.Context, event *GitEvent, payload *CommitEvent) error {
	p.logger.Info("处理提交创建事件",
		zap.String("commit_sha", payload.Commit.SHA),
		zap.String("author", payload.Commit.Author),
		zap.String("repository_id", payload.Commit.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Commit.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "commit_created", map[string]interface{}{
		"commit_sha":     payload.Commit.SHA,
		"commit_message": payload.Commit.Message,
		"author":         payload.Commit.Author,
		"branch":         payload.Commit.Branch,
		"event_id":       event.EventID,
	})
}

// 推送事件处理方法

func (p *DefaultEventProcessor) handlePushEvent(ctx context.Context, event *GitEvent, payload *PushEvent) error {
	p.logger.Info("处理推送事件",
		zap.String("repository_id", payload.RepositoryID),
		zap.String("branch", payload.Branch),
		zap.String("pusher", payload.Pusher),
		zap.Int("commits_count", len(payload.Commits)))

	repositoryID, err := uuid.Parse(payload.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	// 记录推送活动
	activityData := map[string]interface{}{
		"branch":        payload.Branch,
		"pusher":        payload.Pusher,
		"before":        payload.Before,
		"after":         payload.After,
		"commits_count": len(payload.Commits),
		"event_id":      event.EventID,
	}

	// 如果有提交信息，添加最新提交的摘要
	if len(payload.Commits) > 0 {
		latestCommit := payload.Commits[len(payload.Commits)-1]
		activityData["latest_commit"] = map[string]string{
			"sha":     latestCommit.SHA,
			"message": latestCommit.Message,
			"author":  latestCommit.Author,
		}
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "push", activityData)
}

// 标签事件处理方法

func (p *DefaultEventProcessor) handleTagCreated(ctx context.Context, event *GitEvent, payload *TagEvent) error {
	p.logger.Info("处理标签创建事件",
		zap.String("tag_name", payload.Tag.Name),
		zap.String("repository_id", payload.Tag.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Tag.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "tag_created", map[string]interface{}{
		"tag_name": payload.Tag.Name,
		"target":   payload.Tag.Target,
		"message":  payload.Tag.Message,
		"event_id": event.EventID,
	})
}

func (p *DefaultEventProcessor) handleTagDeleted(ctx context.Context, event *GitEvent, payload *TagEvent) error {
	p.logger.Info("处理标签删除事件",
		zap.String("tag_name", payload.Tag.Name),
		zap.String("repository_id", payload.Tag.RepositoryID))

	repositoryID, err := uuid.Parse(payload.Tag.RepositoryID)
	if err != nil {
		return fmt.Errorf("无效的仓库ID: %w", err)
	}

	return p.recordRepositoryActivity(ctx, repositoryID, "tag_deleted", map[string]interface{}{
		"tag_name": payload.Tag.Name,
		"event_id": event.EventID,
	})
}

// 辅助方法

// recordProjectActivity 记录项目活动
func (p *DefaultEventProcessor) recordProjectActivity(ctx context.Context, projectID uuid.UUID, activityType string, data map[string]interface{}) error {
	// 这里可以实现项目活动记录逻辑
	// 例如：插入到项目活动表、发送通知等

	p.logger.Info("记录项目活动",
		zap.String("project_id", projectID.String()),
		zap.String("activity_type", activityType),
		zap.Any("activity_data", data))

	// TODO: 实现实际的活动记录逻辑
	// 可能需要扩展数据库schema来支持活动记录

	return nil
}

// recordRepositoryActivity 记录仓库活动
func (p *DefaultEventProcessor) recordRepositoryActivity(ctx context.Context, repositoryID uuid.UUID, activityType string, data map[string]interface{}) error {
	// 这里可以实现仓库活动记录逻辑

	p.logger.Info("记录仓库活动",
		zap.String("repository_id", repositoryID.String()),
		zap.String("activity_type", activityType),
		zap.Any("activity_data", data))

	// TODO: 实现实际的活动记录逻辑

	return nil
}
