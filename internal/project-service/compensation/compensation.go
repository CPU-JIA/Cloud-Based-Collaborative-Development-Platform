package compensation

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// CompensationAction 补偿动作类型
type CompensationAction string

const (
	CompensationActionDeleteRepository CompensationAction = "delete_repository"
	CompensationActionRollbackProject  CompensationAction = "rollback_project"
	CompensationActionNotifyFailure    CompensationAction = "notify_failure"
)

// CompensationEntry 补偿记录
type CompensationEntry struct {
	ID         uuid.UUID              `json:"id"`
	Action     CompensationAction     `json:"action"`
	ResourceID uuid.UUID              `json:"resource_id"`
	Payload    map[string]interface{} `json:"payload"`
	CreatedAt  time.Time              `json:"created_at"`
	ExecutedAt *time.Time             `json:"executed_at,omitempty"`
	Status     string                 `json:"status"` // pending, executed, failed
	RetryCount int                    `json:"retry_count"`
	MaxRetries int                    `json:"max_retries"`
	LastError  string                 `json:"last_error,omitempty"`
}

// CompensationManager 补偿管理器
type CompensationManager struct {
	gitClient client.GitGatewayClient
	logger    *zap.Logger
	entries   []CompensationEntry // 在实际应用中应使用持久化存储
}

// NewCompensationManager 创建补偿管理器
func NewCompensationManager(gitClient client.GitGatewayClient, logger *zap.Logger) *CompensationManager {
	return &CompensationManager{
		gitClient: gitClient,
		logger:    logger,
		entries:   make([]CompensationEntry, 0),
	}
}

// AddCompensation 添加补偿动作
func (cm *CompensationManager) AddCompensation(action CompensationAction, resourceID uuid.UUID, payload map[string]interface{}) uuid.UUID {
	entry := CompensationEntry{
		ID:         uuid.New(),
		Action:     action,
		ResourceID: resourceID,
		Payload:    payload,
		CreatedAt:  time.Now(),
		Status:     "pending",
		RetryCount: 0,
		MaxRetries: 3,
	}

	cm.entries = append(cm.entries, entry)

	cm.logger.Info("添加补偿动作",
		zap.String("compensation_id", entry.ID.String()),
		zap.String("action", string(action)),
		zap.String("resource_id", resourceID.String()))

	return entry.ID
}

// ExecuteCompensation 执行补偿动作
func (cm *CompensationManager) ExecuteCompensation(ctx context.Context, compensationID uuid.UUID) error {
	// 查找补偿记录
	entryIndex := -1
	for i, entry := range cm.entries {
		if entry.ID == compensationID {
			entryIndex = i
			break
		}
	}

	if entryIndex == -1 {
		return fmt.Errorf("补偿记录不存在: %s", compensationID.String())
	}

	entry := &cm.entries[entryIndex]

	// 检查状态
	if entry.Status == "executed" {
		cm.logger.Info("补偿动作已执行", zap.String("compensation_id", compensationID.String()))
		return nil
	}

	// 检查重试次数
	if entry.RetryCount >= entry.MaxRetries {
		entry.Status = "failed"
		cm.logger.Error("补偿动作超过最大重试次数",
			zap.String("compensation_id", compensationID.String()),
			zap.Int("retry_count", entry.RetryCount))
		return fmt.Errorf("补偿动作超过最大重试次数")
	}

	// 执行补偿动作
	entry.RetryCount++

	cm.logger.Info("执行补偿动作",
		zap.String("compensation_id", compensationID.String()),
		zap.String("action", string(entry.Action)),
		zap.Int("retry_count", entry.RetryCount))

	var err error
	switch entry.Action {
	case CompensationActionDeleteRepository:
		err = cm.executeDeleteRepository(ctx, entry)
	case CompensationActionRollbackProject:
		err = cm.executeRollbackProject(ctx, entry)
	case CompensationActionNotifyFailure:
		err = cm.executeNotifyFailure(ctx, entry)
	default:
		err = fmt.Errorf("未知的补偿动作: %s", entry.Action)
	}

	if err != nil {
		entry.LastError = err.Error()
		cm.logger.Error("补偿动作执行失败",
			zap.String("compensation_id", compensationID.String()),
			zap.Error(err))
		return err
	}

	// 标记为已执行
	now := time.Now()
	entry.ExecutedAt = &now
	entry.Status = "executed"

	cm.logger.Info("补偿动作执行成功",
		zap.String("compensation_id", compensationID.String()),
		zap.String("action", string(entry.Action)))

	return nil
}

