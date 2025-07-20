package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
	"github.com/cloud-platform/collaborative-dev/shared/api"
	"github.com/cloud-platform/collaborative-dev/shared/logger"
)

type APITokenHandler struct {
	tokenService *services.APITokenService
	logger       logger.Logger
	respHandler  *api.ResponseHandler
}

func NewAPITokenHandler(tokenService *services.APITokenService, logger logger.Logger) *APITokenHandler {
	return &APITokenHandler{
		tokenService: tokenService,
		logger:       logger,
		respHandler:  api.NewResponseHandler(),
	}
}

// APITokenResponse represents the response for API token operations
type APITokenResponse struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	Description  string     `json:"description"`
	TokenPrefix  string     `json:"token_prefix"`
	Scopes       []string   `json:"scopes"`
	Permissions  []string   `json:"permissions"`
	Status       string     `json:"status"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	LastUsedIP   string     `json:"last_used_ip"`
	UseCount     int64      `json:"use_count"`
	RateLimitRPS int        `json:"rate_limit_rps"`
	ExpiresAt    *time.Time `json:"expires_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CreateAPITokenResponse includes the actual token (only returned once)
type CreateAPITokenResponse struct {
	APITokenResponse
	Token string `json:"token"` // Only returned during creation
}

// CreateAPIToken creates a new API token
// @Summary Create API Token
// @Description Create a new long-lived API access token
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param request body services.CreateAPITokenRequest true "API token creation request"
// @Success 201 {object} CreateAPITokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens [post]
func (h *APITokenHandler) CreateAPIToken(c *gin.Context) {
	var req services.CreateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	// Get user and tenant from context
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.BadRequest(c, "缺少租户信息", nil)
		return
	}

	// Set default rate limit if not provided
	if req.RateLimitRPS == 0 {
		req.RateLimitRPS = 100 // Default 100 RPS
	}

	// Create token
	token, tokenString, err := h.tokenService.CreateAPIToken(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		userID.(uuid.UUID),
		&req,
	)
	if err != nil {
		h.respHandler.InternalServerError(c, "创建API令牌失败")
		return
	}

	// Parse scopes and permissions for response
	var scopes []string
	json.Unmarshal(token.Scopes, &scopes)

	var permissions []string
	json.Unmarshal(token.Permissions, &permissions)

	response := CreateAPITokenResponse{
		APITokenResponse: APITokenResponse{
			ID:           token.ID,
			Name:         token.Name,
			Description:  token.Description,
			TokenPrefix:  token.TokenPrefix,
			Scopes:       scopes,
			Permissions:  permissions,
			Status:       token.Status,
			LastUsedAt:   token.LastUsedAt,
			LastUsedIP:   token.LastUsedIP,
			UseCount:     token.UseCount,
			RateLimitRPS: token.RateLimitRPS,
			ExpiresAt:    token.ExpiresAt,
			CreatedAt:    token.CreatedAt,
			UpdatedAt:    token.UpdatedAt,
		},
		Token: tokenString,
	}

	h.respHandler.Created(c, "API令牌创建成功", response)
}

// GetAPITokens retrieves all API tokens for the current user
// @Summary Get API Tokens
// @Description Retrieve all API tokens for the current user
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {array} APITokenResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens [get]
func (h *APITokenHandler) GetAPITokens(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息缺失")
		return
	}

	tokens, err := h.tokenService.GetAPITokens(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		userID.(uuid.UUID),
	)
	if err != nil {
		h.respHandler.InternalServerError(c, "获取API令牌列表失败")
		return
	}

	// Convert to response format
	var response []APITokenResponse
	for _, token := range tokens {
		var scopes []string
		json.Unmarshal(token.Scopes, &scopes)

		var permissions []string
		json.Unmarshal(token.Permissions, &permissions)

		response = append(response, APITokenResponse{
			ID:           token.ID,
			Name:         token.Name,
			Description:  token.Description,
			TokenPrefix:  token.TokenPrefix,
			Scopes:       scopes,
			Permissions:  permissions,
			Status:       token.Status,
			LastUsedAt:   token.LastUsedAt,
			LastUsedIP:   token.LastUsedIP,
			UseCount:     token.UseCount,
			RateLimitRPS: token.RateLimitRPS,
			ExpiresAt:    token.ExpiresAt,
			CreatedAt:    token.CreatedAt,
			UpdatedAt:    token.UpdatedAt,
		})
	}

	h.respHandler.OK(c, "获取API令牌列表成功", response)
}

// GetAPITokenByID retrieves a specific API token
// @Summary Get API Token
// @Description Retrieve a specific API token by ID
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Token ID"
// @Success 200 {object} APITokenResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id} [get]
func (h *APITokenHandler) GetAPITokenByID(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "令牌ID格式无效", nil)
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息缺失")
		return
	}

	token, err := h.tokenService.GetAPITokenByID(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tokenID,
	)
	if err != nil {
		h.respHandler.NotFound(c, "API令牌不存在")
		return
	}

	// Parse scopes and permissions for response
	var scopes []string
	json.Unmarshal(token.Scopes, &scopes)

	var permissions []string
	json.Unmarshal(token.Permissions, &permissions)

	response := APITokenResponse{
		ID:           token.ID,
		Name:         token.Name,
		Description:  token.Description,
		TokenPrefix:  token.TokenPrefix,
		Scopes:       scopes,
		Permissions:  permissions,
		Status:       token.Status,
		LastUsedAt:   token.LastUsedAt,
		LastUsedIP:   token.LastUsedIP,
		UseCount:     token.UseCount,
		RateLimitRPS: token.RateLimitRPS,
		ExpiresAt:    token.ExpiresAt,
		CreatedAt:    token.CreatedAt,
		UpdatedAt:    token.UpdatedAt,
	}

	h.respHandler.OK(c, "获取API令牌详情成功", response)
}

