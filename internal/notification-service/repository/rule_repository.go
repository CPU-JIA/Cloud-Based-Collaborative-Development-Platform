package repository

import (
	"context"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// RuleRepository 规则存储库
type RuleRepository struct {
	db *gorm.DB
}

// NewRuleRepository 创建新的规则存储库
func NewRuleRepository(db *gorm.DB) *RuleRepository {
	return &RuleRepository{
		db: db,
	}
}

// Create 创建规则
func (rr *RuleRepository) Create(ctx context.Context, rule *models.NotificationRule) error {
	return rr.db.WithContext(ctx).Create(rule).Error
}

// GetByID 根据ID获取规则
func (rr *RuleRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.NotificationRule, error) {
	var rule models.NotificationRule
	err := rr.db.WithContext(ctx).
		Preload("Template").
		First(&rule, id).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// GetActiveRulesByTypeAndTenant 获取指定类型和租户的活跃规则
func (rr *RuleRepository) GetActiveRulesByTypeAndTenant(ctx context.Context, eventType string, tenantID uuid.UUID, userID *uuid.UUID, projectID *uuid.UUID) ([]*models.NotificationRule, error) {
	query := rr.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Where("JSON_CONTAINS(event_types, ?)", `"`+eventType+`"`)

	// 用户级别规则优先，然后是项目级别，最后是全局规则
	if userID != nil {
		query = query.Where("(user_id = ? OR user_id IS NULL)", *userID)
	} else {
		query = query.Where("user_id IS NULL")
	}

	if projectID != nil {
		query = query.Where("(project_id = ? OR project_id IS NULL)", *projectID)
	} else {
		query = query.Where("project_id IS NULL")
	}

	var rules []*models.NotificationRule
	err := query.
		Preload("Template").
		Order("user_id DESC, project_id DESC, created_at ASC"). // 用户规则 > 项目规则 > 全局规则
		Find(&rules).Error

	return rules, err
}

// GetByTenant 获取租户的所有规则
func (rr *RuleRepository) GetByTenant(ctx context.Context, tenantID uuid.UUID) ([]*models.NotificationRule, error) {
	var rules []*models.NotificationRule
	err := rr.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Preload("Template").
		Order("created_at DESC").
		Find(&rules).Error

	return rules, err
}

// GetByUser 获取用户的规则
func (rr *RuleRepository) GetByUser(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.NotificationRule, error) {
	var rules []*models.NotificationRule
	err := rr.db.WithContext(ctx).
		Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Preload("Template").
		Order("created_at DESC").
		Find(&rules).Error

	return rules, err
}

// GetByProject 获取项目的规则
func (rr *RuleRepository) GetByProject(ctx context.Context, projectID uuid.UUID, tenantID uuid.UUID) ([]*models.NotificationRule, error) {
	var rules []*models.NotificationRule
	err := rr.db.WithContext(ctx).
		Where("project_id = ? AND tenant_id = ?", projectID, tenantID).
		Preload("Template").
		Order("created_at DESC").
		Find(&rules).Error

	return rules, err
}

// Update 更新规则
func (rr *RuleRepository) Update(ctx context.Context, rule *models.NotificationRule) error {
	return rr.db.WithContext(ctx).Save(rule).Error
}

// Delete 删除规则
func (rr *RuleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return rr.db.WithContext(ctx).Delete(&models.NotificationRule{}, id).Error
}

// GetActiveRules 获取活跃的规则
func (rr *RuleRepository) GetActiveRules(ctx context.Context, tenantID uuid.UUID) ([]*models.NotificationRule, error) {
	var rules []*models.NotificationRule
	err := rr.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Preload("Template").
		Order("created_at DESC").
		Find(&rules).Error

	return rules, err
}

// GetGlobalRules 获取全局规则（不限定用户和项目）
func (rr *RuleRepository) GetGlobalRules(ctx context.Context, tenantID uuid.UUID) ([]*models.NotificationRule, error) {
	var rules []*models.NotificationRule
	err := rr.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id IS NULL AND project_id IS NULL AND is_active = ?", tenantID, true).
		Preload("Template").
		Order("created_at DESC").
		Find(&rules).Error

	return rules, err
}
