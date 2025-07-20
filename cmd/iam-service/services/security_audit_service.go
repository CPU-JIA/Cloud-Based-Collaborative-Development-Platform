package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// SecurityAuditService 安全审计服务
type SecurityAuditService struct {
	db *database.PostgresDB
}

// AuditLogRequest 审计日志请求
type AuditLogRequest struct {
	TenantID     uuid.UUID              `json:"tenant_id"`
	UserID       *uuid.UUID             `json:"user_id,omitempty"`
	IPAddress    string                 `json:"ip_address"`
	UserAgent    string                 `json:"user_agent"`
	Action       string                 `json:"action"`
	Resource     string                 `json:"resource"`
	Details      string                 `json:"details"`
	Success      bool                   `json:"success"`
	ErrorCode    string                 `json:"error_code,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LoginAttemptRequest 登录尝试记录请求
type LoginAttemptRequest struct {
	TenantID    uuid.UUID  `json:"tenant_id"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
	Email       string     `json:"email"`
	IPAddress   string     `json:"ip_address"`
	UserAgent   string     `json:"user_agent"`
	Success     bool       `json:"success"`
	FailReason  string     `json:"fail_reason,omitempty"`
	MFARequired bool       `json:"mfa_required"`
	MFASuccess  *bool      `json:"mfa_success,omitempty"`
}

