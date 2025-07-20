package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/example/cloud-platform/shared/models"
)

type SSOService struct {
	db *gorm.DB
}

func NewSSOService(db *gorm.DB) *SSOService {
	return &SSOService{db: db}
}

// CreateSSOProvider creates a new SSO provider configuration
func (s *SSOService) CreateSSOProvider(ctx context.Context, tenantID uuid.UUID, provider *models.SSOProvider) error {
	provider.ID = uuid.New()
	provider.TenantID = tenantID
	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()
	
	// Validate configuration based on provider type
	if err := s.validateProviderConfiguration(provider); err != nil {
		return fmt.Errorf("invalid provider configuration: %w", err)
	}
	
	return s.db.WithContext(ctx).Create(provider).Error
}

// GetSSOProviders retrieves all SSO providers for a tenant
func (s *SSOService) GetSSOProviders(ctx context.Context, tenantID uuid.UUID) ([]models.SSOProvider, error) {
	var providers []models.SSOProvider
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND status != ?", tenantID, "deleted").
		Find(&providers).Error
	return providers, err
}

// GetSSOProviderByID retrieves a specific SSO provider
func (s *SSOService) GetSSOProviderByID(ctx context.Context, tenantID, providerID uuid.UUID) (*models.SSOProvider, error) {
	var provider models.SSOProvider
	err := s.db.WithContext(ctx).
		Where("id = ? AND tenant_id = ? AND status != ?", providerID, tenantID, "deleted").
		First(&provider).Error
	if err != nil {
		return nil, err
	}
	return &provider, nil
}

// InitiateSSO starts the SSO authentication flow
func (s *SSOService) InitiateSSO(ctx context.Context, tenantID, providerID uuid.UUID, redirectURI string) (*models.SSOSession, string, error) {
	// Get provider configuration
	provider, err := s.GetSSOProviderByID(ctx, tenantID, providerID)
	if err != nil {
		return nil, "", fmt.Errorf("provider not found: %w", err)
	}
	
	if provider.Status != "active" {
		return nil, "", fmt.Errorf("provider is not active")
	}
	
	// Generate state and nonce
	state := s.generateSecureToken(32)
	nonce := s.generateSecureToken(32)
	codeChallenge := ""
	
	// Create SSO session
	session := &models.SSOSession{
		ID:            uuid.New(),
		TenantID:      tenantID,
		ProviderID:    providerID,
		State:         state,
		Nonce:         nonce,
		CodeChallenge: codeChallenge,
		RedirectURI:   redirectURI,
		Status:        "pending",
		ExpiresAt:     time.Now().Add(15 * time.Minute), // 15 minutes expiry
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
	
	if err := s.db.WithContext(ctx).Create(session).Error; err != nil {
		return nil, "", fmt.Errorf("failed to create SSO session: %w", err)
	}
	
	// Generate authorization URL based on provider type
	authURL, err := s.generateAuthorizationURL(provider, session)
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate authorization URL: %w", err)
	}
	
	return session, authURL, nil
}

// CompleteSSO completes the SSO authentication flow
func (s *SSOService) CompleteSSO(ctx context.Context, state, code string) (*models.User, error) {
	// Find SSO session by state
	var session models.SSOSession
	err := s.db.WithContext(ctx).
		Preload("Provider").
		Where("state = ? AND status = ? AND expires_at > ?", state, "pending", time.Now()).
		First(&session).Error
	if err != nil {
		return nil, fmt.Errorf("invalid or expired SSO session: %w", err)
	}
	
	// Exchange code for user information
	userInfo, err := s.exchangeCodeForUserInfo(&session, code)
	if err != nil {
		// Update session status to failed
		s.db.WithContext(ctx).Model(&session).Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now(),
		})
		return nil, fmt.Errorf("failed to exchange code: %w", err)
	}
	
	// Find or create user based on external user ID
	user, err := s.findOrCreateUser(ctx, &session, userInfo)
	if err != nil {
		s.db.WithContext(ctx).Model(&session).Updates(map[string]interface{}{
			"status":     "failed",
			"updated_at": time.Now(),
		})
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}
	
	// Update session with user ID and mark as completed
	err = s.db.WithContext(ctx).Model(&session).Updates(map[string]interface{}{
		"user_id":          user.ID,
		"external_user_id": userInfo.ExternalUserID,
		"status":           "completed",
		"updated_at":       time.Now(),
	}).Error
	if err != nil {
		return nil, fmt.Errorf("failed to update SSO session: %w", err)
	}
	
	return user, nil
}

