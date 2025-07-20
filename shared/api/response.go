package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StandardResponse 平台统一API响应格式
// 遵循Claude Code开发规范，确保所有微服务API响应一致性
type StandardResponse struct {
	Success bool        `json:"success"`
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   *ErrorInfo  `json:"error,omitempty"`
	TraceID string      `json:"trace_id,omitempty"`
}

// ErrorInfo 错误详情结构
type ErrorInfo struct {
	Type   string            `json:"type"`
	Detail string            `json:"detail"`
	Fields map[string]string `json:"fields,omitempty"` // 用于字段验证错误
}

// ResponseHandler 响应处理器
type ResponseHandler struct {
	// 可以扩展为包含logger等
}

// NewResponseHandler 创建响应处理器
func NewResponseHandler() *ResponseHandler {
	return &ResponseHandler{}
}

// Success 发送成功响应
func (h *ResponseHandler) Success(c *gin.Context, statusCode int, message string, data interface{}) {
	traceID := h.getTraceID(c)
	response := StandardResponse{
		Success: true,
		Code:    statusCode,
		Message: message,
		Data:    data,
		TraceID: traceID,
	}

	c.JSON(statusCode, response)
}

// Error 发送错误响应
func (h *ResponseHandler) Error(c *gin.Context, statusCode int, errorType, message string, fields map[string]string) {
	traceID := h.getTraceID(c)
	response := StandardResponse{
		Success: false,
		Code:    statusCode,
		Message: "请求失败",
		TraceID: traceID,
		Error: &ErrorInfo{
			Type:   errorType,
			Detail: message,
			Fields: fields,
		},
	}

	c.JSON(statusCode, response)
}

// getTraceID 获取追踪ID
func (h *ResponseHandler) getTraceID(c *gin.Context) string {
	if traceID, exists := c.Get("request_id"); exists {
		if id, ok := traceID.(string); ok {
			return id
		}
	}
	return c.GetHeader("X-Request-ID")
}

// 便捷方法

// OK 200成功响应
func (h *ResponseHandler) OK(c *gin.Context, message string, data interface{}) {
	h.Success(c, http.StatusOK, message, data)
}

// Created 201创建成功响应
func (h *ResponseHandler) Created(c *gin.Context, message string, data interface{}) {
	h.Success(c, http.StatusCreated, message, data)
}

// NoContent 204无内容响应
func (h *ResponseHandler) NoContent(c *gin.Context) {
	c.Status(http.StatusNoContent)
}

// BadRequest 400错误请求
func (h *ResponseHandler) BadRequest(c *gin.Context, message string, fields map[string]string) {
	h.Error(c, http.StatusBadRequest, "bad_request", message, fields)
}

// Unauthorized 401未授权
func (h *ResponseHandler) Unauthorized(c *gin.Context, message string) {
	h.Error(c, http.StatusUnauthorized, "unauthorized", message, nil)
}

// Forbidden 403禁止访问
func (h *ResponseHandler) Forbidden(c *gin.Context, message string) {
	h.Error(c, http.StatusForbidden, "forbidden", message, nil)
}

// NotFound 404未找到
func (h *ResponseHandler) NotFound(c *gin.Context, message string) {
	h.Error(c, http.StatusNotFound, "not_found", message, nil)
}

// Conflict 409冲突
func (h *ResponseHandler) Conflict(c *gin.Context, message string) {
	h.Error(c, http.StatusConflict, "conflict", message, nil)
}

// InternalServerError 500内部服务器错误
func (h *ResponseHandler) InternalServerError(c *gin.Context, message string) {
	h.Error(c, http.StatusInternalServerError, "internal_error", message, nil)
}

// ValidationError 参数验证错误
func (h *ResponseHandler) ValidationError(c *gin.Context, message string, fields map[string]string) {
	h.Error(c, http.StatusBadRequest, "validation_error", message, fields)
}