// GetAuditLogsRequest 获取审计日志请求
type GetAuditLogsRequest struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Action    string     `json:"action,omitempty"`
	Resource  string     `json:"resource,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}

// GetLoginAttemptsRequest 获取登录尝试记录请求
type GetLoginAttemptsRequest struct {
	TenantID  uuid.UUID  `json:"tenant_id"`
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Email     string     `json:"email,omitempty"`
	IPAddress string     `json:"ip_address,omitempty"`
	Success   *bool      `json:"success,omitempty"`
	StartTime *time.Time `json:"start_time,omitempty"`
	EndTime   *time.Time `json:"end_time,omitempty"`
	Page      int        `json:"page"`
	Limit     int        `json:"limit"`
}

// SecurityMetrics 安全指标
type SecurityMetrics struct {
	TotalLoginAttempts   int64                `json:"total_login_attempts"`
	SuccessfulLogins     int64                `json:"successful_logins"`
	FailedLogins         int64                `json:"failed_logins"`
	SuccessRate          float64              `json:"success_rate"`
	UniqueUsers          int64                `json:"unique_users"`
	UniqueIPs            int64                `json:"unique_ips"`
	MFAUsage             int64                `json:"mfa_usage"`
	TotalAuditLogs       int64                `json:"total_audit_logs"`
	TopFailReasons       []FailReasonCount    `json:"top_fail_reasons"`
	HourlyLoginPattern   []HourlyCount        `json:"hourly_login_pattern"`
	SuspiciousActivities []SuspiciousActivity `json:"suspicious_activities"`
}

// FailReasonCount 失败原因统计
type FailReasonCount struct {
	Reason string `json:"reason"`
	Count  int64  `json:"count"`
}

// HourlyCount 小时统计
type HourlyCount struct {
	Hour  int   `json:"hour"`
	Count int64 `json:"count"`
}

// SuspiciousActivity 可疑活动
type SuspiciousActivity struct {
	Type        string     `json:"type"`
	Description string     `json:"description"`
	Count       int64      `json:"count"`
	LastSeen    time.Time  `json:"last_seen"`
	IPAddress   string     `json:"ip_address,omitempty"`
	UserID      *uuid.UUID `json:"user_id,omitempty"`
}

// NewSecurityAuditService 创建安全审计服务
func NewSecurityAuditService(db *database.PostgresDB) *SecurityAuditService {
	return &SecurityAuditService{db: db}
}

// LogAuditEvent 记录审计事件
func (s *SecurityAuditService) LogAuditEvent(ctx context.Context, req *AuditLogRequest) error {
	var metadataJSON string
	if req.Metadata != nil {
		metadataBytes, err := json.Marshal(req.Metadata)
		if err != nil {
			return fmt.Errorf("序列化元数据失败: %w", err)
		}
		metadataJSON = string(metadataBytes)
	}

	auditLog := models.SecurityAuditLog{
		TenantID:     req.TenantID,
		UserID:       req.UserID,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
		Action:       req.Action,
		Resource:     req.Resource,
		Details:      req.Details,
		Success:      req.Success,
		ErrorCode:    req.ErrorCode,
		ErrorMessage: req.ErrorMessage,
		Metadata:     metadataJSON,
	}

	err := s.db.DB.WithContext(ctx).Create(&auditLog).Error
	if err != nil {
		return fmt.Errorf("记录审计日志失败: %w", err)
	}

	return nil
}

// LogLoginAttempt 记录登录尝试
func (s *SecurityAuditService) LogLoginAttempt(ctx context.Context, req *LoginAttemptRequest) error {
	loginAttempt := models.LoginAttempt{
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Email:       req.Email,
		IPAddress:   req.IPAddress,
		UserAgent:   req.UserAgent,
		Success:     req.Success,
		FailReason:  req.FailReason,
		MFARequired: req.MFARequired,
		MFASuccess:  req.MFASuccess,
	}

	err := s.db.DB.WithContext(ctx).Create(&loginAttempt).Error
	if err != nil {
		return fmt.Errorf("记录登录尝试失败: %w", err)
	}

	// 同时记录到审计日志
	action := "login_success"
	if !req.Success {
		action = "login_failed"
	}

	auditReq := &AuditLogRequest{
		TenantID:  req.TenantID,
		UserID:    req.UserID,
		IPAddress: req.IPAddress,
		UserAgent: req.UserAgent,
		Action:    action,
		Resource:  "auth",
		Success:   req.Success,
		Metadata: map[string]interface{}{
			"email":        req.Email,
			"mfa_required": req.MFARequired,
			"mfa_success":  req.MFASuccess,
		},
	}

	if !req.Success {
		auditReq.ErrorMessage = req.FailReason
	}

	return s.LogAuditEvent(ctx, auditReq)
}

// GetAuditLogs 获取审计日志
func (s *SecurityAuditService) GetAuditLogs(ctx context.Context, req *GetAuditLogsRequest) ([]models.SecurityAuditLog, int64, error) {
	query := s.db.DB.WithContext(ctx).Model(&models.SecurityAuditLog{}).
		Where("tenant_id = ?", req.TenantID)

	// 添加过滤条件
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.Resource != "" {
		query = query.Where("resource = ?", req.Resource)
	}
	if req.Success != nil {
		query = query.Where("success = ?", *req.Success)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计审计日志总数失败: %w", err)
	}

	// 分页查询
	var logs []models.SecurityAuditLog
	offset := (req.Page - 1) * req.Limit
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&logs).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询审计日志失败: %w", err)
	}

	return logs, total, nil
}

// GetLoginAttempts 获取登录尝试记录
func (s *SecurityAuditService) GetLoginAttempts(ctx context.Context, req *GetLoginAttemptsRequest) ([]models.LoginAttempt, int64, error) {
	query := s.db.DB.WithContext(ctx).Model(&models.LoginAttempt{}).
		Where("tenant_id = ?", req.TenantID)

	// 添加过滤条件
	if req.UserID != nil {
		query = query.Where("user_id = ?", *req.UserID)
	}
	if req.Email != "" {
		query = query.Where("email ILIKE ?", "%"+req.Email+"%")
	}
	if req.IPAddress != "" {
		query = query.Where("ip_address = ?", req.IPAddress)
	}
	if req.Success != nil {
		query = query.Where("success = ?", *req.Success)
	}
	if req.StartTime != nil {
		query = query.Where("created_at >= ?", *req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("created_at <= ?", *req.EndTime)
	}

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计登录尝试总数失败: %w", err)
	}

	// 分页查询
	var attempts []models.LoginAttempt
	offset := (req.Page - 1) * req.Limit
	err := query.Preload("User").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.Limit).
		Find(&attempts).Error
	if err != nil {
		return nil, 0, fmt.Errorf("查询登录尝试失败: %w", err)
	}

	return attempts, total, nil
}

// GetSecurityMetrics 获取安全指标
func (s *SecurityAuditService) GetSecurityMetrics(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time) (*SecurityMetrics, error) {
	db := s.db.DB.WithContext(ctx)
	metrics := &SecurityMetrics{}

	// 基础登录统计
	err := db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Select("COUNT(*) as total, COUNT(CASE WHEN success = true THEN 1 END) as successful").
		Row().Scan(&metrics.TotalLoginAttempts, &metrics.SuccessfulLogins)
	if err != nil {
		return nil, fmt.Errorf("查询登录统计失败: %w", err)
	}

	metrics.FailedLogins = metrics.TotalLoginAttempts - metrics.SuccessfulLogins
	if metrics.TotalLoginAttempts > 0 {
		metrics.SuccessRate = float64(metrics.SuccessfulLogins) / float64(metrics.TotalLoginAttempts) * 100
	}

	// 唯一用户和IP统计
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Select("COUNT(DISTINCT user_id) as unique_users, COUNT(DISTINCT ip_address) as unique_ips").
		Row().Scan(&metrics.UniqueUsers, &metrics.UniqueIPs)
	if err != nil {
		return nil, fmt.Errorf("查询唯一用户统计失败: %w", err)
	}

	// MFA使用统计
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND mfa_required = true", tenantID, startTime, endTime).
		Count(&metrics.MFAUsage).Error
	if err != nil {
		return nil, fmt.Errorf("查询MFA使用统计失败: %w", err)
	}

	// 审计日志总数
	err = db.Model(&models.SecurityAuditLog{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Count(&metrics.TotalAuditLogs).Error
	if err != nil {
		return nil, fmt.Errorf("查询审计日志统计失败: %w", err)
	}

	// 失败原因统计
	var failReasons []FailReasonCount
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND success = false AND fail_reason != ''", tenantID, startTime, endTime).
		Select("fail_reason as reason, COUNT(*) as count").
		Group("fail_reason").
		Order("count DESC").
		Limit(10).
		Find(&failReasons).Error
	if err != nil {
		return nil, fmt.Errorf("查询失败原因统计失败: %w", err)
	}
	metrics.TopFailReasons = failReasons

	// 小时分布统计
	var hourlyPattern []HourlyCount
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Select("EXTRACT(hour FROM created_at) as hour, COUNT(*) as count").
		Group("hour").
		Order("hour").
		Find(&hourlyPattern).Error
	if err != nil {
		return nil, fmt.Errorf("查询小时分布统计失败: %w", err)
	}
	metrics.HourlyLoginPattern = hourlyPattern

	// 可疑活动检测
	suspiciousActivities, err := s.detectSuspiciousActivities(ctx, tenantID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("检测可疑活动失败: %w", err)
	}
	metrics.SuspiciousActivities = suspiciousActivities

	return metrics, nil
}

// detectSuspiciousActivities 检测可疑活动
func (s *SecurityAuditService) detectSuspiciousActivities(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time) ([]SuspiciousActivity, error) {
	var activities []SuspiciousActivity
	db := s.db.DB.WithContext(ctx)

	// 检测暴力破解攻击（同一IP短时间内大量失败登录）
	type IPFailCount struct {
		IPAddress string    `json:"ip_address"`
		Count     int64     `json:"count"`
		LastSeen  time.Time `json:"last_seen"`
	}

	var bruteForceIPs []IPFailCount
	err := db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND success = false", tenantID, startTime, endTime).
		Select("ip_address, COUNT(*) as count, MAX(created_at) as last_seen").
		Group("ip_address").
		Having("COUNT(*) >= 10"). // 10次失败登录视为可疑
		Order("count DESC").
		Find(&bruteForceIPs).Error
	if err != nil {
		return nil, err
	}

	for _, ip := range bruteForceIPs {
		activities = append(activities, SuspiciousActivity{
			Type:        "brute_force",
			Description: fmt.Sprintf("IP %s 在短时间内尝试登录 %d 次失败", ip.IPAddress, ip.Count),
			Count:       ip.Count,
			LastSeen:    ip.LastSeen,
			IPAddress:   ip.IPAddress,
		})
	}

	// 检测异常登录时间（非工作时间大量登录）
	var nightLogins int64
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND (EXTRACT(hour FROM created_at) < 6 OR EXTRACT(hour FROM created_at) > 22)", tenantID, startTime, endTime).
		Count(&nightLogins).Error
	if err != nil {
		return nil, err
	}

	if nightLogins > 50 { // 夜间登录超过50次视为异常
		activities = append(activities, SuspiciousActivity{
			Type:        "unusual_hours",
			Description: fmt.Sprintf("非工作时间（22:00-06:00）有 %d 次登录尝试", nightLogins),
			Count:       nightLogins,
			LastSeen:    endTime,
		})
	}

	// 检测多地登录（同一用户在短时间内来自不同IP）
	type UserMultiIP struct {
		UserID   uuid.UUID `json:"user_id"`
		IPCount  int64     `json:"ip_count"`
		LastSeen time.Time `json:"last_seen"`
	}

	var multiIPUsers []UserMultiIP
	err = db.Model(&models.LoginAttempt{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND success = true AND user_id IS NOT NULL", tenantID, startTime, endTime).
		Select("user_id, COUNT(DISTINCT ip_address) as ip_count, MAX(created_at) as last_seen").
		Group("user_id").
		Having("COUNT(DISTINCT ip_address) >= 5"). // 5个不同IP视为异常
		Find(&multiIPUsers).Error
	if err != nil {
		return nil, err
	}

	for _, user := range multiIPUsers {
		activities = append(activities, SuspiciousActivity{
			Type:        "multiple_locations",
			Description: fmt.Sprintf("用户从 %d 个不同IP地址登录", user.IPCount),
			Count:       user.IPCount,
			LastSeen:    user.LastSeen,
			UserID:      &user.UserID,
		})
	}

	return activities, nil
}

// CleanupOldLogs 清理旧日志（数据保留策略）
func (s *SecurityAuditService) CleanupOldLogs(ctx context.Context, retentionDays int) error {
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	// 清理旧的审计日志
	err := s.db.DB.WithContext(ctx).
		Where("created_at < ?", cutoffTime).
		Delete(&models.SecurityAuditLog{}).Error
	if err != nil {
		return fmt.Errorf("清理审计日志失败: %w", err)
	}

	// 清理旧的登录尝试记录
	err = s.db.DB.WithContext(ctx).
		Where("created_at < ?", cutoffTime).
		Delete(&models.LoginAttempt{}).Error
	if err != nil {
		return fmt.Errorf("清理登录尝试记录失败: %w", err)
	}

	// 清理旧的MFA尝试记录
	err = s.db.DB.WithContext(ctx).
		Where("created_at < ?", cutoffTime).
		Delete(&models.MFAAttempt{}).Error
	if err != nil {
		return fmt.Errorf("清理MFA尝试记录失败: %w", err)
	}

	return nil
}
