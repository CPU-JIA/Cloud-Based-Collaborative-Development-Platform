package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
}

// ErrorInfo 错误信息
type ErrorInfo struct {
	Type    string      `json:"type,omitempty"`
	Details interface{} `json:"details,omitempty"`
}

// Success 成功响应
func Success(c *gin.Context, statusCode int, message string, data interface{}) {
	c.JSON(statusCode, Response{
		Code:    statusCode,
		Message: message,
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, statusCode int, message string, err interface{}) {
	response := Response{
		Code:    statusCode,
		Message: message,
	}

	if err != nil {
		response.Error = &ErrorInfo{
			Type:    http.StatusText(statusCode),
			Details: err,
		}
	}

	c.JSON(statusCode, response)
}

// InternalError 内部错误响应
func InternalError(c *gin.Context, logger *zap.Logger, message string, err error) {
	if logger != nil {
		logger.Error("内部错误", zap.Error(err))
	}

	Error(c, http.StatusInternalServerError, message, nil)
}

// BadRequest 请求错误响应
func BadRequest(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusBadRequest, message, details)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, message string) {
	Error(c, http.StatusUnauthorized, message, nil)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, message string) {
	Error(c, http.StatusForbidden, message, nil)
}

// NotFound 未找到响应
func NotFound(c *gin.Context, message string) {
	Error(c, http.StatusNotFound, message, nil)
}

// Conflict 冲突响应
func Conflict(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusConflict, message, details)
}

// ValidationError 验证错误响应
func ValidationError(c *gin.Context, message string, details interface{}) {
	Error(c, http.StatusUnprocessableEntity, message, details)
}
