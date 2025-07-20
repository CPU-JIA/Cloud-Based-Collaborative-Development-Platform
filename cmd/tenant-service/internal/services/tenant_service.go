package services

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/tenant-service/internal/repository"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// TenantService 租户服务接口
type TenantService interface {
	// 租户管理
	CreateTenant(ctx context.Context, req *CreateTenantRequest) (*models.Tenant, error)
	GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, req *UpdateTenantRequest) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id uuid.UUID) error

	// 租户查询
	ListTenants(ctx context.Context, req *ListTenantsRequest) (*ListTenantsResponse, error)
	SearchTenants(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error)
	GetTenantStats(ctx context.Context) (*TenantStatsResponse, error)

	// 租户状态管理
	ActivateTenant(ctx context.Context, id uuid.UUID) error
	SuspendTenant(ctx context.Context, id uuid.UUID, reason string) error

	// 完整信息获取
	GetTenantWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
}

// 请求和响应结构

// CreateTenantRequest 创建租户请求
type CreateTenantRequest struct {
	Name         string            `json:"name" binding:"required,min=2,max=100"`
	Domain       string            `json:"domain" binding:"required,min=3,max=63"`
	Plan         models.TenantPlan `json:"plan" binding:"required"`
	BillingEmail string            `json:"billing_email" binding:"required,email"`
	Description  string            `json:"description"`
	ContactName  string            `json:"contact_name"`
	ContactEmail string            `json:"contact_email" binding:"omitempty,email"`
	ContactPhone string            `json:"contact_phone"`
	Address      string            `json:"address"`
	City         string            `json:"city"`
	State        string            `json:"state"`
	Country      string            `json:"country"`
	PostalCode   string            `json:"postal_code"`
}

// UpdateTenantRequest 更新租户请求
type UpdateTenantRequest struct {
	Name         *string `json:"name,omitempty" binding:"omitempty,min=2,max=100"`
	Description  *string `json:"description,omitempty"`
	ContactName  *string `json:"contact_name,omitempty"`
	ContactEmail *string `json:"contact_email,omitempty" binding:"omitempty,email"`
	ContactPhone *string `json:"contact_phone,omitempty"`
	Address      *string `json:"address,omitempty"`
	City         *string `json:"city,omitempty"`
	State        *string `json:"state,omitempty"`
	Country      *string `json:"country,omitempty"`
	PostalCode   *string `json:"postal_code,omitempty"`
}

// ListTenantsRequest 租户列表请求
type ListTenantsRequest struct {
	Status models.TenantStatus `json:"status,omitempty"`
	Plan   models.TenantPlan   `json:"plan,omitempty"`
	Offset int                 `json:"offset"`
	Limit  int                 `json:"limit" binding:"max=100"`
}

// ListTenantsResponse 租户列表响应
type ListTenantsResponse struct {
	Tenants []*models.Tenant `json:"tenants"`
	Total   int64            `json:"total"`
	Offset  int              `json:"offset"`
	Limit   int              `json:"limit"`
}

// TenantStatsResponse 租户统计响应
type TenantStatsResponse struct {
	TotalTenants  int64                         `json:"total_tenants"`
	StatusStats   map[models.TenantStatus]int64 `json:"status_stats"`
	PlanStats     map[models.TenantPlan]int64   `json:"plan_stats"`
	RecentTenants []*models.Tenant              `json:"recent_tenants"`
}

// tenantService 租户服务实现
type tenantService struct {
	tenantRepo       repository.TenantRepository
	configRepo       repository.ConfigRepository
	subscriptionRepo repository.SubscriptionRepository
	brandingRepo     repository.BrandingRepository
}

// NewTenantService 创建租户服务实例
func NewTenantService(
	tenantRepo repository.TenantRepository,
	configRepo repository.ConfigRepository,
	subscriptionRepo repository.SubscriptionRepository,
	brandingRepo repository.BrandingRepository,
) TenantService {
	return &tenantService{
		tenantRepo:       tenantRepo,
		configRepo:       configRepo,
		subscriptionRepo: subscriptionRepo,
		brandingRepo:     brandingRepo,
	}
}

// CreateTenant 创建租户
func (s *tenantService) CreateTenant(ctx context.Context, req *CreateTenantRequest) (*models.Tenant, error) {
	// 验证请求
	if err := s.validateCreateTenantRequest(req); err != nil {
		return nil, fmt.Errorf("请求验证失败: %w", err)
	}

	// 创建租户实体
	tenant := &models.Tenant{
		ID:           uuid.New(),
		Name:         req.Name,
		Domain:       strings.ToLower(req.Domain),
		Status:       models.TenantStatusPending,
		Plan:         req.Plan,
		BillingEmail: req.BillingEmail,
		Description:  req.Description,
		ContactName:  req.ContactName,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Address:      req.Address,
		City:         req.City,
		State:        req.State,
		Country:      req.Country,
		PostalCode:   req.PostalCode,
	}

	// 创建租户
	if err := s.tenantRepo.Create(ctx, tenant); err != nil {
		return nil, fmt.Errorf("创建租户失败: %w", err)
	}

	// 创建默认配置
	if err := s.createDefaultConfig(ctx, tenant.ID); err != nil {
		// 回滚租户创建（在实际项目中应该使用事务）
		_ = s.tenantRepo.Delete(ctx, tenant.ID)
		return nil, fmt.Errorf("创建默认配置失败: %w", err)
	}

	// 创建默认订阅
	if err := s.createDefaultSubscription(ctx, tenant.ID, req.Plan); err != nil {
		// 回滚（在实际项目中应该使用事务）
		_ = s.configRepo.Delete(ctx, tenant.ID)
		_ = s.tenantRepo.Delete(ctx, tenant.ID)
		return nil, fmt.Errorf("创建默认订阅失败: %w", err)
	}

	return tenant, nil
}

