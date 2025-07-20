package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/auth"
	"github.com/cloud-platform/collaborative-dev/shared/database"
	"github.com/cloud-platform/collaborative-dev/shared/models"
)

// MFAManagementService MFA管理服务
type MFAManagementService struct {
	db         *database.PostgresDB
	mfaService *auth.MFAService
}

// EnableMFARequest 启用MFA请求
type EnableMFARequest struct {
	UserID     uuid.UUID `json:"user_id" binding:"required"`
	TenantID   uuid.UUID `json:"tenant_id" binding:"required"`
	DeviceName string    `json:"device_name" binding:"required,min=1,max=100"`
	DeviceType string    `json:"device_type" binding:"required,oneof=totp sms email"`
	Phone      string    `json:"phone,omitempty"`
	Email      string    `json:"email,omitempty"`
}

// EnableMFAResponse 启用MFA响应
type EnableMFAResponse struct {
	DeviceID      uuid.UUID               `json:"device_id"`
	Secret        string                  `json:"secret,omitempty"`
	QRCodeB64     string                  `json:"qr_code_base64,omitempty"`
	QRCodeURL     string                  `json:"qr_code_url,omitempty"`
	BackupCodes   []string                `json:"backup_codes"`
	SetupRequired bool                    `json:"setup_required"`
}

// VerifyMFARequest 验证MFA请求
type VerifyMFARequest struct {
	UserID   uuid.UUID `json:"user_id" binding:"required"`
	TenantID uuid.UUID `json:"tenant_id" binding:"required"`
	Code     string    `json:"code" binding:"required,min=6,max=8"`
	DeviceID *uuid.UUID `json:"device_id,omitempty"`
	IsBackupCode bool   `json:"is_backup_code,omitempty"`
}

