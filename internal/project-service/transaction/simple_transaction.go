package transaction

import (
	"context"
	"fmt"
	"time"

	"github.com/cloud-platform/collaborative-dev/internal/project-service/client"
	"github.com/cloud-platform/collaborative-dev/internal/project-service/models"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// EventType 事件类型
type EventType string

const (
	EventProjectCreated    EventType = "project.created"
	EventProjectUpdated    EventType = "project.updated"
	EventRepositoryCreated EventType = "repository.created"
	EventRepositoryDeleted EventType = "repository.deleted"
	EventUserInvited       EventType = "user.invited"
)

// DomainEvent 领域事件
type DomainEvent struct {
	ID          uuid.UUID              `json:"id" gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
	Type        EventType              `json:"type" gorm:"not null"`
	AggregateID uuid.UUID              `json:"aggregate_id" gorm:"type:uuid;not null"`
	TenantID    uuid.UUID              `json:"tenant_id" gorm:"type:uuid;not null;index"`
	UserID      *uuid.UUID             `json:"user_id,omitempty" gorm:"type:uuid"`
	Payload     map[string]interface{} `json:"payload" gorm:"type:jsonb"`
	Timestamp   time.Time              `json:"timestamp" gorm:"not null;default:current_timestamp"`
	Processed   bool                   `json:"processed" gorm:"default:false;index"`
	Error       string                 `json:"error,omitempty"`
	RetryCount  int                    `json:"retry_count" gorm:"default:0"`
	CreatedAt   time.Time              `json:"created_at" gorm:"autoCreateTime"`
	UpdatedAt   time.Time              `json:"updated_at" gorm:"autoUpdateTime"`
}

// EventBus 事件总线接口
type EventBus interface {
	Publish(ctx context.Context, event *DomainEvent) error
	Subscribe(eventType EventType, handler EventHandler) error
}

// EventHandler 事件处理器
type EventHandler func(ctx context.Context, event *DomainEvent) error

// SimpleTransactionService 简化的事务处理服务
type SimpleTransactionService struct {
	db        *gorm.DB
	eventBus  EventBus
	gitClient client.GitGatewayClient
	logger    *zap.Logger
}

// NewSimpleTransactionService 创建简化事务服务
func NewSimpleTransactionService(
	db *gorm.DB,
	eventBus EventBus,
	gitClient client.GitGatewayClient,
	logger *zap.Logger,
) *SimpleTransactionService {
	return &SimpleTransactionService{
		db:        db,
		eventBus:  eventBus,
		gitClient: gitClient,
		logger:    logger,
	}
}

// CreateProjectWithRepository 创建项目并初始化仓库
func (s *SimpleTransactionService) CreateProjectWithRepository(
	ctx context.Context,
	req *CreateProjectRequest,
) (*models.Project, error) {
	s.logger.Info("开始创建项目和仓库",
		zap.String("project_name", req.Name),
		zap.String("tenant_id", req.TenantID.String()),
		zap.String("user_id", req.UserID.String()))

	var project *models.Project

	// 数据库事务处理核心业务逻辑
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 创建项目
		project = &models.Project{
			ID:          uuid.New(),
			TenantID:    req.TenantID,
			Name:        req.Name,
			Description: &req.Description,
			Status:      "active",
			Key:         fmt.Sprintf("PROJ-%d", time.Now().Unix()),
		}

		if err := tx.Create(project).Error; err != nil {
			return fmt.Errorf("创建项目失败: %w", err)
		}

		// 2. 先创建或获取owner角色
		var ownerRole models.Role
		err := tx.Where("name = ? AND scope = 'project'", "owner").First(&ownerRole).Error
		if err != nil {
			// 如果角色不存在，创建默认owner角色
			ownerRole = models.Role{
				ID:          uuid.New(),
				TenantID:    req.TenantID,
				Name:        "owner",
				Scope:       "project",
				Permissions: []string{"project.admin", "project.read", "project.write"},
			}
			if err := tx.Create(&ownerRole).Error; err != nil {
				return fmt.Errorf("创建owner角色失败: %w", err)
			}
		}

		// 3. 创建项目成员关系
		member := &models.ProjectMember{
			ProjectID: project.ID,
			UserID:    req.UserID,
			RoleID:    ownerRole.ID,
			AddedBy:   &req.UserID,
		}

		if err := tx.Create(member).Error; err != nil {
			return fmt.Errorf("创建项目成员关系失败: %w", err)
		}

		s.logger.Info("项目创建成功",
			zap.String("project_id", project.ID.String()),
			zap.String("project_name", project.Name))

		return nil
	})

	if err != nil {
		s.logger.Error("项目创建事务失败", zap.Error(err))
		return nil, err
	}

	// 3. 异步处理其他服务调用
	go s.handleProjectCreatedAsync(context.Background(), project, req.InitRepository, req.RepositoryName, req.UserID)

	return project, nil
}

// handleProjectCreatedAsync 异步处理项目创建后的操作
func (s *SimpleTransactionService) handleProjectCreatedAsync(
	ctx context.Context,
	project *models.Project,
	initRepository bool,
	repositoryName string,
	userID uuid.UUID,
) {
	s.logger.Info("开始异步处理项目后续操作",
		zap.String("project_id", project.ID.String()),
		zap.Bool("init_repository", initRepository))

	// 发布项目创建事件
	event := &DomainEvent{
		ID:          uuid.New(),
		Type:        EventProjectCreated,
		AggregateID: project.ID,
		TenantID:    project.TenantID,
		UserID:      &userID,
		Payload: map[string]interface{}{
			"project_name": project.Name,
			"description":  project.Description,
			"status":       project.Status,
		},
		Timestamp: time.Now(),
	}

	if err := s.eventBus.Publish(ctx, event); err != nil {
		s.logger.Error("发布项目创建事件失败", zap.Error(err))
	}

	// 如果需要初始化仓库
	if initRepository {
		if err := s.createRepositoryAsync(ctx, project, repositoryName, userID); err != nil {
			s.logger.Error("异步创建仓库失败",
				zap.String("project_id", project.ID.String()),
				zap.Error(err))
		}
	}
}

// createRepositoryAsync 异步创建仓库
func (s *SimpleTransactionService) createRepositoryAsync(
	ctx context.Context,
	project *models.Project,
	repositoryName string,
	userID uuid.UUID,
) error {
	s.logger.Info("异步创建仓库",
		zap.String("project_id", project.ID.String()),
		zap.String("repository_name", repositoryName))

	// 调用Git网关创建仓库
	gitRepo, err := s.gitClient.CreateRepository(ctx, &client.CreateRepositoryRequest{
		Name:          repositoryName,
		Description:   &[]string{fmt.Sprintf("项目 %s 的主仓库", project.Name)}[0],
		ProjectID:     project.ID.String(),
		Visibility:    client.RepositoryVisibilityPrivate,
		DefaultBranch: &[]string{"main"}[0],
		InitReadme:    true,
	})

	if err != nil {
		s.logger.Error("Git网关创建仓库失败",
			zap.String("project_id", project.ID.String()),
			zap.Error(err))

		// 发布仓库创建失败事件，可以触发重试或通知
		failEvent := &DomainEvent{
			ID:          uuid.New(),
			Type:        EventType("repository.creation_failed"),
			AggregateID: project.ID,
			TenantID:    project.TenantID,
			UserID:      &userID,
			Payload: map[string]interface{}{
				"project_id":      project.ID.String(),
				"repository_name": repositoryName,
				"error":           err.Error(),
			},
			Timestamp: time.Now(),
		}

		s.eventBus.Publish(ctx, failEvent)
		return err
	}

	// 发布仓库创建成功事件
	successEvent := &DomainEvent{
		ID:          uuid.New(),
		Type:        EventRepositoryCreated,
		AggregateID: gitRepo.ID,
		TenantID:    project.TenantID,
		UserID:      &userID,
		Payload: map[string]interface{}{
			"project_id":      project.ID.String(),
			"repository_id":   gitRepo.ID.String(),
			"repository_name": gitRepo.Name,
			"description":     gitRepo.Description,
			"default_branch":  gitRepo.DefaultBranch,
		},
		Timestamp: time.Now(),
	}

	if err := s.eventBus.Publish(ctx, successEvent); err != nil {
		s.logger.Error("发布仓库创建事件失败", zap.Error(err))
	}

	s.logger.Info("仓库异步创建成功",
		zap.String("project_id", project.ID.String()),
		zap.String("repository_id", gitRepo.ID.String()),
		zap.String("repository_name", gitRepo.Name))

	return nil
}

// CreateRepositoryForProject 为已存在项目创建仓库
func (s *SimpleTransactionService) CreateRepositoryForProject(
	ctx context.Context,
	projectID, userID, tenantID uuid.UUID,
	createReq *client.CreateRepositoryRequest,
) (*models.Repository, error) {
	s.logger.Info("为项目创建仓库",
		zap.String("project_id", projectID.String()),
		zap.String("repository_name", createReq.Name))

	// 1. 验证项目存在和用户权限
	var project models.Project
	err := s.db.Where("id = ? AND tenant_id = ?", projectID, tenantID).First(&project).Error
	if err != nil {
		return nil, fmt.Errorf("项目不存在或无权访问: %w", err)
	}

	// 2. 直接调用Git网关创建仓库
	gitRepo, err := s.gitClient.CreateRepository(ctx, createReq)
	if err != nil {
		s.logger.Error("Git网关创建仓库失败", zap.Error(err))
		return nil, fmt.Errorf("创建仓库失败: %w", err)
	}

	// 3. 转换为项目服务模型
	repository := &models.Repository{
		ID:            gitRepo.ID,
		ProjectID:     gitRepo.ProjectID,
		Name:          gitRepo.Name,
		Description:   gitRepo.Description,
		Visibility:    string(gitRepo.Visibility),
		DefaultBranch: gitRepo.DefaultBranch,
	}

	// 4. 异步发布事件
	go func() {
		event := &DomainEvent{
			ID:          uuid.New(),
			Type:        EventRepositoryCreated,
			AggregateID: gitRepo.ID,
			TenantID:    tenantID,
			UserID:      &userID,
			Payload: map[string]interface{}{
				"project_id":      projectID.String(),
				"repository_id":   gitRepo.ID.String(),
				"repository_name": gitRepo.Name,
				"description":     gitRepo.Description,
			},
			Timestamp: time.Now(),
		}

		if err := s.eventBus.Publish(context.Background(), event); err != nil {
			s.logger.Error("发布仓库创建事件失败", zap.Error(err))
		}
	}()

	s.logger.Info("仓库创建成功",
		zap.String("repository_id", gitRepo.ID.String()),
		zap.String("repository_name", gitRepo.Name))

	return repository, nil
}

// ProcessPendingEvents 处理待处理事件（可由定时任务调用）
func (s *SimpleTransactionService) ProcessPendingEvents(ctx context.Context) error {
	var events []DomainEvent

	// 获取待处理事件（最多100个）
	err := s.db.Where("processed = false AND retry_count < 5").
		Order("created_at ASC").
		Limit(100).
		Find(&events).Error

	if err != nil {
		return fmt.Errorf("获取待处理事件失败: %w", err)
	}

	if len(events) == 0 {
		return nil
	}

	s.logger.Info("开始处理待处理事件", zap.Int("count", len(events)))

	for _, event := range events {
		if err := s.processEvent(ctx, &event); err != nil {
			s.logger.Error("事件处理失败",
				zap.String("event_id", event.ID.String()),
				zap.String("event_type", string(event.Type)),
				zap.Error(err))

			// 更新重试次数
			s.db.Model(&event).Updates(map[string]interface{}{
				"retry_count": event.RetryCount + 1,
				"error":       err.Error(),
				"updated_at":  time.Now(),
			})
		} else {
			// 标记为已处理
			s.db.Model(&event).Updates(map[string]interface{}{
				"processed":  true,
				"updated_at": time.Now(),
			})
		}
	}

	return nil
}

// processEvent 处理单个事件
func (s *SimpleTransactionService) processEvent(ctx context.Context, event *DomainEvent) error {
	switch event.Type {
	case EventRepositoryCreated:
		// 可以在这里添加仓库创建后的处理逻辑
		// 例如：发送通知、更新统计信息等
		return s.handleRepositoryCreated(ctx, event)
	case EventProjectCreated:
		// 项目创建后的处理逻辑
		return s.handleProjectCreated(ctx, event)
	default:
		s.logger.Debug("未知事件类型", zap.String("type", string(event.Type)))
		return nil
	}
}

// handleRepositoryCreated 处理仓库创建事件
func (s *SimpleTransactionService) handleRepositoryCreated(ctx context.Context, event *DomainEvent) error {
	s.logger.Info("处理仓库创建事件",
		zap.String("event_id", event.ID.String()),
		zap.String("aggregate_id", event.AggregateID.String()))

	// 这里可以添加仓库创建后的后续处理
	// 例如：
	// 1. 发送通知给项目成员
	// 2. 初始化默认分支保护规则
	// 3. 创建初始的Webhook配置
	// 4. 更新项目统计信息

	return nil
}

// handleProjectCreated 处理项目创建事件
func (s *SimpleTransactionService) handleProjectCreated(ctx context.Context, event *DomainEvent) error {
	s.logger.Info("处理项目创建事件",
		zap.String("event_id", event.ID.String()),
		zap.String("aggregate_id", event.AggregateID.String()))

	// 项目创建后的处理逻辑
	return nil
}

// CreateProjectRequest 创建项目请求
type CreateProjectRequest struct {
	TenantID       uuid.UUID `json:"tenant_id" validate:"required"`
	UserID         uuid.UUID `json:"user_id" validate:"required"`
	Name           string    `json:"name" validate:"required,min=1,max=100"`
	Description    string    `json:"description" validate:"max=500"`
	InitRepository bool      `json:"init_repository"`
	RepositoryName string    `json:"repository_name" validate:"max=100"`
}
