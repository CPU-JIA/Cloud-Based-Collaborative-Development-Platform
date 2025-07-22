package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/compensation"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/repository"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TransactionPhase 事务阶段
type TransactionPhase string

const (
	PhaseValidation TransactionPhase = "validation"  // 验证阶段
	PhaseExecution  TransactionPhase = "execution"   // 执行阶段
	PhaseConfirm    TransactionPhase = "confirm"     // 确认阶段
	PhaseCancel     TransactionPhase = "cancel"      // 取消阶段
)

// TransactionStatus 事务状态
type TransactionStatus string

const (
	StatusPending   TransactionStatus = "pending"   // 待处理
	StatusValidated TransactionStatus = "validated" // 已验证
	StatusExecuted  TransactionStatus = "executed"  // 已执行
	StatusConfirmed TransactionStatus = "confirmed" // 已确认
	StatusCancelled TransactionStatus = "cancelled" // 已取消
	StatusFailed    TransactionStatus = "failed"    // 失败
)

// DistributedTransaction 分布式事务
type DistributedTransaction struct {
	ID              uuid.UUID         `json:"id"`
	Type            string            `json:"type"`
	Status          TransactionStatus `json:"status"`
	CurrentPhase    TransactionPhase  `json:"current_phase"`
	ProjectID       uuid.UUID         `json:"project_id"`
	UserID          uuid.UUID         `json:"user_id"`
	TenantID        uuid.UUID         `json:"tenant_id"`
	Payload         map[string]interface{} `json:"payload"`
	CompensationIDs []uuid.UUID       `json:"compensation_ids"`
	CreatedAt       time.Time         `json:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at"`
	CompletedAt     *time.Time        `json:"completed_at,omitempty"`
	ErrorMessage    string            `json:"error_message,omitempty"`
}

// DistributedTransactionManager 分布式事务管理器
type DistributedTransactionManager struct {
	projectRepo       repository.ProjectRepository
	gitClient         client.GitGatewayClient
	compensationMgr   *compensation.CompensationManager
	logger            *zap.Logger
	transactions      map[uuid.UUID]*DistributedTransaction // 在实际应用中应使用持久化存储
}

// 确保实现了TransactionManager接口
var _ TransactionManager = (*DistributedTransactionManager)(nil)

// NewDistributedTransactionManager 创建分布式事务管理器
func NewDistributedTransactionManager(
	projectRepo repository.ProjectRepository,
	gitClient client.GitGatewayClient,
	compensationMgr *compensation.CompensationManager,
	logger *zap.Logger,
) *DistributedTransactionManager {
	return &DistributedTransactionManager{
		projectRepo:     projectRepo,
		gitClient:       gitClient,
		compensationMgr: compensationMgr,
		logger:          logger,
		transactions:    make(map[uuid.UUID]*DistributedTransaction),
	}
}

