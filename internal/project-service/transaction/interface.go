package transaction

import (
	"context"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
)

// TransactionManager 事务管理器接口
type TransactionManager interface {
	// CreateRepositoryTransaction 创建仓库事务
	CreateRepositoryTransaction(
		ctx context.Context,
		projectID, userID, tenantID uuid.UUID,
		createReq *client.CreateRepositoryRequest,
	) (*models.Repository, error)

	// GetTransaction 获取事务信息
	GetTransaction(transactionID uuid.UUID) (*DistributedTransaction, error)

	// ListActiveTransactions 列出活跃事务
	ListActiveTransactions() []*DistributedTransaction

	// CleanupCompletedTransactions 清理已完成的事务
	CleanupCompletedTransactions() int
}