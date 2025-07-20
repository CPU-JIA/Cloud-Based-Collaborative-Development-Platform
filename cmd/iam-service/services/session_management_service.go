package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// SessionManagementService 会话管理服务
type SessionManagementService struct {
	db *database.PostgresDB
}

// CreateSessionRequest 创建会话请求
type CreateSessionRequest struct {
	UserID       uuid.UUID `json:"user_id" binding:"required"`
	TenantID     uuid.UUID `json:"tenant_id" binding:"required"`
	IPAddress    string    `json:"ip_address" binding:"required"`
	UserAgent    string    `json:"user_agent"`
	DeviceInfo   string    `json:"device_info"`
	AccessToken  string    `json:"access_token" binding:"required"`
	RefreshToken string    `json:"refresh_token" binding:"required"`
	ExpiresAt    time.Time `json:"expires_at" binding:"required"`
}

// SessionInfo 会话信息
type SessionInfo struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	DeviceInfo   string    `json:"device_info"`
	Location     string    `json:"location,omitempty"`
	IsActive     bool      `json:"is_active"`
	IsCurrent    bool      `json:"is_current"`
	LastActivity time.Time `json:"last_activity"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// GetUserSessionsRequest 获取用户会话请求
type GetUserSessionsRequest struct {
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	TenantID   uuid.UUID `json:"tenant_id" binding:"required"`
	OnlyActive bool      `json:"only_active"`
}

// NewSessionManagementService 创建会话管理服务
func NewSessionManagementService(db *database.PostgresDB) *SessionManagementService {
	return &SessionManagementService{db: db}
}

// CreateSession 创建用户会话
func (s *SessionManagementService) CreateSession(ctx context.Context, req *CreateSessionRequest) (*models.UserSession, error) {
	tx := s.db.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查并发会话数限制
	maxSessions := 5 // 每个用户最多5个并发会话
	var activeSessionCount int64
	err := tx.Model(&models.UserSession{}).
		Where("user_id = ? AND tenant_id = ? AND is_active = true AND expires_at > ?",
			req.UserID, req.TenantID, time.Now()).
		Count(&activeSessionCount).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("检查活跃会话数失败: %w", err)
	}

	// 如果超过限制，删除最旧的会话
	if activeSessionCount >= int64(maxSessions) {
		var oldestSession models.UserSession
		err := tx.Where("user_id = ? AND tenant_id = ? AND is_active = true", req.UserID, req.TenantID).
			Order("last_activity ASC").
			First(&oldestSession).Error
		if err == nil {
			// 删除最旧的会话
			err = tx.Model(&oldestSession).Updates(map[string]interface{}{
				"is_active":     false,
				"revoked_at":    time.Now(),
				"revoke_reason": "exceeded_max_sessions",
			}).Error
			if err != nil {
				tx.Rollback()
				return nil, fmt.Errorf("删除最旧会话失败: %w", err)
			}
		}
	}

	// 生成设备指纹
	deviceFingerprint := s.generateDeviceFingerprint(req.IPAddress, req.UserAgent, req.DeviceInfo)

	// 创建新会话
	session := models.UserSession{
		UserID:            req.UserID,
		TenantID:          req.TenantID,
		SessionToken:      s.hashToken(req.AccessToken),
		RefreshToken:      s.hashToken(req.RefreshToken),
		IPAddress:         req.IPAddress,
		UserAgent:         req.UserAgent,
		DeviceInfo:        req.DeviceInfo,
		DeviceFingerprint: deviceFingerprint,
		IsActive:          true,
		LastActivity:      time.Now(),
		ExpiresAt:         req.ExpiresAt,
	}

	if err := tx.Create(&session).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建会话失败: %w", err)
	}

	// 更新用户最后登录时间
	now := time.Now()
	err = tx.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", req.UserID, req.TenantID).
		Update("last_login_at", &now).Error
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("更新用户登录时间失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &session, nil
}

// ValidateSession 验证会话
func (s *SessionManagementService) ValidateSession(ctx context.Context, accessToken string, userID, tenantID uuid.UUID) (*models.UserSession, error) {
	hashedToken := s.hashToken(accessToken)

	var session models.UserSession
	err := s.db.DB.WithContext(ctx).
		Where("session_token = ? AND user_id = ? AND tenant_id = ? AND is_active = true AND expires_at > ?",
			hashedToken, userID, tenantID, time.Now()).
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("会话不存在或已过期")
		}
		return nil, fmt.Errorf("验证会话失败: %w", err)
	}

	// 更新最后活动时间
	now := time.Now()
	err = s.db.DB.WithContext(ctx).Model(&session).
		Update("last_activity", now).Error
	if err != nil {
		return nil, fmt.Errorf("更新会话活动时间失败: %w", err)
	}

	session.LastActivity = now
	return &session, nil
}

// RefreshSession 刷新会话
func (s *SessionManagementService) RefreshSession(ctx context.Context, refreshToken string, newAccessToken, newRefreshToken string, newExpiresAt time.Time) error {
	hashedRefreshToken := s.hashToken(refreshToken)

	var session models.UserSession
	err := s.db.DB.WithContext(ctx).
		Where("refresh_token = ? AND is_active = true AND expires_at > ?",
			hashedRefreshToken, time.Now()).
		First(&session).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("刷新令牌不存在或已过期")
		}
		return fmt.Errorf("查询会话失败: %w", err)
	}

	// 更新会话令牌
	updates := map[string]interface{}{
		"session_token": s.hashToken(newAccessToken),
		"refresh_token": s.hashToken(newRefreshToken),
		"last_activity": time.Now(),
		"expires_at":    newExpiresAt,
	}

	err = s.db.DB.WithContext(ctx).Model(&session).Updates(updates).Error
	if err != nil {
		return fmt.Errorf("更新会话失败: %w", err)
	}

	return nil
}

// RevokeSession 撤销单个会话
func (s *SessionManagementService) RevokeSession(ctx context.Context, sessionID uuid.UUID, userID, tenantID uuid.UUID, reason string) error {
	updates := map[string]interface{}{
		"is_active":     false,
		"revoked_at":    time.Now(),
		"revoke_reason": reason,
	}

	err := s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("id = ? AND user_id = ? AND tenant_id = ?", sessionID, userID, tenantID).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("撤销会话失败: %w", err)
	}

	return nil
}

// RevokeUserSessions 撤销用户所有会话
func (s *SessionManagementService) RevokeUserSessions(ctx context.Context, userID, tenantID uuid.UUID, excludeSessionID *uuid.UUID, reason string) error {
	query := s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("user_id = ? AND tenant_id = ? AND is_active = true", userID, tenantID)

	if excludeSessionID != nil {
		query = query.Where("id != ?", *excludeSessionID)
	}

	updates := map[string]interface{}{
		"is_active":     false,
		"revoked_at":    time.Now(),
		"revoke_reason": reason,
	}

	err := query.Updates(updates).Error
	if err != nil {
		return fmt.Errorf("撤销用户会话失败: %w", err)
	}

	return nil
}

// GetUserSessions 获取用户会话列表
func (s *SessionManagementService) GetUserSessions(ctx context.Context, req *GetUserSessionsRequest, currentSessionToken string) ([]SessionInfo, error) {
	query := s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("user_id = ? AND tenant_id = ?", req.UserID, req.TenantID)

	if req.OnlyActive {
		query = query.Where("is_active = true AND expires_at > ?", time.Now())
	}

	var sessions []models.UserSession
	err := query.Order("last_activity DESC").Find(&sessions).Error
	if err != nil {
		return nil, fmt.Errorf("查询用户会话失败: %w", err)
	}

	// 当前会话令牌哈希（用于标识当前会话）
	var currentSessionHash string
	if currentSessionToken != "" {
		currentSessionHash = s.hashToken(currentSessionToken)
	}

	sessionInfos := make([]SessionInfo, len(sessions))
	for i, session := range sessions {
		sessionInfos[i] = SessionInfo{
			ID:           session.ID,
			UserID:       session.UserID,
			IPAddress:    session.IPAddress,
			UserAgent:    session.UserAgent,
			DeviceInfo:   session.DeviceInfo,
			Location:     s.getLocationFromIP(session.IPAddress),
			IsActive:     session.IsActive,
			IsCurrent:    session.SessionToken == currentSessionHash,
			LastActivity: session.LastActivity,
			CreatedAt:    session.CreatedAt,
			ExpiresAt:    session.ExpiresAt,
		}
	}

	return sessionInfos, nil
}

// CleanupExpiredSessions 清理过期会话
func (s *SessionManagementService) CleanupExpiredSessions(ctx context.Context) error {
	updates := map[string]interface{}{
		"is_active":     false,
		"revoked_at":    time.Now(),
		"revoke_reason": "expired",
	}

	err := s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("is_active = true AND expires_at <= ?", time.Now()).
		Updates(updates).Error
	if err != nil {
		return fmt.Errorf("清理过期会话失败: %w", err)
	}

	return nil
}

// GetSessionStats 获取会话统计信息
func (s *SessionManagementService) GetSessionStats(ctx context.Context, tenantID uuid.UUID, startTime, endTime time.Time) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 总会话数
	var totalSessions int64
	err := s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Count(&totalSessions).Error
	if err != nil {
		return nil, fmt.Errorf("统计总会话数失败: %w", err)
	}
	stats["total_sessions"] = totalSessions

	// 活跃会话数
	var activeSessions int64
	err = s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("tenant_id = ? AND is_active = true AND expires_at > ?", tenantID, time.Now()).
		Count(&activeSessions).Error
	if err != nil {
		return nil, fmt.Errorf("统计活跃会话数失败: %w", err)
	}
	stats["active_sessions"] = activeSessions

	// 唯一用户数
	var uniqueUsers int64
	err = s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ?", tenantID, startTime, endTime).
		Distinct("user_id").Count(&uniqueUsers).Error
	if err != nil {
		return nil, fmt.Errorf("统计唯一用户数失败: %w", err)
	}
	stats["unique_users"] = uniqueUsers

	// 平均会话时长
	type SessionDuration struct {
		AvgDuration float64 `json:"avg_duration"`
	}
	var avgDuration SessionDuration
	err = s.db.DB.WithContext(ctx).Model(&models.UserSession{}).
		Where("tenant_id = ? AND created_at BETWEEN ? AND ? AND revoked_at IS NOT NULL", tenantID, startTime, endTime).
		Select("AVG(EXTRACT(EPOCH FROM (revoked_at - created_at))/60) as avg_duration").
		Scan(&avgDuration).Error
	if err != nil {
		return nil, fmt.Errorf("统计平均会话时长失败: %w", err)
	}
	stats["avg_session_duration_minutes"] = avgDuration.AvgDuration

	return stats, nil
}

// generateDeviceFingerprint 生成设备指纹
func (s *SessionManagementService) generateDeviceFingerprint(ipAddress, userAgent, deviceInfo string) string {
	data := fmt.Sprintf("%s|%s|%s", ipAddress, userAgent, deviceInfo)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// hashToken 哈希令牌
func (s *SessionManagementService) hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// getLocationFromIP 根据IP获取地理位置（简化实现）
func (s *SessionManagementService) getLocationFromIP(ipAddress string) string {
	// 这里应该集成IP地理位置服务，如MaxMind GeoIP
	// 简化实现，返回IP地址
	if ipAddress == "127.0.0.1" || ipAddress == "::1" {
		return "本地"
	}
	return "未知位置"
}