// MFADeviceInfo MFA设备信息
type MFADeviceInfo struct {
	ID         uuid.UUID  `json:"id"`
	DeviceName string     `json:"device_name"`
	DeviceType string     `json:"device_type"`
	IsActive   bool       `json:"is_active"`
	IsPrimary  bool       `json:"is_primary"`
	LastUsedAt *time.Time `json:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

// NewMFAManagementService 创建MFA管理服务
func NewMFAManagementService(db *database.PostgresDB, mfaService *auth.MFAService) *MFAManagementService {
	return &MFAManagementService{
		db:         db,
		mfaService: mfaService,
	}
}

// EnableMFA 启用MFA
func (s *MFAManagementService) EnableMFA(ctx context.Context, req *EnableMFARequest, ipAddress, userAgent string) (*EnableMFAResponse, error) {
	tx := s.db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 检查用户是否存在
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", req.UserID, req.TenantID).First(&user).Error
	if err != nil {
		tx.Rollback()
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("用户不存在")
		}
		return nil, fmt.Errorf("查询用户失败: %w", err)
	}

	// 检查是否已经启用MFA
	if user.TwoFactorEnabled {
		tx.Rollback()
		return nil, fmt.Errorf("用户已启用MFA")
	}

	var response EnableMFAResponse
	var device models.UserMFADevice

	switch req.DeviceType {
	case "totp":
		// 生成TOTP密钥和QR码
		qrInfo, err := s.mfaService.GenerateSecret(user.Email)
		if err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("生成TOTP密钥失败: %w", err)
		}

		device = models.UserMFADevice{
			UserID:     req.UserID,
			TenantID:   req.TenantID,
			DeviceName: req.DeviceName,
			DeviceType: "totp",
			Secret:     qrInfo.Secret,
			IsActive:   true,
			IsPrimary:  true,
		}

		response.Secret = qrInfo.Secret
		response.QRCodeB64 = qrInfo.QRCodeB64
		response.QRCodeURL = qrInfo.QRCodeURL
		response.SetupRequired = true

	case "sms":
		if req.Phone == "" {
			tx.Rollback()
			return nil, fmt.Errorf("SMS设备需要提供手机号")
		}

		device = models.UserMFADevice{
			UserID:     req.UserID,
			TenantID:   req.TenantID,
			DeviceName: req.DeviceName,
			DeviceType: "sms",
			Phone:      req.Phone,
			IsActive:   true,
			IsPrimary:  true,
		}

	case "email":
		email := req.Email
		if email == "" {
			email = user.Email // 使用用户默认邮箱
		}

		device = models.UserMFADevice{
			UserID:     req.UserID,
			TenantID:   req.TenantID,
			DeviceName: req.DeviceName,
			DeviceType: "email",
			Email:      email,
			IsActive:   true,
			IsPrimary:  true,
		}
	}

	// 保存MFA设备
	if err := tx.Create(&device).Error; err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("创建MFA设备失败: %w", err)
	}

	// 生成备用验证码
	backupCodes, err := s.mfaService.GenerateBackupCodes(10)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("生成备用验证码失败: %w", err)
	}

	// 保存备用验证码（加密存储）
	for _, code := range backupCodes {
		hashedCode := s.hashBackupCode(code)
		backupCodeRecord := models.UserMFABackupCode{
			UserID:   req.UserID,
			TenantID: req.TenantID,
			Code:     hashedCode,
		}
		if err := tx.Create(&backupCodeRecord).Error; err != nil {
			tx.Rollback()
			return nil, fmt.Errorf("保存备用验证码失败: %w", err)
		}
	}

	response.DeviceID = device.ID
	response.BackupCodes = backupCodes

	if err := tx.Commit().Error; err != nil {
		return nil, fmt.Errorf("提交事务失败: %w", err)
	}

	return &response, nil
}

// VerifyMFASetup 验证MFA设置
func (s *MFAManagementService) VerifyMFASetup(ctx context.Context, req *VerifyMFARequest, ipAddress, userAgent string) error {
	tx := s.db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 记录MFA尝试
	attempt := models.MFAAttempt{
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		DeviceID:  req.DeviceID,
	}

	// 查找用户和设备
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ?", req.UserID, req.TenantID).First(&user).Error
	if err != nil {
		attempt.Success = false
		attempt.FailReason = "用户不存在"
		tx.Create(&attempt)
		tx.Commit()
		return fmt.Errorf("用户不存在")
	}

	var isValid bool
	
	if req.IsBackupCode {
		// 验证备用码
		isValid, err = s.validateBackupCode(tx, req.UserID, req.TenantID, req.Code)
	} else {
		// 验证TOTP代码
		isValid, err = s.validateTOTPCode(tx, req.UserID, req.TenantID, req.Code, req.DeviceID)
	}

	if err != nil {
		attempt.Success = false
		attempt.FailReason = err.Error()
		tx.Create(&attempt)
		tx.Commit()
		return err
	}

	if !isValid {
		attempt.Success = false
		attempt.FailReason = "验证码无效"
		tx.Create(&attempt)
		tx.Commit()
		return fmt.Errorf("验证码无效")
	}

	// 激活用户的MFA
	err = tx.Model(&user).Where("id = ? AND tenant_id = ?", req.UserID, req.TenantID).
		Update("two_factor_enabled", true).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("启用MFA失败: %w", err)
	}

	// 更新设备最后使用时间
	if req.DeviceID != nil {
		now := time.Now()
		tx.Model(&models.UserMFADevice{}).
			Where("id = ? AND user_id = ? AND tenant_id = ?", *req.DeviceID, req.UserID, req.TenantID).
			Update("last_used_at", &now)
	}

	attempt.Success = true
	tx.Create(&attempt)

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// VerifyMFA 验证MFA（登录时使用）
func (s *MFAManagementService) VerifyMFA(ctx context.Context, req *VerifyMFARequest, ipAddress, userAgent string) error {
	tx := s.db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 记录MFA尝试
	attempt := models.MFAAttempt{
		UserID:    req.UserID,
		TenantID:  req.TenantID,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		DeviceID:  req.DeviceID,
	}

	// 检查用户是否启用了MFA
	var user models.User
	err := tx.Where("id = ? AND tenant_id = ? AND two_factor_enabled = true", req.UserID, req.TenantID).First(&user).Error
	if err != nil {
		attempt.Success = false
		attempt.FailReason = "MFA未启用"
		tx.Create(&attempt)
		tx.Commit()
		return fmt.Errorf("MFA未启用")
	}

	var isValid bool
	
	if req.IsBackupCode {
		// 验证备用码
		isValid, err = s.validateBackupCode(tx, req.UserID, req.TenantID, req.Code)
	} else {
		// 验证TOTP代码
		isValid, err = s.validateTOTPCode(tx, req.UserID, req.TenantID, req.Code, req.DeviceID)
	}

	if err != nil {
		attempt.Success = false
		attempt.FailReason = err.Error()
		tx.Create(&attempt)
		tx.Commit()
		return err
	}

	if !isValid {
		attempt.Success = false
		attempt.FailReason = "验证码无效"
		tx.Create(&attempt)
		tx.Commit()
		return fmt.Errorf("验证码无效")
	}

	// 更新设备最后使用时间
	if req.DeviceID != nil {
		now := time.Now()
		tx.Model(&models.UserMFADevice{}).
			Where("id = ? AND user_id = ? AND tenant_id = ?", *req.DeviceID, req.UserID, req.TenantID).
			Update("last_used_at", &now)
	}

	attempt.Success = true
	tx.Create(&attempt)

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// GetUserMFADevices 获取用户MFA设备列表
func (s *MFAManagementService) GetUserMFADevices(ctx context.Context, userID, tenantID uuid.UUID) ([]MFADeviceInfo, error) {
	var devices []models.UserMFADevice
	err := s.db.GetDB().WithContext(ctx).
		Where("user_id = ? AND tenant_id = ? AND is_active = true", userID, tenantID).
		Order("is_primary DESC, created_at ASC").
		Find(&devices).Error
	if err != nil {
		return nil, fmt.Errorf("查询MFA设备失败: %w", err)
	}

	deviceInfos := make([]MFADeviceInfo, len(devices))
	for i, device := range devices {
		deviceInfos[i] = MFADeviceInfo{
			ID:         device.ID,
			DeviceName: device.DeviceName,
			DeviceType: device.DeviceType,
			IsActive:   device.IsActive,
			IsPrimary:  device.IsPrimary,
			LastUsedAt: device.LastUsedAt,
			CreatedAt:  device.CreatedAt,
		}
	}

	return deviceInfos, nil
}

// DisableMFA 禁用用户MFA
func (s *MFAManagementService) DisableMFA(ctx context.Context, userID, tenantID uuid.UUID) error {
	tx := s.db.GetDB().WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 禁用用户MFA
	err := tx.Model(&models.User{}).
		Where("id = ? AND tenant_id = ?", userID, tenantID).
		Updates(map[string]interface{}{
			"two_factor_enabled": false,
			"two_factor_secret":  "",
		}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("禁用MFA失败: %w", err)
	}

	// 删除所有MFA设备
	err = tx.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.UserMFADevice{}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("删除MFA设备失败: %w", err)
	}

	// 删除所有备用验证码
	err = tx.Where("user_id = ? AND tenant_id = ?", userID, tenantID).
		Delete(&models.UserMFABackupCode{}).Error
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("删除备用验证码失败: %w", err)
	}

	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("提交事务失败: %w", err)
	}

	return nil
}

// validateTOTPCode 验证TOTP代码
func (s *MFAManagementService) validateTOTPCode(tx *gorm.DB, userID, tenantID uuid.UUID, code string, deviceID *uuid.UUID) (bool, error) {
	var device models.UserMFADevice
	query := tx.Where("user_id = ? AND tenant_id = ? AND device_type = 'totp' AND is_active = true", userID, tenantID)
	
	if deviceID != nil {
		query = query.Where("id = ?", *deviceID)
	}
	
	err := query.First(&device).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, fmt.Errorf("TOTP设备不存在")
		}
		return false, fmt.Errorf("查询TOTP设备失败: %w", err)
	}

	return s.mfaService.ValidateCode(device.Secret, code), nil
}

// validateBackupCode 验证备用验证码
func (s *MFAManagementService) validateBackupCode(tx *gorm.DB, userID, tenantID uuid.UUID, code string) (bool, error) {
	if !s.mfaService.IsValidBackupCode(code) {
		return false, fmt.Errorf("备用码格式无效")
	}

	hashedCode := s.hashBackupCode(code)
	
	var backupCode models.UserMFABackupCode
	err := tx.Where("user_id = ? AND tenant_id = ? AND code = ? AND used = false", userID, tenantID, hashedCode).
		First(&backupCode).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil // 备用码无效或已使用
		}
		return false, fmt.Errorf("查询备用验证码失败: %w", err)
	}

	// 标记备用码为已使用
	now := time.Now()
	err = tx.Model(&backupCode).Updates(map[string]interface{}{
		"used":    true,
		"used_at": &now,
	}).Error
	if err != nil {
		return false, fmt.Errorf("更新备用验证码状态失败: %w", err)
	}

	return true, nil
}

// hashBackupCode 对备用验证码进行哈希
func (s *MFAManagementService) hashBackupCode(code string) string {
	hash := sha256.Sum256([]byte(code + "mfa_backup_salt"))
	return hex.EncodeToString(hash[:])
}