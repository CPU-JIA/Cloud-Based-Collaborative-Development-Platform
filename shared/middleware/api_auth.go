package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cloud-platform/collaborative-dev/cmd/iam-service/services"
)

// APIAuthMiddleware creates middleware for API token authentication
func APIAuthMiddleware(tokenService *services.APITokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Authorization header is required",
			})
			c.Abort()
			return
		}

		// Check for Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Invalid authorization header format",
			})
			c.Abort()
			return
		}

		scheme := parts[0]
		tokenString := parts[1]

		// Handle different token types
		switch strings.ToLower(scheme) {
		case "bearer":
			// Bearer tokens should be handled by JWTAuth middleware
			// This middleware is only for API tokens, so we should not allow Bearer here
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized", 
				"message": "This endpoint requires API token authentication. Use 'Token <your-api-token>' format",
			})
			c.Abort()
			return
		case "token":
			// This is an API token
			if err := validateAPIToken(c, tokenService, tokenString); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error":   "Unauthorized",
					"message": err.Error(),
				})
				c.Abort()
				return
			}
		default:
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "Unsupported authorization scheme",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APITokenOnlyMiddleware creates middleware that only accepts API tokens
func APITokenOnlyMiddleware(tokenService *services.APITokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API token is required",
			})
			c.Abort()
			return
		}

		// Check for Token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "token" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": "API token required. Use 'Token <your-api-token>' format",
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		if err := validateAPIToken(c, tokenService, tokenString); err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "Unauthorized",
				"message": err.Error(),
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RequireAPIScope creates middleware that checks for specific API scopes
func RequireAPIScope(tokenService *services.APITokenService, resource, action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token ID from context (set by validateAPIToken)
		tokenID, exists := c.Get("api_token_id")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "API token context not found",
			})
			c.Abort()
			return
		}

		// Check if token has the required permission
		hasPermission, err := tokenService.CheckTokenPermission(
			c.Request.Context(),
			tokenID.(uuid.UUID),
			resource,
			action,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Failed to check token permissions",
			})
			c.Abort()
			return
		}

		if !hasPermission {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "Forbidden",
				"message": "Insufficient permissions",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// APITokenRateLimitMiddleware creates middleware for API rate limiting
func APITokenRateLimitMiddleware(tokenService *services.APITokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply rate limiting to API tokens
		tokenID, exists := c.Get("api_token_id")
		if !exists {
			// Not an API token request, skip rate limiting
			c.Next()
			return
		}

		// Check rate limit
		allowed, err := tokenService.CheckRateLimit(c.Request.Context(), tokenID.(uuid.UUID))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "Internal Server Error",
				"message": "Failed to check rate limit",
			})
			c.Abort()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "Rate Limit Exceeded",
				"message": "API rate limit exceeded",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// validateAPIToken validates an API token and sets context
func validateAPIToken(c *gin.Context, tokenService *services.APITokenService, tokenString string) error {
	// Validate token
	token, err := tokenService.ValidateAPIToken(c.Request.Context(), tokenString)
	if err != nil {
		return err
	}

	// Update token usage
	clientIP := c.ClientIP()
	tokenService.UpdateTokenUsage(c.Request.Context(), token.ID, clientIP)

	// Set token information in context
	c.Set("api_token_id", token.ID)
	c.Set("user_id", token.UserID)
	c.Set("tenant_id", token.TenantID)
	c.Set("auth_type", "api_token")

	// Set user information if available
	if token.User != nil {
		c.Set("username", token.User.Username)
		c.Set("user_email", token.User.Email)
	}

	return nil
}