// GetTenant 获取租户
func (s *tenantService) GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

// GetTenantByDomain 根据域名获取租户
func (s *tenantService) GetTenantByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	return s.tenantRepo.GetByDomain(ctx, strings.ToLower(domain))
}

// UpdateTenant 更新租户
func (s *tenantService) UpdateTenant(ctx context.Context, id uuid.UUID, req *UpdateTenantRequest) (*models.Tenant, error) {
	// 获取现有租户
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != nil {
		tenant.Name = *req.Name
	}
	if req.Description != nil {
		tenant.Description = *req.Description
	}
	if req.ContactName != nil {
		tenant.ContactName = *req.ContactName
	}
	if req.ContactEmail != nil {
		tenant.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		tenant.ContactPhone = *req.ContactPhone
	}
	if req.Address != nil {
		tenant.Address = *req.Address
	}
	if req.City != nil {
		tenant.City = *req.City
	}
	if req.State != nil {
		tenant.State = *req.State
	}
	if req.Country != nil {
		tenant.Country = *req.Country
	}
	if req.PostalCode != nil {
		tenant.PostalCode = *req.PostalCode
	}

	// 更新租户
	if err := s.tenantRepo.Update(ctx, tenant); err != nil {
		return nil, fmt.Errorf("更新租户失败: %w", err)
	}

	return tenant, nil
}

// DeleteTenant 删除租户
func (s *tenantService) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	// 检查租户是否存在
	_, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// 软删除租户（实际项目中应该删除所有关联数据）
	return s.tenantRepo.Delete(ctx, id)
}

// ListTenants 获取租户列表
func (s *tenantService) ListTenants(ctx context.Context, req *ListTenantsRequest) (*ListTenantsResponse, error) {
	// 设置默认分页
	if req.Limit <= 0 || req.Limit > 100 {
		req.Limit = 20
	}

	var tenants []*models.Tenant
	var total int64
	var err error

	// 根据条件查询
	if req.Status != "" && req.Plan != "" {
		// 复合查询（这里简化处理，实际应该有专门的复合查询方法）
		tenants, err = s.tenantRepo.GetByStatus(ctx, req.Status, req.Offset, req.Limit)
		if err != nil {
			return nil, err
		}
		total, err = s.tenantRepo.CountByStatus(ctx, req.Status)
	} else if req.Status != "" {
		tenants, err = s.tenantRepo.GetByStatus(ctx, req.Status, req.Offset, req.Limit)
		if err != nil {
			return nil, err
		}
		total, err = s.tenantRepo.CountByStatus(ctx, req.Status)
	} else if req.Plan != "" {
		tenants, err = s.tenantRepo.GetByPlan(ctx, req.Plan, req.Offset, req.Limit)
		if err != nil {
			return nil, err
		}
		total, err = s.tenantRepo.CountByPlan(ctx, req.Plan)
	} else {
		tenants, err = s.tenantRepo.List(ctx, req.Offset, req.Limit)
		if err != nil {
			return nil, err
		}
		total, err = s.tenantRepo.Count(ctx)
	}

	if err != nil {
		return nil, err
	}

	return &ListTenantsResponse{
		Tenants: tenants,
		Total:   total,
		Offset:  req.Offset,
		Limit:   req.Limit,
	}, nil
}

// SearchTenants 搜索租户
func (s *tenantService) SearchTenants(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	return s.tenantRepo.Search(ctx, query, offset, limit)
}

// GetTenantStats 获取租户统计
func (s *tenantService) GetTenantStats(ctx context.Context) (*TenantStatsResponse, error) {
	// 获取总数
	total, err := s.tenantRepo.Count(ctx)
	if err != nil {
		return nil, err
	}

	// 获取状态统计
	statusStats := make(map[models.TenantStatus]int64)
	for _, status := range []models.TenantStatus{
		models.TenantStatusActive,
		models.TenantStatusSuspended,
		models.TenantStatusPending,
		models.TenantStatusDeleted,
	} {
		count, err := s.tenantRepo.CountByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		statusStats[status] = count
	}

	// 获取计划统计
	planStats := make(map[models.TenantPlan]int64)
	for _, plan := range []models.TenantPlan{
		models.TenantPlanFree,
		models.TenantPlanBasic,
		models.TenantPlanProfessional,
		models.TenantPlanEnterprise,
	} {
		count, err := s.tenantRepo.CountByPlan(ctx, plan)
		if err != nil {
			return nil, err
		}
		planStats[plan] = count
	}

	// 获取最近租户
	recentTenants, err := s.tenantRepo.List(ctx, 0, 10)
	if err != nil {
		return nil, err
	}

	return &TenantStatsResponse{
		TotalTenants:  total,
		StatusStats:   statusStats,
		PlanStats:     planStats,
		RecentTenants: recentTenants,
	}, nil
}

