package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// postgresSubscriptionRepository PostgreSQL订阅仓储实现
type postgresSubscriptionRepository struct {
	db *database.PostgresDB
}

// NewSubscriptionRepository 创建订阅仓储实例
func NewSubscriptionRepository(db *database.PostgresDB) SubscriptionRepository {
	return &postgresSubscriptionRepository{db: db}
}

// Create 创建租户订阅
func (r *postgresSubscriptionRepository) Create(ctx context.Context, subscription *models.TenantSubscription) error {
	// 检查是否已存在订阅
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("tenant_id = ? AND deleted_at IS NULL", subscription.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查订阅存在性失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("租户 %s 的订阅已存在", subscription.TenantID)
	}

	// 设置默认值
	if subscription.UsageMetrics == nil {
		subscription.UsageMetrics = make(map[string]interface{})
	}

	// 创建订阅
	if err := r.db.DB.WithContext(ctx).Create(subscription).Error; err != nil {
		return fmt.Errorf("创建租户订阅失败: %w", err)
	}

	return nil
}

// GetByTenantID 根据租户ID获取订阅
func (r *postgresSubscriptionRepository) GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantSubscription, error) {
	var subscription models.TenantSubscription
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户 %s 的订阅不存在", tenantID)
		}
		return nil, fmt.Errorf("获取租户订阅失败: %w", err)
	}

	return &subscription, nil
}

// Update 更新租户订阅
func (r *postgresSubscriptionRepository) Update(ctx context.Context, subscription *models.TenantSubscription) error {
	// 检查订阅是否存在
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("tenant_id = ? AND deleted_at IS NULL", subscription.TenantID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查订阅存在性失败: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("租户 %s 的订阅不存在", subscription.TenantID)
	}

	// 更新订阅
	if err := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", subscription.TenantID).
		Save(subscription).Error; err != nil {
		return fmt.Errorf("更新租户订阅失败: %w", err)
	}

	return nil
}

// Delete 删除租户订阅
func (r *postgresSubscriptionRepository) Delete(ctx context.Context, tenantID uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).
		Where("tenant_id = ?", tenantID).
		Delete(&models.TenantSubscription{})

	if result.Error != nil {
		return fmt.Errorf("删除租户订阅失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的订阅不存在", tenantID)
	}

	return nil
}

// GetExpiringSoon 获取即将过期的订阅
func (r *postgresSubscriptionRepository) GetExpiringSoon(ctx context.Context, days int) ([]*models.TenantSubscription, error) {
	var subscriptions []*models.TenantSubscription
	expiryDate := time.Now().AddDate(0, 0, days)

	if err := r.db.DB.WithContext(ctx).
		Where("expires_at <= ? AND expires_at > ? AND status = ? AND deleted_at IS NULL",
			expiryDate, time.Now(), models.SubscriptionStatusActive).
		Find(&subscriptions).Error; err != nil {
		return nil, fmt.Errorf("获取即将过期的订阅失败: %w", err)
	}

	return subscriptions, nil
}

// GetByStripeCustomerID 根据Stripe客户ID获取订阅
func (r *postgresSubscriptionRepository) GetByStripeCustomerID(ctx context.Context, customerID string) (*models.TenantSubscription, error) {
	var subscription models.TenantSubscription
	if err := r.db.DB.WithContext(ctx).
		Where("stripe_customer_id = ? AND deleted_at IS NULL", customerID).
		First(&subscription).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("Stripe客户 %s 的订阅不存在", customerID)
		}
		return nil, fmt.Errorf("获取Stripe客户订阅失败: %w", err)
	}

	return &subscription, nil
}

// UpdateUsageMetrics 更新使用量统计
func (r *postgresSubscriptionRepository) UpdateUsageMetrics(ctx context.Context, tenantID uuid.UUID, metrics map[string]interface{}) error {
	result := r.db.DB.WithContext(ctx).Model(&models.TenantSubscription{}).
		Where("tenant_id = ? AND deleted_at IS NULL", tenantID).
		Update("usage_metrics", metrics)

	if result.Error != nil {
		return fmt.Errorf("更新使用量统计失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户 %s 的订阅不存在", tenantID)
	}

	return nil
}
