package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/cloud-platform/collaborative-dev/shared/models"
)

type APITokenService struct {
	db *gorm.DB
}

func NewAPITokenService(db *gorm.DB) *APITokenService {
	return &APITokenService{db: db}
}

// CreateAPIToken creates a new API token
func (s *APITokenService) CreateAPIToken(ctx context.Context, tenantID, userID uuid.UUID, req *CreateAPITokenRequest) (*models.APIToken, string, error) {
	// Generate secure token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return nil, "", fmt.Errorf("failed to generate token: %w", err)
	}

	// Create token string with prefix
	tokenString := fmt.Sprintf("cdt_%s", hex.EncodeToString(tokenBytes))
	tokenPrefix := tokenString[:10] // First 10 characters for display

	// Hash the token for storage
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	// Convert scopes to JSON
	scopesJSON, err := json.Marshal(req.Scopes)
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize scopes: %w", err)
	}

	// Convert permissions to JSON
	permissionsJSON, err := json.Marshal(req.Permissions)
	if err != nil {
		return nil, "", fmt.Errorf("failed to serialize permissions: %w", err)
	}

	// Create token record
	token := &models.APIToken{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Description,
		TokenHash:    tokenHash,
		TokenPrefix:  tokenPrefix,
		Scopes:       scopesJSON,
		Permissions:  permissionsJSON,
		Status:       "active",
		RateLimitRPS: req.RateLimitRPS,
		ExpiresAt:    req.ExpiresAt,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Start transaction
	tx := s.db.WithContext(ctx).Begin()

	// Create token
	if err := tx.Create(token).Error; err != nil {
		tx.Rollback()
		return nil, "", fmt.Errorf("failed to create API token: %w", err)
	}

	// Create token-scope relationships
	for _, scopeName := range req.Scopes {
		var scope models.APIScope
		if err := tx.Where("name = ?", scopeName).First(&scope).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				tx.Rollback()
				return nil, "", fmt.Errorf("invalid scope: %s", scopeName)
			}
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to find scope: %w", err)
		}

		tokenScope := &models.APITokenScope{
			ID:        uuid.New(),
			TenantID:  tenantID,
			TokenID:   token.ID,
			ScopeID:   scope.ID,
			GrantedAt: time.Now(),
		}

		if err := tx.Create(tokenScope).Error; err != nil {
			tx.Rollback()
			return nil, "", fmt.Errorf("failed to create token scope: %w", err)
		}
	}

	tx.Commit()

	return token, tokenString, nil
}

// GetAPITokens retrieves API tokens for a user
func (s *APITokenService) GetAPITokens(ctx context.Context, tenantID, userID uuid.UUID) ([]models.APIToken, error) {
	var tokens []models.APIToken

	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND status != ?", tenantID, userID, "deleted").
		Order("created_at DESC").
		Find(&tokens).Error

	return tokens, err
}

// GetAPITokenByID retrieves a specific API token
func (s *APITokenService) GetAPITokenByID(ctx context.Context, tenantID, tokenID uuid.UUID) (*models.APIToken, error) {
	var token models.APIToken

	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status != ?", tokenID, tenantID, "deleted").
		First(&token).Error

	if err != nil {
		return nil, err
	}

	return &token, nil
}

// ValidateAPIToken validates a token and returns the associated token record
func (s *APITokenService) ValidateAPIToken(ctx context.Context, tokenString string) (*models.APIToken, error) {
	if !strings.HasPrefix(tokenString, "cdt_") {
		return nil, fmt.Errorf("invalid token format")
	}

	// Hash the provided token
	hash := sha256.Sum256([]byte(tokenString))
	tokenHash := hex.EncodeToString(hash[:])

	var token models.APIToken
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("token_hash = ? AND status = ?", tokenHash, "active").
		First(&token).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("invalid token")
		}
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	// Check if token is expired
	if token.ExpiresAt != nil && token.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("token expired")
	}

	return &token, nil
}

// UpdateTokenUsage updates token usage statistics
func (s *APITokenService) UpdateTokenUsage(ctx context.Context, tokenID uuid.UUID, ipAddress string) error {
	now := time.Now()

	updates := map[string]interface{}{
		"last_used_at": now,
		"last_used_ip": ipAddress,
		"use_count":    gorm.Expr("use_count + 1"),
		"updated_at":   now,
	}

	return s.db.WithContext(ctx).
		Model(&models.APIToken{}).
		Where("id = ?", tokenID).
		Updates(updates).Error
}