// ActivateTenant 激活租户
func (s *tenantService) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.Status == models.TenantStatusActive {
		return fmt.Errorf("租户已处于激活状态")
	}

	tenant.Status = models.TenantStatusActive
	return s.tenantRepo.Update(ctx, tenant)
}

// SuspendTenant 暂停租户
func (s *tenantService) SuspendTenant(ctx context.Context, id uuid.UUID, reason string) error {
	tenant, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	if tenant.Status == models.TenantStatusSuspended {
		return fmt.Errorf("租户已处于暂停状态")
	}

	tenant.Status = models.TenantStatusSuspended
	return s.tenantRepo.Update(ctx, tenant)
}

// GetTenantWithConfig 获取租户及配置
func (s *tenantService) GetTenantWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetWithConfig(ctx, id)
}

// GetTenantWithSubscription 获取租户及订阅
func (s *tenantService) GetTenantWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetWithSubscription(ctx, id)
}

// GetTenantWithAll 获取租户完整信息
func (s *tenantService) GetTenantWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	return s.tenantRepo.GetWithAll(ctx, id)
}

// 私有方法

// validateCreateTenantRequest 验证创建租户请求
func (s *tenantService) validateCreateTenantRequest(req *CreateTenantRequest) error {
	// 验证域名格式
	if !s.isValidDomain(req.Domain) {
		return fmt.Errorf("域名格式无效: %s", req.Domain)
	}

	// 验证计划类型
	validPlans := map[models.TenantPlan]bool{
		models.TenantPlanFree:         true,
		models.TenantPlanBasic:        true,
		models.TenantPlanProfessional: true,
		models.TenantPlanEnterprise:   true,
	}

	if !validPlans[req.Plan] {
		return fmt.Errorf("无效的计划类型: %s", req.Plan)
	}

	return nil
}

// isValidDomain 验证域名格式
func (s *tenantService) isValidDomain(domain string) bool {
	// 域名规则：3-63字符，只能包含字母、数字和连字符，不能以连字符开头或结尾
	pattern := `^[a-zA-Z0-9]([a-zA-Z0-9\-]{1,61}[a-zA-Z0-9])?$`
	matched, _ := regexp.MatchString(pattern, domain)
	return matched
}

// createDefaultConfig 创建默认配置
func (s *tenantService) createDefaultConfig(ctx context.Context, tenantID uuid.UUID) error {
	config := &models.TenantConfig{
		ID:                  uuid.New(),
		TenantID:            tenantID,
		MaxUsers:            10,
		MaxProjects:         5,
		MaxStorage:          1024, // 1GB
		MaxAPICallsDaily:    10000,
		FeatureFlags:        make(map[string]interface{}),
		SecurityPolicy:      make(map[string]interface{}),
		IntegrationSettings: make(map[string]interface{}),
	}

	// 设置默认功能开关
	config.FeatureFlags["api_access"] = true
	config.FeatureFlags["web_interface"] = true
	config.FeatureFlags["mobile_app"] = false
	config.FeatureFlags["advanced_analytics"] = false

	// 设置默认安全策略
	config.SecurityPolicy["require_mfa"] = false
	config.SecurityPolicy["password_min_length"] = 8
	config.SecurityPolicy["session_timeout"] = 3600 // 1小时

	return s.configRepo.Create(ctx, config)
}

// createDefaultSubscription 创建默认订阅
func (s *tenantService) createDefaultSubscription(ctx context.Context, tenantID uuid.UUID, plan models.TenantPlan) error {
	now := time.Now()

	subscription := &models.TenantSubscription{
		ID:                 uuid.New(),
		TenantID:           tenantID,
		PlanType:           plan,
		Status:             models.SubscriptionStatusTrialing,
		BillingCycle:       models.BillingCycleMonthly,
		CurrentPeriodStart: now,
		CurrentPeriodEnd:   now.AddDate(0, 1, 0), // 1个月后
		Amount:             0,                    // 试用期免费
		Currency:           "USD",
		UsageMetrics:       make(map[string]interface{}),
	}

	// 设置试用期（免费计划无试用期）
	if plan != models.TenantPlanFree {
		trialEnd := now.AddDate(0, 0, 14) // 14天试用
		subscription.TrialEndsAt = &trialEnd
	}

	// 设置使用量统计初始值
	subscription.UsageMetrics["users_count"] = 0
	subscription.UsageMetrics["projects_count"] = 0
	subscription.UsageMetrics["storage_used"] = 0
	subscription.UsageMetrics["api_calls_today"] = 0

	return s.subscriptionRepo.Create(ctx, subscription)
}