// CreateRepositoryTransaction 创建仓库事务
func (dtm *DistributedTransactionManager) CreateRepositoryTransaction(
	ctx context.Context,
	projectID, userID, tenantID uuid.UUID,
	createReq *client.CreateRepositoryRequest,
) (*models.Repository, error) {
	// 创建分布式事务
	tx := &DistributedTransaction{
		ID:           uuid.New(),
		Type:         "create_repository",
		Status:       StatusPending,
		CurrentPhase: PhaseValidation,
		ProjectID:    projectID,
		UserID:       userID,
		TenantID:     tenantID,
		Payload: map[string]interface{}{
			"repository_name": createReq.Name,
			"description":     createReq.Description,
			"visibility":      createReq.Visibility,
			"default_branch":  createReq.DefaultBranch,
			"init_readme":     createReq.InitReadme,
		},
		CompensationIDs: make([]uuid.UUID, 0),
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	dtm.transactions[tx.ID] = tx

	dtm.logger.Info("开始创建仓库分布式事务",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("project_id", projectID.String()),
		zap.String("repository_name", createReq.Name))

	// 阶段1：验证
	if err := dtm.validateCreateRepository(ctx, tx, createReq); err != nil {
		return nil, dtm.failTransaction(tx, fmt.Errorf("验证阶段失败: %w", err))
	}

	// 阶段2：执行
	repository, err := dtm.executeCreateRepository(ctx, tx, createReq)
	if err != nil {
		return nil, dtm.failTransaction(tx, fmt.Errorf("执行阶段失败: %w", err))
	}

	// 阶段3：确认
	if err := dtm.confirmCreateRepository(ctx, tx, repository); err != nil {
		// 确认失败，需要执行补偿
		dtm.executeCompensations(ctx, tx)
		return nil, dtm.failTransaction(tx, fmt.Errorf("确认阶段失败: %w", err))
	}

	// 事务成功完成
	now := time.Now()
	tx.Status = StatusConfirmed
	tx.CompletedAt = &now
	tx.UpdatedAt = now

	dtm.logger.Info("创建仓库分布式事务完成",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("repository_id", repository.ID.String()))

	return repository, nil
}

// validateCreateRepository 验证创建仓库
func (dtm *DistributedTransactionManager) validateCreateRepository(
	ctx context.Context,
	tx *DistributedTransaction,
	createReq *client.CreateRepositoryRequest,
) error {
	dtm.logger.Info("验证创建仓库请求",
		zap.String("transaction_id", tx.ID.String()))

	// 1. 验证项目存在性和权限
	project, err := dtm.projectRepo.GetByID(ctx, tx.ProjectID, tx.TenantID)
	if err != nil {
		return fmt.Errorf("获取项目失败: %w", err)
	}

	// 2. 检查用户权限
	hasAccess, err := dtm.projectRepo.CheckUserAccess(ctx, tx.ProjectID, tx.UserID)
	if err != nil {
		return fmt.Errorf("检查用户权限失败: %w", err)
	}
	if !hasAccess {
		return fmt.Errorf("用户无权限在此项目中创建仓库")
	}

	// 3. 验证仓库名称唯一性（简化实现）
	if createReq.Name == "" {
		return fmt.Errorf("仓库名称不能为空")
	}

	// 更新事务状态
	tx.Status = StatusValidated
	tx.CurrentPhase = PhaseExecution
	tx.UpdatedAt = time.Now()

	dtm.logger.Info("创建仓库验证完成",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("project_name", project.Name))

	return nil
}

// executeCreateRepository 执行创建仓库
func (dtm *DistributedTransactionManager) executeCreateRepository(
	ctx context.Context,
	tx *DistributedTransaction,
	createReq *client.CreateRepositoryRequest,
) (*models.Repository, error) {
	dtm.logger.Info("执行创建仓库",
		zap.String("transaction_id", tx.ID.String()))

	// 调用Git网关创建仓库
	gitRepo, err := dtm.gitClient.CreateRepository(ctx, createReq)
	if err != nil {
		return nil, fmt.Errorf("Git网关创建仓库失败: %w", err)
	}

	// 添加删除仓库的补偿动作
	compensationID := dtm.compensationMgr.AddCompensation(
		compensation.CompensationActionDeleteRepository,
		gitRepo.ID,
		map[string]interface{}{
			"repository_name": gitRepo.Name,
			"project_id":      gitRepo.ProjectID.String(),
		},
	)
	tx.CompensationIDs = append(tx.CompensationIDs, compensationID)

	// 转换为项目服务的仓库模型
	repository := &models.Repository{
		ID:            gitRepo.ID,
		ProjectID:     gitRepo.ProjectID,
		Name:          gitRepo.Name,
		Description:   gitRepo.Description,
		Visibility:    string(gitRepo.Visibility),
		DefaultBranch: gitRepo.DefaultBranch,
	}

	// 更新事务状态
	tx.Status = StatusExecuted
	tx.CurrentPhase = PhaseConfirm
	tx.UpdatedAt = time.Now()

	dtm.logger.Info("创建仓库执行完成",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("repository_id", gitRepo.ID.String()))

	return repository, nil
}

// confirmCreateRepository 确认创建仓库
func (dtm *DistributedTransactionManager) confirmCreateRepository(
	ctx context.Context,
	tx *DistributedTransaction,
	repository *models.Repository,
) error {
	dtm.logger.Info("确认创建仓库",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("repository_id", repository.ID.String()))

	// 在实际应用中，这里可能需要：
	// 1. 更新项目的仓库关联关系
	// 2. 记录操作日志
	// 3. 发送通知
	// 4. 清理临时数据
	
	// 由于我们的项目服务不直接存储仓库信息（通过Git网关获取），
	// 这里主要是验证操作是否成功
	
	// 验证仓库是否真的创建成功
	verifyRepo, err := dtm.gitClient.GetRepository(ctx, repository.ID)
	if err != nil {
		return fmt.Errorf("验证仓库创建失败: %w", err)
	}
	
	if verifyRepo.Name != repository.Name {
		return fmt.Errorf("仓库验证失败：名称不匹配")
	}

	dtm.logger.Info("创建仓库确认完成",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("repository_name", repository.Name))

	return nil
}

// failTransaction 失败事务
func (dtm *DistributedTransactionManager) failTransaction(tx *DistributedTransaction, err error) error {
	tx.Status = StatusFailed
	tx.ErrorMessage = err.Error()
	now := time.Now()
	tx.CompletedAt = &now
	tx.UpdatedAt = now

	dtm.logger.Error("分布式事务失败",
		zap.String("transaction_id", tx.ID.String()),
		zap.String("type", tx.Type),
		zap.Error(err))

	// 执行补偿动作
	if len(tx.CompensationIDs) > 0 {
		dtm.executeCompensations(context.Background(), tx)
	}

	return err
}

// executeCompensations 执行补偿动作
func (dtm *DistributedTransactionManager) executeCompensations(ctx context.Context, tx *DistributedTransaction) {
	dtm.logger.Info("开始执行补偿动作",
		zap.String("transaction_id", tx.ID.String()),
		zap.Int("compensation_count", len(tx.CompensationIDs)))

	for _, compensationID := range tx.CompensationIDs {
		if err := dtm.compensationMgr.ExecuteCompensation(ctx, compensationID); err != nil {
			dtm.logger.Error("补偿动作执行失败",
				zap.String("transaction_id", tx.ID.String()),
				zap.String("compensation_id", compensationID.String()),
				zap.Error(err))
		}
	}
}

// GetTransaction 获取事务信息
func (dtm *DistributedTransactionManager) GetTransaction(transactionID uuid.UUID) (*DistributedTransaction, error) {
	tx, exists := dtm.transactions[transactionID]
	if !exists {
		return nil, fmt.Errorf("事务不存在: %s", transactionID.String())
	}
	return tx, nil
}

// ListActiveTransactions 列出活跃事务
func (dtm *DistributedTransactionManager) ListActiveTransactions() []*DistributedTransaction {
	active := make([]*DistributedTransaction, 0)
	for _, tx := range dtm.transactions {
		if tx.Status == StatusPending || tx.Status == StatusValidated || tx.Status == StatusExecuted {
			active = append(active, tx)
		}
	}
	return active
}

// CleanupCompletedTransactions 清理已完成的事务
func (dtm *DistributedTransactionManager) CleanupCompletedTransactions() int {
	cleaned := 0
	for id, tx := range dtm.transactions {
		if tx.Status == StatusConfirmed || tx.Status == StatusFailed {
			if tx.CompletedAt != nil && time.Since(*tx.CompletedAt) > 24*time.Hour {
				delete(dtm.transactions, id)
				cleaned++
			}
		}
	}

	dtm.logger.Info("清理已完成的事务", zap.Int("cleaned_count", cleaned))
	return cleaned
}