// RevokeAPIToken revokes an API token
func (s *APITokenService) RevokeAPIToken(ctx context.Context, tenantID, tokenID, revokedBy uuid.UUID) error {
	now := time.Now()

	updates := map[string]interface{}{
		"status":     "revoked",
		"revoked_at": now,
		"revoked_by": revokedBy,
		"updated_at": now,
	}

	return s.db.WithContext(ctx).
		Model(&models.APIToken{}).
		Where("id = ? AND tenant_id = ?", tokenID, tenantID).
		Updates(updates).Error
}

// UpdateAPIToken updates token metadata
func (s *APITokenService) UpdateAPIToken(ctx context.Context, tenantID, tokenID uuid.UUID, req *UpdateAPITokenRequest) error {
	updates := map[string]interface{}{
		"updated_at": time.Now(),
	}

	if req.Name != "" {
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.RateLimitRPS > 0 {
		updates["rate_limit_rps"] = req.RateLimitRPS
	}

	if req.ExpiresAt != nil {
		updates["expires_at"] = req.ExpiresAt
	}

	return s.db.WithContext(ctx).
		Model(&models.APIToken{}).
		Where("id = ? AND tenant_id = ?", tokenID, tenantID).
		Updates(updates).Error
}

// GetTokenScopes retrieves scopes for a token
func (s *APITokenService) GetTokenScopes(ctx context.Context, tokenID uuid.UUID) ([]string, error) {
	var token models.APIToken
	if err := s.db.WithContext(ctx).Where("id = ?", tokenID).First(&token).Error; err != nil {
		return nil, err
	}

	var scopes []string
	if err := json.Unmarshal(token.Scopes, &scopes); err != nil {
		return nil, fmt.Errorf("failed to parse token scopes: %w", err)
	}

	return scopes, nil
}

// CheckTokenPermission checks if a token has a specific permission
func (s *APITokenService) CheckTokenPermission(ctx context.Context, tokenID uuid.UUID, resource, action string) (bool, error) {
	scopes, err := s.GetTokenScopes(ctx, tokenID)
	if err != nil {
		return false, err
	}

	// Check if token has required scope
	requiredScope := fmt.Sprintf("%s:%s", resource, action)
	for _, scope := range scopes {
		if scope == requiredScope || scope == fmt.Sprintf("%s:admin", resource) {
			return true, nil
		}

		// Check wildcard permissions
		if strings.HasSuffix(scope, ":admin") && strings.HasPrefix(scope, resource) {
			return true, nil
		}
	}

	return false, nil
}

// LogTokenUsage logs detailed token usage for analytics
func (s *APITokenService) LogTokenUsage(ctx context.Context, usage *models.APITokenUsage) error {
	usage.ID = uuid.New()
	usage.CreatedAt = time.Now()

	return s.db.WithContext(ctx).Create(usage).Error
}

// GetTokenUsageStats retrieves usage statistics for a token
func (s *APITokenService) GetTokenUsageStats(ctx context.Context, tenantID, tokenID uuid.UUID, days int) (*TokenUsageStats, error) {
	startDate := time.Now().AddDate(0, 0, -days)

	var stats TokenUsageStats

	// Total requests
	err := s.db.WithContext(ctx).
		Model(&models.APITokenUsage{}).
		Where("tenant_id = ? AND token_id = ? AND created_at >= ?", tenantID, tokenID, startDate).
		Count(&stats.TotalRequests).Error
	if err != nil {
		return nil, err
	}

	// Successful requests
	err = s.db.WithContext(ctx).
		Model(&models.APITokenUsage{}).
		Where("tenant_id = ? AND token_id = ? AND created_at >= ? AND status_code < 400", tenantID, tokenID, startDate).
		Count(&stats.SuccessfulRequests).Error
	if err != nil {
		return nil, err
	}

	// Error requests
	stats.ErrorRequests = stats.TotalRequests - stats.SuccessfulRequests

	// Average response time
	var avgResponseTime float64
	err = s.db.WithContext(ctx).
		Model(&models.APITokenUsage{}).
		Where("tenant_id = ? AND token_id = ? AND created_at >= ?", tenantID, tokenID, startDate).
		Select("AVG(response_time)").
		Scan(&avgResponseTime).Error
	if err != nil {
		return nil, err
	}
	stats.AvgResponseTime = int64(avgResponseTime)

	// Most used endpoints
	var endpoints []EndpointUsage
	err = s.db.WithContext(ctx).
		Model(&models.APITokenUsage{}).
		Select("endpoint, COUNT(*) as count").
		Where("tenant_id = ? AND token_id = ? AND created_at >= ?", tenantID, tokenID, startDate).
		Group("endpoint").
		Order("count DESC").
		Limit(10).
		Scan(&endpoints).Error
	if err != nil {
		return nil, err
	}
	stats.TopEndpoints = endpoints

	return &stats, nil
}

// CheckRateLimit checks if token has exceeded rate limit
func (s *APITokenService) CheckRateLimit(ctx context.Context, tokenID uuid.UUID) (bool, error) {
	var token models.APIToken
	if err := s.db.WithContext(ctx).Where("id = ?", tokenID).First(&token).Error; err != nil {
		return false, err
	}

	windowStart := time.Now().Truncate(time.Minute)

	var rateLimit models.APIRateLimit
	err := s.db.WithContext(ctx).
		Where("token_id = ? AND window_start = ?", tokenID, windowStart).
		First(&rateLimit).Error

	if err == gorm.ErrRecordNotFound {
		// Create new rate limit window
		rateLimit = models.APIRateLimit{
			ID:           uuid.New(),
			TokenID:      tokenID,
			WindowStart:  windowStart,
			WindowSize:   60, // 1 minute window
			RequestCount: 0,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := s.db.WithContext(ctx).Create(&rateLimit).Error; err != nil {
			return false, err
		}
	} else if err != nil {
		return false, err
	}

	// Check if rate limit exceeded
	if rateLimit.RequestCount >= token.RateLimitRPS*60 { // RPS * 60 seconds
		return false, nil // Rate limit exceeded
	}

	// Increment request count
	s.db.WithContext(ctx).
		Model(&rateLimit).
		UpdateColumn("request_count", gorm.Expr("request_count + 1"))

	return true, nil // Rate limit OK
}

// GetAvailableScopes retrieves all available API scopes
func (s *APITokenService) GetAvailableScopes(ctx context.Context) ([]models.APIScope, error) {
	var scopes []models.APIScope

	err := s.db.WithContext(ctx).
		Order("category, name").
		Find(&scopes).Error

	return scopes, err
}

// CleanupExpiredTokens removes expired and revoked tokens
func (s *APITokenService) CleanupExpiredTokens(ctx context.Context) (int64, error) {
	now := time.Now()

	// Mark expired tokens
	result := s.db.WithContext(ctx).
		Model(&models.APIToken{}).
		Where("expires_at < ? AND status = ?", now, "active").
		Updates(map[string]interface{}{
			"status":     "expired",
			"updated_at": now,
		})

	if result.Error != nil {
		return 0, result.Error
	}

	// Clean up old token usage records (older than 90 days)
	oldDate := now.AddDate(0, 0, -90)
	s.db.WithContext(ctx).
		Where("created_at < ?", oldDate).
		Delete(&models.APITokenUsage{})

	// Clean up old rate limit records (older than 1 day)
	oldRateDate := now.AddDate(0, 0, -1)
	s.db.WithContext(ctx).
		Where("created_at < ?", oldRateDate).
		Delete(&models.APIRateLimit{})

	return result.RowsAffected, nil
}

// Request/Response types
type CreateAPITokenRequest struct {
	Name         string     `json:"name" binding:"required,min=1,max=255"`
	Description  string     `json:"description" binding:"max=1000"`
	Scopes       []string   `json:"scopes" binding:"required,min=1"`
	Permissions  []string   `json:"permissions"`
	RateLimitRPS int        `json:"rate_limit_rps" binding:"min=1,max=10000"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type UpdateAPITokenRequest struct {
	Name         string     `json:"name" binding:"max=255"`
	Description  string     `json:"description" binding:"max=1000"`
	RateLimitRPS int        `json:"rate_limit_rps" binding:"min=1,max=10000"`
	ExpiresAt    *time.Time `json:"expires_at"`
}

type TokenUsageStats struct {
	TotalRequests      int64           `json:"total_requests"`
	SuccessfulRequests int64           `json:"successful_requests"`
	ErrorRequests      int64           `json:"error_requests"`
	AvgResponseTime    int64           `json:"avg_response_time"`
	TopEndpoints       []EndpointUsage `json:"top_endpoints"`
}

type EndpointUsage struct {
	Endpoint string `json:"endpoint"`
	Count    int64  `json:"count"`
}
