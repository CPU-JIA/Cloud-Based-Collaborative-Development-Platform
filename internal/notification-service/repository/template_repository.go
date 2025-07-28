package repository

import (
	"context"

	"github.com/cloud-platform/collaborative-dev/internal/notification-service/models"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// TemplateRepository 模板存储库
type TemplateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建新的模板存储库
func NewTemplateRepository(db *gorm.DB) *TemplateRepository {
	return &TemplateRepository{
		db: db,
	}
}

// Create 创建模板
func (tr *TemplateRepository) Create(ctx context.Context, template *models.NotificationTemplate) error {
	return tr.db.WithContext(ctx).Create(template).Error
}

// GetByID 根据ID获取模板
func (tr *TemplateRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.NotificationTemplate, error) {
	var template models.NotificationTemplate
	err := tr.db.WithContext(ctx).First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetByType 根据类型获取模板
func (tr *TemplateRepository) GetByType(ctx context.Context, templateType string, tenantID uuid.UUID) ([]*models.NotificationTemplate, error) {
	var templates []*models.NotificationTemplate
	err := tr.db.WithContext(ctx).
		Where("type = ? AND tenant_id = ? AND is_active = ?", templateType, tenantID, true).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, err
}

// GetDefaultTemplateByType 获取指定类型的默认模板
func (tr *TemplateRepository) GetDefaultTemplateByType(ctx context.Context, templateType string, tenantID uuid.UUID) (*models.NotificationTemplate, error) {
	var template models.NotificationTemplate
	err := tr.db.WithContext(ctx).
		Where("type = ? AND tenant_id = ? AND is_active = ?", templateType, tenantID, true).
		Order("created_at ASC"). // 取最早创建的作为默认模板
		First(&template).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// GetAll 获取租户的所有模板
func (tr *TemplateRepository) GetAll(ctx context.Context, tenantID uuid.UUID) ([]*models.NotificationTemplate, error) {
	var templates []*models.NotificationTemplate
	err := tr.db.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Order("created_at DESC").
		Find(&templates).Error

	return templates, err
}

// Update 更新模板
func (tr *TemplateRepository) Update(ctx context.Context, template *models.NotificationTemplate) error {
	return tr.db.WithContext(ctx).Save(template).Error
}

// Delete 删除模板
func (tr *TemplateRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return tr.db.WithContext(ctx).Delete(&models.NotificationTemplate{}, id).Error
}

// GetActiveTemplates 获取活跃的模板
func (tr *TemplateRepository) GetActiveTemplates(ctx context.Context, tenantID uuid.UUID) ([]*models.NotificationTemplate, error) {
	var templates []*models.NotificationTemplate
	err := tr.db.WithContext(ctx).
		Where("tenant_id = ? AND is_active = ?", tenantID, true).
		Order("type ASC, created_at DESC").
		Find(&templates).Error

	return templates, err
}
