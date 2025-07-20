package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// postgresBrandingRepository PostgreSQL品牌仓储实现
type postgresBrandingRepository struct {
	db *database.PostgresDB
}

// NewBrandingRepository 创建品牌仓储实例
func NewBrandingRepository(db *database.PostgresDB) BrandingRepository {
	return &postgresBrandingRepository{db: db}
}

// Create 创建租户品牌
func (r *postgresBrandingRepository) Create(ctx context.Context, branding *models.TenantBranding) error {
	// 检查是否已存在品牌配置
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantBranding{}).
		Where("tenant_id = ? AND deleted_at IS NULL", branding.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查品牌配置存在性失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("租户 %s 的品牌配置已存在", branding.TenantID)
	}

	// 设置默认值
	if branding.BrandingSettings == nil {
		branding.BrandingSettings = make(map[string]interface{})
	}

	// 创建品牌配置
	if err := r.db.DB.WithContext(ctx).Create(branding).Error; err != nil {
		return fmt.Errorf("创建租户品牌配置失败: %w", err)
	}

	return nil
}

// GetByTenantID 根据租户ID获取品牌配置
func (r *postgresBrandingRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantBranding, error) {
	var branding models.TenantBranding
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		First(&branding).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户 %s 的品牌配置不存在", tenantID)
		}
		return nil, fmt.Errorf("获取租户品牌配置失败: %w", err)
	}

	return &branding, nil
}

// GetByCustomDomain 根据自定义域名获取品牌配置
func (r *postgresBrandingRepository) GetByCustomDomain(ctx context.Context, domain string) (*models.TenantBranding, error) {
	var branding models.TenantBranding
	if err := r.db.DB.WithContext(ctx).
		Where("custom_domain = ? AND deleted_at IS NULL", domain).
		First(&branding).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("自定义域名 %s 的品牌配置不存在", domain)
		}
		return nil, fmt.Errorf("获取自定义域名品牌配置失败: %w", err)
	}

	return &branding, nil
}

// Update 更新租户品牌配置
func (r *postgresBrandingRepository) Update(ctx context.Context, branding *models.TenantBranding) error {
	// 检查品牌配置是否存在
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantBranding{}).
		Where("tenant_id = ? AND deleted_at IS NULL", branding.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查品牌配置存在性失败: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("租户 %s 的品牌配置不存在", branding.TenantID)
	}

	// 更新品牌配置
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", branding.TenantID).
		Save(branding).Error; err != nil {
		return fmt.Errorf("更新租户品牌配置失败: %w", err)
	}

	return nil
}

// Delete 删除租户品牌配置
func (r *postgresBrandingRepository) Delete(ctx context.Context, tenantID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Delete(&models.TenantBranding{})

	if result.Error != nil {
		return fmt.Errorf("删除租户品牌配置失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的品牌配置不存在", tenantID)
	}

	return nil
}
