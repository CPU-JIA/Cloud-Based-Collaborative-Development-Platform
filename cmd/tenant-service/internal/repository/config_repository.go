package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// postgresConfigRepository PostgreSQL配置仓储实现
type postgresConfigRepository struct {
	db *database.PostgresDB
}

// NewConfigRepository 创建配置仓储实例
func NewConfigRepository(db *database.PostgresDB) ConfigRepository {
	return &postgresConfigRepository{db: db}
}

// Create 创建租户配置
func (r *postgresConfigRepository) Create(ctx context.Context, config *models.TenantConfig) error {
	// 检查是否已存在配置
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantConfig{}).
		Where("tenant_id = ? AND deleted_at IS NULL", config.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查配置存在性失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("租户 %s 的配置已存在", config.TenantID)
	}

	// 设置默认值
	if config.FeatureFlags == nil {
		config.FeatureFlags = make(map[string]interface{})
	}
	if config.SecurityPolicy == nil {
		config.SecurityPolicy = make(map[string]interface{})
	}
	if config.IntegrationSettings == nil {
		config.IntegrationSettings = make(map[string]interface{})
	}

	// 创建配置
	if err := r.db.DB.WithContext(ctx).Create(config).Error; err != nil {
		return fmt.Errorf("创建租户配置失败: %w", err)
	}

	return nil
}

// GetByTenantID 根据租户ID获取配置
func (r *postgresConfigRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantConfig, error) {
	var config models.TenantConfig
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		First(&config).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户 %s 的配置不存在", tenantID)
		}
		return nil, fmt.Errorf("获取租户配置失败: %w", err)
	}

	return &config, nil
}

// Update 更新租户配置
func (r *postgresConfigRepository) Update(ctx context.Context, config *models.TenantConfig) error {
	// 检查配置是否存在
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantConfig{}).
		Where("tenant_id = ? AND deleted_at IS NULL", config.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查配置存在性失败: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("租户 %s 的配置不存在", config.TenantID)
	}

	// 更新配置
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", config.TenantID).
		Save(config).Error; err != nil {
		return fmt.Errorf("更新租户配置失败: %w", err)
	}

	return nil
}

// Delete 删除租户配置
func (r *postgresConfigRepository) Delete(ctx context.Context, tenantID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Delete(&models.TenantConfig{})

	if result.Error != nil {
		return fmt.Errorf("删除租户配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的配置不存在", tenantID)
	}

	return nil
}

// UpdateFeatureFlags 更新功能开关
func (r *postgresConfigRepository) UpdateFeatureFlags(ctx context.Context, tenantID uuid.UUID, flags map[string]interface{}) error {
	result := r.db.DB.WithContext(ctx).Model(&models.TenantConfig{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Update("feature_flags", flags)

	if result.Error != nil {
		return fmt.Errorf("更新功能开关失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的配置不存在", tenantID)
	}

	return nil
}

// UpdateSecurityPolicy 更新安全策略
func (r *postgresConfigRepository) UpdateSecurityPolicy(ctx context.Context, tenantID uuid.UUID, policy map[string]interface{}) error {
	result := r.db.DB.WithContext(ctx).Model(&models.TenantConfig{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Update("security_policy", policy)

	if result.Error != nil {
		return fmt.Errorf("更新安全策略失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的配置不存在", tenantID)
	}

	return nil
}
