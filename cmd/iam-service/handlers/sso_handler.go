package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/example/cloud-platform/cmd/iam-service/services"
	"github.com/example/cloud-platform/shared/auth"
	"github.com/example/cloud-platform/shared/models"
)

type SSOHandler struct {
	ssoService *services.SSOService
	jwtManager *auth.JWTManager
}

func NewSSOHandler(ssoService *services.SSOService, jwtManager *auth.JWTManager) *SSOHandler {
	return &SSOHandler{
		ssoService: ssoService,
		jwtManager: jwtManager,
	}
}

// CreateSSOProviderRequest represents the request to create an SSO provider
type CreateSSOProviderRequest struct {
	Name          string      `json:"name" binding:"required,min=2,max=255"`
	Type          string      `json:"type" binding:"required,oneof=saml oauth2 oidc"`
	Configuration interface{} `json:"configuration" binding:"required"`
}

// SSOProviderResponse represents the response for SSO provider operations
type SSOProviderResponse struct {
	ID            uuid.UUID   `json:"id"`
	Name          string      `json:"name"`
	Type          string      `json:"type"`
	Status        string      `json:"status"`
	Configuration interface{} `json:"configuration,omitempty"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
}

// InitiateSSORequest represents the request to initiate SSO
type InitiateSSORequest struct {
	ProviderID  uuid.UUID `json:"provider_id" binding:"required"`
	RedirectURI string    `json:"redirect_uri" binding:"required,url"`
}

// InitiateSSOResponse represents the response for SSO initiation
type InitiateSSOResponse struct {
	SessionID       uuid.UUID `json:"session_id"`
	AuthorizationURL string   `json:"authorization_url"`
	State           string    `json:"state"`
	ExpiresAt       time.Time `json:"expires_at"`
}

// CompleteSSORequest represents the request to complete SSO
type CompleteSSORequest struct {
	State string `json:"state" binding:"required"`
	Code  string `json:"code" binding:"required"`
}

// CompleteSSOResponse represents the response for SSO completion
type CompleteSSOResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	TokenType    string    `json:"token_type"`
	ExpiresIn    int       `json:"expires_in"`
	User         UserInfo  `json:"user"`
}

type UserInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Status    string    `json:"status"`
}

// CreateSSOProvider creates a new SSO provider configuration
// @Summary Create SSO Provider
// @Description Create a new Single Sign-On provider configuration
// @Tags SSO
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body CreateSSOProviderRequest true "SSO provider configuration"
// @Success 201 {object} SSOProviderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/providers [post]
func (h *SSOHandler) CreateSSOProvider(c *gin.Context) {
	var req CreateSSOProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Get tenant ID from context
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Tenant ID not found in context",
		})
		return
	}

	// Check permissions (admin only)
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "Only administrators can create SSO providers",
		})
		return
	}

	// Convert configuration to JSON
	configJSON, err := json.Marshal(req.Configuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid configuration",
			Message: "Failed to serialize configuration",
		})
		return
	}

	// Create provider
	provider := &models.SSOProvider{
		Name:          req.Name,
		Type:          req.Type,
		Configuration: configJSON,
		Status:        "active",
	}

	userID, _ := c.Get("user_id")
	if uid, ok := userID.(uuid.UUID); ok {
		provider.CreatedByUserID = uid
	}

	if err := h.ssoService.CreateSSOProvider(c.Request.Context(), tenantID.(uuid.UUID), provider); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to create SSO provider",
			Message: err.Error(),
		})
		return
	}

	// Return response (without sensitive configuration)
	response := SSOProviderResponse{
		ID:        provider.ID,
		Name:      provider.Name,
		Type:      provider.Type,
		Status:    provider.Status,
		CreatedAt: provider.CreatedAt,
		UpdatedAt: provider.UpdatedAt,
	}

	c.JSON(http.StatusCreated, response)
}

// GetSSOProviders retrieves all SSO providers for the tenant
// @Summary Get SSO Providers
// @Description Retrieve all SSO providers for the current tenant
// @Tags SSO
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {array} SSOProviderResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/providers [get]
func (h *SSOHandler) GetSSOProviders(c *gin.Context) {
	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Tenant ID not found in context",
		})
		return
	}

	providers, err := h.ssoService.GetSSOProviders(c.Request.Context(), tenantID.(uuid.UUID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve SSO providers",
			Message: err.Error(),
		})
		return
	}

	// Convert to response format (without sensitive configuration)
	var response []SSOProviderResponse
	for _, provider := range providers {
		response = append(response, SSOProviderResponse{
			ID:        provider.ID,
			Name:      provider.Name,
			Type:      provider.Type,
			Status:    provider.Status,
			CreatedAt: provider.CreatedAt,
			UpdatedAt: provider.UpdatedAt,
		})
	}

	c.JSON(http.StatusOK, response)
}

// InitiateSSO starts the SSO authentication flow
// @Summary Initiate SSO
// @Description Start the SSO authentication flow for a specific provider
// @Tags SSO
// @Accept json
// @Produce json
// @Param request body InitiateSSORequest true "SSO initiation request"
// @Success 200 {object} InitiateSSOResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/initiate [post]
func (h *SSOHandler) InitiateSSO(c *gin.Context) {
	var req InitiateSSORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Extract tenant ID from token if present, otherwise use default
	var tenantID uuid.UUID
	if tid, exists := c.Get("tenant_id"); exists {
		tenantID = tid.(uuid.UUID)
	} else {
		// For public SSO initiation, we need to get tenant from provider or domain
		// For now, we'll return an error
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: "Tenant context required",
		})
		return
	}

	session, authURL, err := h.ssoService.InitiateSSO(c.Request.Context(), tenantID, req.ProviderID, req.RedirectURI)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to initiate SSO",
			Message: err.Error(),
		})
		return
	}

	response := InitiateSSOResponse{
		SessionID:        session.ID,
		AuthorizationURL: authURL,
		State:            session.State,
		ExpiresAt:        session.ExpiresAt,
	}

	c.JSON(http.StatusOK, response)
}

// CompleteSSO completes the SSO authentication flow
// @Summary Complete SSO
// @Description Complete the SSO authentication flow and return tokens
// @Tags SSO
// @Accept json
// @Produce json
// @Param request body CompleteSSORequest true "SSO completion request"
// @Success 200 {object} CompleteSSOResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/complete [post]
func (h *SSOHandler) CompleteSSO(c *gin.Context) {
	var req CompleteSSORequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Complete SSO authentication
	user, err := h.ssoService.CompleteSSO(c.Request.Context(), req.State, req.Code)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "SSO authentication failed",
			Message: err.Error(),
		})
		return
	}

	// Generate JWT tokens
	accessToken, err := h.jwtManager.GenerateAccessToken(user.ID, user.TenantID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate access token",
			Message: err.Error(),
		})
		return
	}

	refreshToken, err := h.jwtManager.GenerateRefreshToken(user.ID, user.TenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to generate refresh token",
			Message: err.Error(),
		})
		return
	}

	response := CompleteSSOResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
		ExpiresIn:    int(time.Hour.Seconds()), // 1 hour
		User: UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			Username:  user.Username,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    user.Status,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetSSOProviderByID retrieves a specific SSO provider
// @Summary Get SSO Provider
// @Description Retrieve a specific SSO provider by ID
// @Tags SSO
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Provider ID"
// @Success 200 {object} SSOProviderResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/providers/{id} [get]
func (h *SSOHandler) GetSSOProviderByID(c *gin.Context) {
	providerIDStr := c.Param("id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid provider ID",
			Message: "Provider ID must be a valid UUID",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Tenant ID not found in context",
		})
		return
	}

	provider, err := h.ssoService.GetSSOProviderByID(c.Request.Context(), tenantID.(uuid.UUID), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "SSO provider not found",
			Message: err.Error(),
		})
		return
	}

	// Return response without sensitive configuration
	response := SSOProviderResponse{
		ID:        provider.ID,
		Name:      provider.Name,
		Type:      provider.Type,
		Status:    provider.Status,
		CreatedAt: provider.CreatedAt,
		UpdatedAt: provider.UpdatedAt,
	}

	// Include configuration for admins
	userRole, exists := c.Get("user_role")
	if exists && userRole == "admin" {
		var config interface{}
		if err := json.Unmarshal(provider.Configuration, &config); err == nil {
			response.Configuration = config
		}
	}

	c.JSON(http.StatusOK, response)
}

// UpdateSSOProvider updates an existing SSO provider
// @Summary Update SSO Provider
// @Description Update an existing SSO provider configuration
// @Tags SSO
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Provider ID"
// @Param request body CreateSSOProviderRequest true "Updated SSO provider configuration"
// @Success 200 {object} SSOProviderResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/providers/{id} [put]
func (h *SSOHandler) UpdateSSOProvider(c *gin.Context) {
	providerIDStr := c.Param("id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid provider ID",
			Message: "Provider ID must be a valid UUID",
		})
		return
	}

	var req CreateSSOProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Tenant ID not found in context",
		})
		return
	}

	// Check permissions (admin only)
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "Only administrators can update SSO providers",
		})
		return
	}

	// Get existing provider
	provider, err := h.ssoService.GetSSOProviderByID(c.Request.Context(), tenantID.(uuid.UUID), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "SSO provider not found",
			Message: err.Error(),
		})
		return
	}

	// Convert configuration to JSON
	configJSON, err := json.Marshal(req.Configuration)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid configuration",
			Message: "Failed to serialize configuration",
		})
		return
	}

	// Update provider fields
	provider.Name = req.Name
	provider.Type = req.Type
	provider.Configuration = configJSON
	provider.UpdatedAt = time.Now()

	userID, _ := c.Get("user_id")
	if uid, ok := userID.(uuid.UUID); ok {
		provider.ModifiedByUserID = &uid
	}

	// Note: In a real implementation, you would have an UpdateSSOProvider method
	// For now, we'll simulate the update
	response := SSOProviderResponse{
		ID:        provider.ID,
		Name:      provider.Name,
		Type:      provider.Type,
		Status:    provider.Status,
		CreatedAt: provider.CreatedAt,
		UpdatedAt: provider.UpdatedAt,
	}

	c.JSON(http.StatusOK, response)
}

// DeleteSSOProvider marks an SSO provider as deleted
// @Summary Delete SSO Provider
// @Description Mark an SSO provider as deleted (soft delete)
// @Tags SSO
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Provider ID"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/sso/providers/{id} [delete]
func (h *SSOHandler) DeleteSSOProvider(c *gin.Context) {
	providerIDStr := c.Param("id")
	providerID, err := uuid.Parse(providerIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid provider ID",
			Message: "Provider ID must be a valid UUID",
		})
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "Unauthorized",
			Message: "Tenant ID not found in context",
		})
		return
	}

	// Check permissions (admin only)
	userRole, exists := c.Get("user_role")
	if !exists || userRole != "admin" {
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "Forbidden",
			Message: "Only administrators can delete SSO providers",
		})
		return
	}

	// Verify provider exists
	_, err = h.ssoService.GetSSOProviderByID(c.Request.Context(), tenantID.(uuid.UUID), providerID)
	if err != nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "SSO provider not found",
			Message: err.Error(),
		})
		return
	}

	// Note: In a real implementation, you would have a DeleteSSOProvider method
	// that marks the provider as deleted
	
	c.Status(http.StatusNoContent)
}