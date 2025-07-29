package integration

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/cloud-platform/collaborative-dev/shared/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MockTenantService 租户服务模拟实现
type MockTenantService struct {
	tenants       map[uuid.UUID]*models.Tenant
	configs       map[uuid.UUID]*models.TenantConfig
	subscriptions map[uuid.UUID]*models.TenantSubscription
}

// NewMockTenantService 创建模拟租户服务
func NewMockTenantService() *MockTenantService {
	return &MockTenantService{
		tenants:       make(map[uuid.UUID]*models.Tenant),
		configs:       make(map[uuid.UUID]*models.TenantConfig),
		subscriptions: make(map[uuid.UUID]*models.TenantSubscription),
	}
}

// CreateTenant 创建租户
func (m *MockTenantService) CreateTenant(ctx context.Context, req *TenantCreateRequest) (*models.Tenant, error) {
	// 检查域名是否已存在
	for _, tenant := range m.tenants {
		if tenant.Domain == req.Domain {
			return nil, fmt.Errorf("domain already exists: %s", req.Domain)
		}
	}

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
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	m.tenants[tenant.ID] = tenant

	// 创建默认配置
	config := &models.TenantConfig{
		ID:              uuid.New(),
		TenantID:        tenant.ID,
		MaxUsers:        10,
		MaxProjects:     5,
		MaxStorage:      1024,
		MaxAPICallsDaily: 10000,
		FeatureFlags:    map[string]interface{}{
			"api_access":    true,
			"web_interface": true,
		},
		SecurityPolicy: map[string]interface{}{
			"require_mfa":         false,
			"password_min_length": 8,
		},
		IntegrationSettings: map[string]interface{}{
			"github_enabled": true,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.configs[tenant.ID] = config

	// 创建默认订阅
	subscription := &models.TenantSubscription{
		ID:                 uuid.New(),
		TenantID:           tenant.ID,
		PlanType:           req.Plan,
		Status:             models.SubscriptionStatusTrialing,
		BillingCycle:       models.BillingCycleMonthly,
		CurrentPeriodStart: time.Now(),
		CurrentPeriodEnd:   time.Now().AddDate(0, 1, 0),
		Amount:             0,
		Currency:           "USD",
		UsageMetrics:       map[string]interface{}{
			"users_count":    0,
			"projects_count": 0,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	m.subscriptions[tenant.ID] = subscription

	return tenant, nil
}

// GetTenant 获取租户
func (m *MockTenantService) GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}
	return tenant, nil
}

// GetTenantByDomain 根据域名获取租户
func (m *MockTenantService) GetTenantByDomain(ctx context.Context, domain string) (*models.Tenant, error) {
	for _, tenant := range m.tenants {
		if tenant.Domain == domain {
			return tenant, nil
		}
	}
	return nil, fmt.Errorf("tenant not found for domain: %s", domain)
}

// UpdateTenant 更新租户
func (m *MockTenantService) UpdateTenant(ctx context.Context, id uuid.UUID, req *TenantUpdateRequest) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}

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

	tenant.UpdatedAt = time.Now()
	return tenant, nil
}

// DeleteTenant 删除租户
func (m *MockTenantService) DeleteTenant(ctx context.Context, id uuid.UUID) error {
	_, exists := m.tenants[id]
	if !exists {
		return fmt.Errorf("tenant not found: %s", id)
	}
	delete(m.tenants, id)
	delete(m.configs, id)
	delete(m.subscriptions, id)
	return nil
}

// ListTenants 获取租户列表
func (m *MockTenantService) ListTenants(ctx context.Context, req *TenantListRequest) (*TenantListResponse, error) {
	var filteredTenants []*models.Tenant

	for _, tenant := range m.tenants {
		// 应用过滤条件
		if req.Status != "" && tenant.Status != req.Status {
			continue
		}
		if req.Plan != "" && tenant.Plan != req.Plan {
			continue
		}
		filteredTenants = append(filteredTenants, tenant)
	}

	// 应用分页
	total := int64(len(filteredTenants))
	start := req.Offset
	end := req.Offset + req.Limit

	if start > len(filteredTenants) {
		start = len(filteredTenants)
	}
	if end > len(filteredTenants) {
		end = len(filteredTenants)
	}

	pagedTenants := filteredTenants[start:end]

	return &TenantListResponse{
		Tenants: pagedTenants,
		Total:   total,
		Offset:  req.Offset,
		Limit:   req.Limit,
	}, nil
}

