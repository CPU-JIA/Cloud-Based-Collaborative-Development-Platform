package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// TenantRepository 租户仓储接口
type TenantRepository interface {
	// 基础CRUD操作
	Create(ctx context.Context, tenant *models.Tenant) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	Update(ctx context.Context, tenant *models.Tenant) error
	Delete(ctx context.Context, id uuid.UUID) error

	// 查询操作
	List(ctx context.Context, offset, limit int) ([]*models.Tenant, error)
	Search(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error)
	GetByStatus(ctx context.Context, status models.TenantStatus, offset, limit int) ([]*models.Tenant, error)
	GetByPlan(ctx context.Context, plan models.TenantPlan, offset, limit int) ([]*models.Tenant, error)

	// 统计操作
	Count(ctx context.Context) (int64, error)
	CountByStatus(ctx context.Context, status models.TenantStatus) (int64, error)
	CountByPlan(ctx context.Context, plan models.TenantPlan) (int64, error)

	// 关联数据操作
	GetWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetWithBranding(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
}

// ConfigRepository 租户配置仓储接口
type ConfigRepository interface {
	Create(ctx context.Context, config *models.TenantConfig) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantConfig, error)
	Update(ctx context.Context, config *models.TenantConfig) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
	UpdateFeatureFlags(ctx context.Context, tenantID uuid.UUID, flags map[string]interface{}) error
	UpdateSecurityPolicy(ctx context.Context, tenantID uuid.UUID, policy map[string]interface{}) error
}

// SubscriptionRepository 订阅仓储接口
type SubscriptionRepository interface {
	Create(ctx context.Context, subscription *models.TenantSubscription) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantSubscription, error)
	Update(ctx context.Context, subscription *models.TenantSubscription) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
	GetExpiringSoon(ctx context.Context, days int) ([]*models.TenantSubscription, error)
	GetByStripeCustomerID(ctx context.Context, customerID string) (*models.TenantSubscription, error)
	UpdateUsageMetrics(ctx context.Context, tenantID uuid.UUID, metrics map[string]interface{}) error
}

// BrandingRepository 品牌仓储接口
type BrandingRepository interface {
	Create(ctx context.Context, branding *models.TenantBranding) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID) (*models.TenantBranding, error)
	GetByCustomDomain(ctx context.Context, domain string) (*models.TenantBranding, error)
	Update(ctx context.Context, branding *models.TenantBranding) error
	Delete(ctx context.Context, tenantID uuid.UUID) error
}

// AuditLogRepository 审计日志仓储接口
type AuditLogRepository interface {
	Create(ctx context.Context, log *models.TenantAuditLog) error
	GetByTenantID(ctx context.Context, tenantID uuid.UUID, offset, limit int) ([]*models.TenantAuditLog, error)
	GetByUserID(ctx context.Context, userID uuid.UUID, offset, limit int) ([]*models.TenantAuditLog, error)
	GetByAction(ctx context.Context, action string, offset, limit int) ([]*models.TenantAuditLog, error)
	Search(ctx context.Context, tenantID uuid.UUID, filters map[string]interface{}, offset, limit int) ([]*models.TenantAuditLog, error)
}

// 实现类

// postgresTenantRepository PostgreSQL租户仓储实现
type postgresTenantRepository struct {
	db *database.PostgresDB
}

// NewTenantRepository 创建租户仓储实例
func NewTenantRepository(db *database.PostgresDB) TenantRepository {
	return &postgresTenantRepository{db: db}
}

// Create 创建租户
func (r *postgresTenantRepository) Create(ctx context.Context, tenant *models.Tenant) error {
	// 检查域名是否已存在
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("domain = ? AND deleted_at IS NULL", tenant.Domain).Count(&count).Error; err != nil {
		return fmt.Errorf("检查域名唯一性失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("域名 %s 已被使用", tenant.Domain)
	}

	// 创建租户
	if err := r.db.DB.WithContext(ctx).Create(tenant).Error; err != nil {
		return fmt.Errorf("创建租户失败: %w", err)
	}

	return nil
}

// GetByID 根据ID获取租户
func (r *postgresTenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户不存在: %s", id)
		}
		return nil, fmt.Errorf("获取租户失败: %w", err)
	}

	return &tenant, nil
}

// GetByDomain 根据域名获取租户
func (r *postgresTenantRepository) GetByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Where("domain = ? AND deleted_at IS NULL", domain).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("域名对应的租户不存在: %s", domain)
		}
		return nil, fmt.Errorf("获取租户失败: %w", err)
	}

	return &tenant, nil
}

