package handlers

import "github.com/cloud-platform/collaborative-dev/shared/api"

// StandardResponse 使用统一响应格式
type StandardResponse = api.StandardResponse

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}