// SearchTenants 搜索租户
func (m *MockTenantService) SearchTenants(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error) {
	var results []*models.Tenant

	query = strings.ToLower(query)
	for _, tenant := range m.tenants {
		if strings.Contains(strings.ToLower(tenant.Name), query) ||
			strings.Contains(strings.ToLower(tenant.Domain), query) ||
			strings.Contains(strings.ToLower(tenant.Description), query) {
			results = append(results, tenant)
		}
	}

	// 应用分页
	start := offset
	end := offset + limit

	if start > len(results) {
		start = len(results)
	}
	if end > len(results) {
		end = len(results)
	}

	return results[start:end], nil
}

// GetTenantStats 获取租户统计
func (m *MockTenantService) GetTenantStats(ctx context.Context) (*TenantStats, error) {
	statusStats := make(map[models.TenantStatus]int64)
	planStats := make(map[models.TenantPlan]int64)

	for _, tenant := range m.tenants {
		statusStats[tenant.Status]++
		planStats[tenant.Plan]++
	}

	var recentTenants []*models.Tenant
	count := 0
	for _, tenant := range m.tenants {
		if count < 10 {
			recentTenants = append(recentTenants, tenant)
			count++
		}
	}

	return &TenantStats{
		TotalTenants:  int64(len(m.tenants)),
		StatusStats:   statusStats,
		PlanStats:     planStats,
		RecentTenants: recentTenants,
	}, nil
}

// ActivateTenant 激活租户
func (m *MockTenantService) ActivateTenant(ctx context.Context, id uuid.UUID) error {
	tenant, exists := m.tenants[id]
	if !exists {
		return fmt.Errorf("tenant not found: %s", id)
	}
	tenant.Status = models.TenantStatusActive
	tenant.UpdatedAt = time.Now()
	return nil
}

// SuspendTenant 暂停租户
func (m *MockTenantService) SuspendTenant(ctx context.Context, id uuid.UUID, reason string) error {
	tenant, exists := m.tenants[id]
	if !exists {
		return fmt.Errorf("tenant not found: %s", id)
	}
	tenant.Status = models.TenantStatusSuspended
	tenant.UpdatedAt = time.Now()
	return nil
}

// GetTenantWithConfig 获取租户及配置
func (m *MockTenantService) GetTenantWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}

	config, exists := m.configs[id]
	if exists {
		tenant.Config = config
	}

	return tenant, nil
}

// GetTenantWithSubscription 获取租户及订阅
func (m *MockTenantService) GetTenantWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}

	subscription, exists := m.subscriptions[id]
	if exists {
		tenant.Subscription = subscription
	}

	return tenant, nil
}

// GetTenantWithAll 获取租户完整信息
func (m *MockTenantService) GetTenantWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error) {
	tenant, exists := m.tenants[id]
	if !exists {
		return nil, fmt.Errorf("tenant not found: %s", id)
	}

	config, exists := m.configs[id]
	if exists {
		tenant.Config = config
	}

	subscription, exists := m.subscriptions[id]
	if exists {
		tenant.Subscription = subscription
	}

	return tenant, nil
}

// MockTenantHandler 租户处理器模拟实现
type MockTenantHandler struct {
	tenantService TenantService
	logger        logger.Logger
}

// NewMockTenantHandler 创建模拟租户处理器
func NewMockTenantHandler(tenantService TenantService, logger logger.Logger) *MockTenantHandler {
	return &MockTenantHandler{
		tenantService: tenantService,
		logger:        logger,
	}
}

// CreateTenant 创建租户接口
func (h *MockTenantHandler) CreateTenant(c *gin.Context) {
	var req TenantCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数无效", "error": err.Error()})
		return
	}

	tenant, err := h.tenantService.CreateTenant(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("创建租户失败", "error", err)
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(201, gin.H{"code": 201, "message": "租户创建成功", "data": tenant})
}

// GetTenant 获取租户详情接口
func (h *MockTenantHandler) GetTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	tenant, err := h.tenantService.GetTenant(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": tenant})
}

// GetTenantByDomain 根据域名获取租户接口
func (h *MockTenantHandler) GetTenantByDomain(c *gin.Context) {
	domain := c.Query("domain")
	if domain == "" {
		c.JSON(400, gin.H{"code": 400, "message": "域名参数不能为空"})
		return
	}

	tenant, err := h.tenantService.GetTenantByDomain(c.Request.Context(), domain)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": tenant})
}

// UpdateTenant 更新租户接口
func (h *MockTenantHandler) UpdateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	var req TenantUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数无效", "error": err.Error()})
		return
	}

	tenant, err := h.tenantService.UpdateTenant(c.Request.Context(), id, &req)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "更新成功", "data": tenant})
}

// DeleteTenant 删除租户接口
func (h *MockTenantHandler) DeleteTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	if err := h.tenantService.DeleteTenant(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.Status(204)
}