// Update 更新租户
func (r *postgresTenantRepository) Update(ctx context.Context, tenant *models.Tenant) error {
	// 检查租户是否存在
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("id = ? AND deleted_at IS NULL", tenant.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("检查租户存在性失败: %w", err)
	}

	if count == 0 {
		return fmt.Errorf("租户不存在: %s", tenant.ID)
	}

	// 检查域名唯一性（排除自己）
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("domain = ? AND id != ? AND deleted_at IS NULL", tenant.Domain, tenant.ID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("检查域名唯一性失败: %w", err)
	}

	if count > 0 {
		return fmt.Errorf("域名 %s 已被其他租户使用", tenant.Domain)
	}

	// 更新租户
	if err := r.db.DB.WithContext(ctx).Save(tenant).Error; err != nil {
		return fmt.Errorf("更新租户失败: %w", err)
	}

	return nil
}

// Delete 软删除租户
func (r *postgresTenantRepository) Delete(ctx context.Context, id uuid.UUID) error {
	result := r.db.DB.WithContext(ctx).Delete(&models.Tenant{}, id)
	if result.Error != nil {
		return fmt.Errorf("删除租户失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("租户不存在: %s", id)
	}

	return nil
}

// List 分页获取租户列表
func (r *postgresTenantRepository) List(ctx context.Context, offset, limit int) ([]*models.Tenant, error) {
	var tenants []*models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Where("deleted_at IS NULL").
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("获取租户列表失败: %w", err)
	}

	return tenants, nil
}

// Search 搜索租户
func (r *postgresTenantRepository) Search(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error) {
	var tenants []*models.Tenant
	searchPattern := "%" + query + "%"

	if err := r.db.DB.WithContext(ctx).
		Where("deleted_at IS NULL AND (name ILIKE ? OR domain ILIKE ? OR billing_email ILIKE ?)",
			searchPattern, searchPattern, searchPattern).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("搜索租户失败: %w", err)
	}

	return tenants, nil
}

// GetByStatus 根据状态获取租户
func (r *postgresTenantRepository) GetByStatus(ctx context.Context, status models.TenantStatus, offset, limit int) ([]*models.Tenant, error) {
	var tenants []*models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Where("status = ? AND deleted_at IS NULL", status).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("获取指定状态租户失败: %w", err)
	}

	return tenants, nil
}

// GetByPlan 根据计划获取租户
func (r *postgresTenantRepository) GetByPlan(ctx context.Context, plan models.TenantPlan, offset, limit int) ([]*models.Tenant, error) {
	var tenants []*models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Where("plan = ? AND deleted_at IS NULL", plan).
		Order("created_at DESC").
		Offset(offset).Limit(limit).
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("获取指定计划租户失败: %w", err)
	}

	return tenants, nil
}

// Count 统计租户总数
func (r *postgresTenantRepository) Count(ctx context.Context) (int64, error) {
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("deleted_at IS NULL").Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计租户总数失败: %w", err)
	}

	return count, nil
}

// CountByStatus 根据状态统计租户数量
func (r *postgresTenantRepository) CountByStatus(ctx context.Context, status models.TenantStatus) (int64, error) {
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("status = ? AND deleted_at IS NULL", status).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计指定状态租户数量失败: %w", err)
	}

	return count, nil
}

// CountByPlan 根据计划统计租户数量
func (r *postgresTenantRepository) CountByPlan(ctx context.Context, plan models.TenantPlan) (int64, error) {
	var count int64
	if err := r.db.DB.WithContext(ctx).Model(&models.Tenant{}).
		Where("plan = ? AND deleted_at IS NULL", plan).Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计指定计划租户数量失败: %w", err)
	}

	return count, nil
}

// GetWithConfig 获取租户及其配置
func (r *postgresTenantRepository) GetWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Preload("Config").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户不存在: %s", id)
		}
		return nil, fmt.Errorf("获取租户和配置失败: %w", err)
	}

	return &tenant, nil
}

// GetWithSubscription 获取租户及其订阅信息
func (r *postgresTenantRepository) GetWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Preload("Subscription").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户不存在: %s", id)
		}
		return nil, fmt.Errorf("获取租户和订阅信息失败: %w", err)
	}

	return &tenant, nil
}

// GetWithBranding 获取租户及其品牌信息
func (r *postgresTenantRepository) GetWithBranding(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Preload("Branding").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户不存在: %s", id)
		}
		return nil, fmt.Errorf("获取租户和品牌信息失败: %w", err)
	}

	return &tenant, nil
}

// GetWithAll 获取租户及其所有关联信息
func (r *postgresTenantRepository) GetWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	var tenant models.Tenant
	if err := r.db.DB.WithContext(ctx).
		Preload("Config").
		Preload("Subscription").
		Preload("Branding").
		Where("id = ? AND deleted_at IS NULL", id).
		First(&tenant).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("租户不存在: %s", id)
		}
		return nil, fmt.Errorf("获取租户完整信息失败: %w", err)
	}

	return &tenant, nil
}