// ExecuteAllPendingCompensations 执行所有待处理的补偿动作
func (cm *CompensationManager) ExecuteAllPendingCompensations(ctx context.Context) error {
	pendingEntries := make([]uuid.UUID, 0)

	for _, entry := range cm.entries {
		if entry.Status == "pending" {
			pendingEntries = append(pendingEntries, entry.ID)
		}
	}

	cm.logger.Info("开始执行待处理的补偿动作", zap.Int("count", len(pendingEntries)))

	var lastError error
	successCount := 0

	for _, compensationID := range pendingEntries {
		if err := cm.ExecuteCompensation(ctx, compensationID); err != nil {
			lastError = err
			cm.logger.Error("补偿动作执行失败",
				zap.String("compensation_id", compensationID.String()),
				zap.Error(err))
		} else {
			successCount++
		}
	}

	cm.logger.Info("补偿动作执行完成",
		zap.Int("total", len(pendingEntries)),
		zap.Int("success", successCount),
		zap.Int("failed", len(pendingEntries)-successCount))

	return lastError
}

// executeDeleteRepository 执行删除仓库补偿
func (cm *CompensationManager) executeDeleteRepository(ctx context.Context, entry *CompensationEntry) error {
	repositoryID := entry.ResourceID

	cm.logger.Info("执行删除仓库补偿",
		zap.String("repository_id", repositoryID.String()))

	// 调用Git网关删除仓库
	if err := cm.gitClient.DeleteRepository(ctx, repositoryID); err != nil {
		return fmt.Errorf("删除Git仓库失败: %w", err)
	}

	return nil
}

// executeRollbackProject 执行回滚项目补偿
func (cm *CompensationManager) executeRollbackProject(ctx context.Context, entry *CompensationEntry) error {
	projectID := entry.ResourceID

	cm.logger.Info("执行回滚项目补偿",
		zap.String("project_id", projectID.String()))

	// 这里应该调用项目服务的回滚逻辑
	// 例如：删除项目记录、清理相关数据等
	// 由于这是演示，我们只记录日志
	cm.logger.Info("项目回滚补偿执行完成",
		zap.String("project_id", projectID.String()))

	return nil
}

// executeNotifyFailure 执行失败通知补偿
func (cm *CompensationManager) executeNotifyFailure(ctx context.Context, entry *CompensationEntry) error {
	cm.logger.Info("执行失败通知补偿",
		zap.String("resource_id", entry.ResourceID.String()),
		zap.Any("payload", entry.Payload))

	// 这里应该发送通知（邮件、消息队列等）
	// 由于这是演示，我们只记录日志
	payloadBytes, _ := json.Marshal(entry.Payload)
	cm.logger.Info("失败通知已发送",
		zap.String("resource_id", entry.ResourceID.String()),
		zap.String("payload", string(payloadBytes)))

	return nil
}

// GetCompensationStatus 获取补偿状态
func (cm *CompensationManager) GetCompensationStatus(compensationID uuid.UUID) (*CompensationEntry, error) {
	for _, entry := range cm.entries {
		if entry.ID == compensationID {
			return &entry, nil
		}
	}
	return nil, fmt.Errorf("补偿记录不存在: %s", compensationID.String())
}

// ListPendingCompensations 列出待处理的补偿动作
func (cm *CompensationManager) ListPendingCompensations() []CompensationEntry {
	pending := make([]CompensationEntry, 0)
	for _, entry := range cm.entries {
		if entry.Status == "pending" {
			pending = append(pending, entry)
		}
	}
	return pending
}

// ClearExecutedCompensations 清理已执行的补偿记录
func (cm *CompensationManager) ClearExecutedCompensations() int {
	newEntries := make([]CompensationEntry, 0)
	removedCount := 0

	for _, entry := range cm.entries {
		if entry.Status == "executed" {
			removedCount++
		} else {
			newEntries = append(newEntries, entry)
		}
	}

	cm.entries = newEntries

	cm.logger.Info("清理已执行的补偿记录", zap.Int("removed_count", removedCount))

	return removedCount
}