// ListTenants 获取租户列表接口
func (h *MockTenantHandler) ListTenants(c *gin.Context) {
	req := &TenantListRequest{
		Limit: 20, // 默认值
	}

	if status := c.Query("status"); status != "" {
		req.Status = models.TenantStatus(status)
	}
	if plan := c.Query("plan"); plan != "" {
		req.Plan = models.TenantPlan(plan)
	}

	response, err := h.tenantService.ListTenants(c.Request.Context(), req)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": response})
}

// SearchTenants 搜索租户接口
func (h *MockTenantHandler) SearchTenants(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(400, gin.H{"code": 400, "message": "搜索关键词不能为空"})
		return
	}

	tenants, err := h.tenantService.SearchTenants(c.Request.Context(), query, 0, 20)
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "搜索成功", "data": tenants})
}

// GetTenantStats 获取租户统计接口
func (h *MockTenantHandler) GetTenantStats(c *gin.Context) {
	stats, err := h.tenantService.GetTenantStats(c.Request.Context())
	if err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": stats})
}

// ActivateTenant 激活租户接口
func (h *MockTenantHandler) ActivateTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	if err := h.tenantService.ActivateTenant(c.Request.Context(), id); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "激活成功"})
}

// SuspendTenant 暂停租户接口
func (h *MockTenantHandler) SuspendTenant(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	var req TenantSuspendRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "请求参数无效"})
		return
	}

	if err := h.tenantService.SuspendTenant(c.Request.Context(), id, req.Reason); err != nil {
		c.JSON(500, gin.H{"code": 500, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "暂停成功"})
}

// GetTenantWithConfig 获取租户及配置接口
func (h *MockTenantHandler) GetTenantWithConfig(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	tenant, err := h.tenantService.GetTenantWithConfig(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": tenant})
}

// GetTenantWithAll 获取租户完整信息接口
func (h *MockTenantHandler) GetTenantWithAll(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "message": "租户ID格式无效"})
		return
	}

	tenant, err := h.tenantService.GetTenantWithAll(c.Request.Context(), id)
	if err != nil {
		c.JSON(404, gin.H{"code": 404, "message": err.Error()})
		return
	}

	c.JSON(200, gin.H{"code": 200, "message": "获取成功", "data": tenant})
}

// 接口定义

// TenantService 租户服务接口
type TenantService interface {
	CreateTenant(ctx context.Context, req *TenantCreateRequest) (*models.Tenant, error)
	GetTenant(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantByDomain(ctx context.Context, domain string) (*models.Tenant, error)
	UpdateTenant(ctx context.Context, id uuid.UUID, req *TenantUpdateRequest) (*models.Tenant, error)
	DeleteTenant(ctx context.Context, id uuid.UUID) error
	ListTenants(ctx context.Context, req *TenantListRequest) (*TenantListResponse, error)
	SearchTenants(ctx context.Context, query string, offset, limit int) ([]*models.Tenant, error)
	GetTenantStats(ctx context.Context) (*TenantStats, error)
	ActivateTenant(ctx context.Context, id uuid.UUID) error
	SuspendTenant(ctx context.Context, id uuid.UUID, reason string) error
	GetTenantWithConfig(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantWithSubscription(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
	GetTenantWithAll(ctx context.Context, id uuid.UUID) (*models.Tenant, error)
}

// 请求和响应结构

// TenantCreateRequest 创建租户请求
type TenantCreateRequest struct {
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

// TenantUpdateRequest 更新租户请求
type TenantUpdateRequest struct {
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

// TenantListRequest 租户列表请求
type TenantListRequest struct {
	Status models.TenantStatus `json:"status,omitempty"`
	Plan   models.TenantPlan   `json:"plan,omitempty"`
	Offset int                 `json:"offset"`
	Limit  int                 `json:"limit" binding:"max=100"`
}

// TenantListResponse 租户列表响应
type TenantListResponse struct {
	Tenants []*models.Tenant `json:"tenants"`
	Total   int64            `json:"total"`
	Offset  int              `json:"offset"`
	Limit   int              `json:"limit"`
}

// TenantStats 租户统计响应
type TenantStats struct {
	TotalTenants  int64                         `json:"total_tenants"`
	StatusStats   map[models.TenantStatus]int64 `json:"status_stats"`
	PlanStats     map[models.TenantPlan]int64   `json:"plan_stats"`
	RecentTenants []*models.Tenant              `json:"recent_tenants"`
}

// TenantSuspendRequest 暂停租户请求
type TenantSuspendRequest struct {
	Reason string `json:"reason" binding:"required"`
}