// UpdateAPIToken updates an existing API token
// @Summary Update API Token
// @Description Update an existing API token's metadata
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Token ID"
// @Param request body services.UpdateAPITokenRequest true "API token update request"
// @Success 200 {object} APITokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id} [put]
func (h *APITokenHandler) UpdateAPIToken(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "令牌ID格式无效", nil)
		return
	}

	var req services.UpdateAPITokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.respHandler.BadRequest(c, "请求参数无效", nil)
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息缺失")
		return
	}

	// Update token
	if err := h.tokenService.UpdateAPIToken(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tokenID,
		&req,
	); err != nil {
		h.respHandler.InternalServerError(c, "更新API令牌失败")
		return
	}

	// Get updated token
	token, err := h.tokenService.GetAPITokenByID(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tokenID,
	)
	if err != nil {
		h.respHandler.NotFound(c, "API令牌不存在")
		return
	}

	// Parse scopes and permissions for response
	var scopes []string
	json.Unmarshal(token.Scopes, &scopes)

	var permissions []string
	json.Unmarshal(token.Permissions, &permissions)

	response := APITokenResponse{
		ID:           token.ID,
		Name:         token.Name,
		Description:  token.Description,
		TokenPrefix:  token.TokenPrefix,
		Scopes:       scopes,
		Permissions:  permissions,
		Status:       token.Status,
		LastUsedAt:   token.LastUsedAt,
		LastUsedIP:   token.LastUsedIP,
		UseCount:     token.UseCount,
		RateLimitRPS: token.RateLimitRPS,
		ExpiresAt:    token.ExpiresAt,
		CreatedAt:    token.CreatedAt,
		UpdatedAt:    token.UpdatedAt,
	}

	h.respHandler.OK(c, "更新API令牌成功", response)
}

// RevokeAPIToken revokes an API token
// @Summary Revoke API Token
// @Description Revoke an API token to prevent further use
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Token ID"
// @Success 204
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id}/revoke [post]
func (h *APITokenHandler) RevokeAPIToken(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "令牌ID格式无效", nil)
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		h.respHandler.Unauthorized(c, "用户未认证")
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息缺失")
		return
	}

	if err := h.tokenService.RevokeAPIToken(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tokenID,
		userID.(uuid.UUID),
	); err != nil {
		h.respHandler.InternalServerError(c, "撤销API令牌失败")
		return
	}

	c.Status(http.StatusNoContent)
}

// GetTokenUsageStats retrieves usage statistics for a token
// @Summary Get Token Usage Stats
// @Description Get usage statistics for an API token
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Param id path string true "Token ID"
// @Param days query int false "Number of days to analyze (default: 30)"
// @Success 200 {object} services.TokenUsageStats
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/{id}/stats [get]
func (h *APITokenHandler) GetTokenUsageStats(c *gin.Context) {
	tokenIDStr := c.Param("id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		h.respHandler.BadRequest(c, "令牌ID格式无效", nil)
		return
	}

	tenantID, exists := c.Get("tenant_id")
	if !exists {
		h.respHandler.Unauthorized(c, "租户信息缺失")
		return
	}

	// Parse days parameter
	days := 30 // default
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 365 {
			days = d
		}
	}

	stats, err := h.tokenService.GetTokenUsageStats(
		c.Request.Context(),
		tenantID.(uuid.UUID),
		tokenID,
		days,
	)
	if err != nil {
		h.respHandler.InternalServerError(c, "获取令牌使用统计失败")
		return
	}

	h.respHandler.OK(c, "获取令牌使用统计成功", stats)
}

// GetAvailableScopes retrieves all available API scopes
// @Summary Get Available Scopes
// @Description Retrieve all available API scopes
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer token"
// @Success 200 {array} models.APIScope
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/scopes [get]
func (h *APITokenHandler) GetAvailableScopes(c *gin.Context) {
	scopes, err := h.tokenService.GetAvailableScopes(c.Request.Context())
	if err != nil {
		h.respHandler.InternalServerError(c, "获取可用权限范围失败")
		return
	}

	h.respHandler.OK(c, "获取可用权限范围成功", scopes)
}

// ValidateToken validates an API token (internal use)
// @Summary Validate API Token
// @Description Validate an API token for internal use
// @Tags API Tokens
// @Accept json
// @Produce json
// @Param token query string true "API Token"
// @Success 200 {object} models.APIToken
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/tokens/validate [get]
func (h *APITokenHandler) ValidateToken(c *gin.Context) {
	tokenString := c.Query("token")
	if tokenString == "" {
		h.respHandler.BadRequest(c, "令牌参数必填", nil)
		return
	}

	token, err := h.tokenService.ValidateAPIToken(c.Request.Context(), tokenString)
	if err != nil {
		h.respHandler.Unauthorized(c, "无效的令牌")
		return
	}

	// Update token usage
	clientIP := c.ClientIP()
	h.tokenService.UpdateTokenUsage(c.Request.Context(), token.ID, clientIP)

	h.respHandler.OK(c, "令牌验证成功", token)
}