// UpdateUserMapping updates the mapping between external and internal users
func (s *SSOService) UpdateUserMapping(ctx context.Context, tenantID, userID, providerID uuid.UUID, userInfo *ExternalUserInfo) error {
	var mapping models.SSOUserMapping
	
	// Check if mapping already exists
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND user_id = ? AND provider_id = ?", tenantID, userID, providerID).
		First(&mapping).Error
	
	if err == gorm.ErrRecordNotFound {
		// Create new mapping
		mapping = models.SSOUserMapping{
			ID:             uuid.New(),
			TenantID:       tenantID,
			UserID:         userID,
			ProviderID:     providerID,
			ExternalUserID: userInfo.ExternalUserID,
			ExternalEmail:  userInfo.Email,
			ExternalName:   userInfo.Name,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
		
		if userInfo.Attributes != nil {
			attributesJSON, _ := json.Marshal(userInfo.Attributes)
			mapping.Attributes = attributesJSON
		}
		
		return s.db.WithContext(ctx).Create(&mapping).Error
	} else if err != nil {
		return err
	}
	
	// Update existing mapping
	updates := map[string]interface{}{
		"external_email": userInfo.Email,
		"external_name":  userInfo.Name,
		"updated_at":     time.Now(),
	}
	
	if userInfo.Attributes != nil {
		attributesJSON, _ := json.Marshal(userInfo.Attributes)
		updates["attributes"] = attributesJSON
	}
	
	return s.db.WithContext(ctx).Model(&mapping).Updates(updates).Error
}

// Helper types and methods

type ExternalUserInfo struct {
	ExternalUserID string                 `json:"external_user_id"`
	Email          string                 `json:"email"`
	Name           string                 `json:"name"`
	FirstName      string                 `json:"first_name"`
	LastName       string                 `json:"last_name"`
	Attributes     map[string]interface{} `json:"attributes"`
}

func (s *SSOService) generateSecureToken(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (s *SSOService) generateCodeChallenge(codeVerifier string) string {
	hash := sha256.Sum256([]byte(codeVerifier))
	return base64.URLEncoding.WithoutPadding().EncodeToString(hash[:])
}

func (s *SSOService) validateProviderConfiguration(provider *models.SSOProvider) error {
	switch provider.Type {
	case "saml":
		var config models.SAMLConfig
		if err := json.Unmarshal(provider.Configuration, &config); err != nil {
			return fmt.Errorf("invalid SAML configuration: %w", err)
		}
		if config.EntityID == "" || config.SSOURL == "" {
			return fmt.Errorf("SAML configuration missing required fields")
		}
	case "oauth2":
		var config models.OAuth2Config
		if err := json.Unmarshal(provider.Configuration, &config); err != nil {
			return fmt.Errorf("invalid OAuth2 configuration: %w", err)
		}
		if config.ClientID == "" || config.AuthorizationURL == "" || config.TokenURL == "" {
			return fmt.Errorf("OAuth2 configuration missing required fields")
		}
	case "oidc":
		var config models.OIDCConfig
		if err := json.Unmarshal(provider.Configuration, &config); err != nil {
			return fmt.Errorf("invalid OIDC configuration: %w", err)
		}
		if config.ClientID == "" || config.IssuerURL == "" {
			return fmt.Errorf("OIDC configuration missing required fields")
		}
	default:
		return fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
	return nil
}

func (s *SSOService) generateAuthorizationURL(provider *models.SSOProvider, session *models.SSOSession) (string, error) {
	switch provider.Type {
	case "oauth2":
		return s.generateOAuth2AuthURL(provider, session)
	case "oidc":
		return s.generateOIDCAuthURL(provider, session)
	case "saml":
		return s.generateSAMLAuthURL(provider, session)
	default:
		return "", fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}

func (s *SSOService) generateOAuth2AuthURL(provider *models.SSOProvider, session *models.SSOSession) (string, error) {
	var config models.OAuth2Config
	if err := json.Unmarshal(provider.Configuration, &config); err != nil {
		return "", err
	}
	
	// Build OAuth2 authorization URL
	authURL := fmt.Sprintf("%s?client_id=%s&response_type=code&redirect_uri=%s&state=%s",
		config.AuthorizationURL,
		config.ClientID,
		config.RedirectURL,
		session.State,
	)
	
	if len(config.Scopes) > 0 {
		authURL += "&scope=" + fmt.Sprintf("%v", config.Scopes)
	}
	
	return authURL, nil
}

func (s *SSOService) generateOIDCAuthURL(provider *models.SSOProvider, session *models.SSOSession) (string, error) {
	var config models.OIDCConfig
	if err := json.Unmarshal(provider.Configuration, &config); err != nil {
		return "", err
	}
	
	// Build OIDC authorization URL
	authURL := fmt.Sprintf("%s/auth?client_id=%s&response_type=code&redirect_uri=%s&state=%s&nonce=%s",
		config.IssuerURL,
		config.ClientID,
		config.RedirectURL,
		session.State,
		session.Nonce,
	)
	
	if len(config.Scopes) > 0 {
		authURL += "&scope=" + fmt.Sprintf("%v", config.Scopes)
	}
	
	return authURL, nil
}

func (s *SSOService) generateSAMLAuthURL(provider *models.SSOProvider, session *models.SSOSession) (string, error) {
	var config models.SAMLConfig
	if err := json.Unmarshal(provider.Configuration, &config); err != nil {
		return "", err
	}
	
	// For SAML, we would generate a SAML AuthnRequest
	// This is a simplified version - in production, you'd use a SAML library
	authURL := fmt.Sprintf("%s?SAMLRequest=<encoded_request>&RelayState=%s",
		config.SSOURL,
		session.State,
	)
	
	return authURL, nil
}

func (s *SSOService) exchangeCodeForUserInfo(session *models.SSOSession, code string) (*ExternalUserInfo, error) {
	switch session.Provider.Type {
	case "oauth2":
		return s.exchangeOAuth2Code(session, code)
	case "oidc":
		return s.exchangeOIDCCode(session, code)
	case "saml":
		return s.processSAMLResponse(session, code)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", session.Provider.Type)
	}
}

func (s *SSOService) exchangeOAuth2Code(session *models.SSOSession, code string) (*ExternalUserInfo, error) {
	// Simplified OAuth2 code exchange - in production, use proper HTTP client
	// This would involve:
	// 1. Exchange code for access token
	// 2. Use access token to get user info
	// 3. Map attributes according to provider configuration
	
	return &ExternalUserInfo{
		ExternalUserID: "oauth2_user_123",
		Email:          "user@example.com",
		Name:           "OAuth2 User",
		FirstName:      "OAuth2",
		LastName:       "User",
	}, nil
}

func (s *SSOService) exchangeOIDCCode(session *models.SSOSession, code string) (*ExternalUserInfo, error) {
	// Simplified OIDC code exchange - in production, use proper OIDC library
	// This would involve:
	// 1. Exchange code for ID token and access token
	// 2. Validate ID token
	// 3. Extract user claims from ID token
	// 4. Optionally call UserInfo endpoint
	
	return &ExternalUserInfo{
		ExternalUserID: "oidc_user_123",
		Email:          "user@example.com",
		Name:           "OIDC User",
		FirstName:      "OIDC",
		LastName:       "User",
	}, nil
}

func (s *SSOService) processSAMLResponse(session *models.SSOSession, samlResponse string) (*ExternalUserInfo, error) {
	// Simplified SAML response processing - in production, use proper SAML library
	// This would involve:
	// 1. Validate SAML response signature
	// 2. Extract assertions
	// 3. Map attributes according to provider configuration
	
	return &ExternalUserInfo{
		ExternalUserID: "saml_user_123",
		Email:          "user@example.com",
		Name:           "SAML User",
		FirstName:      "SAML",
		LastName:       "User",
	}, nil
}

func (s *SSOService) findOrCreateUser(ctx context.Context, session *models.SSOSession, userInfo *ExternalUserInfo) (*models.User, error) {
	// Check if user mapping already exists
	var mapping models.SSOUserMapping
	err := s.db.WithContext(ctx).
		Preload("User").
		Where("tenant_id = ? AND provider_id = ? AND external_user_id = ?",
			session.TenantID, session.ProviderID, userInfo.ExternalUserID).
		First(&mapping).Error
	
	if err == nil {
		// User mapping exists, update and return user
		s.UpdateUserMapping(ctx, session.TenantID, mapping.UserID, session.ProviderID, userInfo)
		return mapping.User, nil
	}
	
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Check if user exists by email
	var existingUser models.User
	err = s.db.WithContext(ctx).
		Where("tenant_id = ? AND email = ?", session.TenantID, userInfo.Email).
		First(&existingUser).Error
	
	if err == nil {
		// User exists, create mapping
		s.UpdateUserMapping(ctx, session.TenantID, existingUser.ID, session.ProviderID, userInfo)
		return &existingUser, nil
	}
	
	if err != gorm.ErrRecordNotFound {
		return nil, err
	}
	
	// Create new user
	newUser := &models.User{
		ID:          uuid.New(),
		TenantID:    session.TenantID,
		Email:       userInfo.Email,
		Username:    userInfo.Email, // Use email as username for SSO users
		FirstName:   userInfo.FirstName,
		LastName:    userInfo.LastName,
		Status:      "active",
		IsEmailVerified: true, // SSO users are considered verified
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
	
	// Create user in transaction
	tx := s.db.WithContext(ctx).Begin()
	if err := tx.Create(newUser).Error; err != nil {
		tx.Rollback()
		return nil, err
	}
	
	// Create user mapping
	if err := s.updateUserMappingInTx(tx, session.TenantID, newUser.ID, session.ProviderID, userInfo); err != nil {
		tx.Rollback()
		return nil, err
	}
	
	tx.Commit()
	return newUser, nil
}

func (s *SSOService) updateUserMappingInTx(tx *gorm.DB, tenantID, userID, providerID uuid.UUID, userInfo *ExternalUserInfo) error {
	mapping := models.SSOUserMapping{
		ID:             uuid.New(),
		TenantID:       tenantID,
		UserID:         userID,
		ProviderID:     providerID,
		ExternalUserID: userInfo.ExternalUserID,
		ExternalEmail:  userInfo.Email,
		ExternalName:   userInfo.Name,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	
	if userInfo.Attributes != nil {
		attributesJSON, _ := json.Marshal(userInfo.Attributes)
		mapping.Attributes = attributesJSON
	}
	
	return tx.Create(&mapping).Error
}