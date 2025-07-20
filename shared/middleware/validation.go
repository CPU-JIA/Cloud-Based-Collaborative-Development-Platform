package middleware

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidationError 验证错误结构
type ValidationError struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ErrorResponse 统一错误响应格式
type ErrorResponse struct {
	Success     bool              `json:"success"`
	Error       string            `json:"error"`
	Code        string            `json:"code"`
	Message     string            `json:"message,omitempty"`
	Details     []ValidationError `json:"details,omitempty"`
	RequestID   string            `json:"request_id,omitempty"`
	Timestamp   string            `json:"timestamp"`
	Path        string            `json:"path"`
}

var validate *validator.Validate

func init() {
	validate = validator.New()
	
	// 注册自定义验证标签
	validate.RegisterValidation("password", validatePassword)
	validate.RegisterValidation("username", validateUsername)
	validate.RegisterValidation("phone", validatePhone)
}

// ValidateRequest 请求验证中间件
func ValidateRequest() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 设置自定义错误处理器
		c.Set("validation_error_handler", handleValidationError)
		c.Next()
	}
}

// handleValidationError 处理验证错误
func handleValidationError(c *gin.Context, err error) {
	var validationErrors []ValidationError
	
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			validationErrors = append(validationErrors, ValidationError{
				Field:   getFieldName(fe),
				Tag:     fe.Tag(),
				Value:   fmt.Sprintf("%v", fe.Value()),
				Message: getValidationMessage(fe),
			})
		}
	}
	
	requestID := c.GetString("request_id")
	
	c.JSON(http.StatusBadRequest, ErrorResponse{
		Success:   false,
		Error:     "请求参数验证失败",
		Code:      "VALIDATION_ERROR",
		Details:   validationErrors,
		RequestID: requestID,
		Timestamp: getCurrentTimestamp(),
		Path:      c.Request.URL.Path,
	})
}

// getFieldName 获取字段名称（优先使用json标签）
func getFieldName(fe validator.FieldError) string {
	field := fe.Field()
	
	// 如果有structTag，尝试获取json标签
	if fe.StructNamespace() != "" {
		// 这里可以通过反射获取json标签，简化处理直接返回字段名
		return strings.ToLower(field)
	}
	
	return strings.ToLower(field)
}

// getValidationMessage 获取验证错误消息
func getValidationMessage(fe validator.FieldError) string {
	field := getFieldName(fe)
	
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s 是必填字段", field)
	case "email":
		return fmt.Sprintf("%s 必须是有效的邮箱地址", field)
	case "min":
		return fmt.Sprintf("%s 长度不能少于 %s 个字符", field, fe.Param())
	case "max":
		return fmt.Sprintf("%s 长度不能超过 %s 个字符", field, fe.Param())
	case "len":
		return fmt.Sprintf("%s 长度必须是 %s 个字符", field, fe.Param())
	case "gte":
		return fmt.Sprintf("%s 必须大于或等于 %s", field, fe.Param())
	case "lte":
		return fmt.Sprintf("%s 必须小于或等于 %s", field, fe.Param())
	case "gt":
		return fmt.Sprintf("%s 必须大于 %s", field, fe.Param())
	case "lt":
		return fmt.Sprintf("%s 必须小于 %s", field, fe.Param())
	case "oneof":
		return fmt.Sprintf("%s 必须是以下值之一: %s", field, fe.Param())
	case "uuid":
		return fmt.Sprintf("%s 必须是有效的UUID格式", field)
	case "password":
		return fmt.Sprintf("%s 必须包含至少8个字符，包括大小写字母和数字", field)
	case "username":
		return fmt.Sprintf("%s 只能包含字母、数字、下划线和连字符", field)
	case "phone":
		return fmt.Sprintf("%s 必须是有效的手机号码", field)
	default:
		return fmt.Sprintf("%s 验证失败", field)
	}
}

// 自定义验证函数

// validatePassword 密码验证
func validatePassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	
	if len(password) < 8 {
		return false
	}
	
	var hasUpper, hasLower, hasDigit bool
	for _, char := range password {
		switch {
		case 'A' <= char && char <= 'Z':
			hasUpper = true
		case 'a' <= char && char <= 'z':
			hasLower = true
		case '0' <= char && char <= '9':
			hasDigit = true
		}
	}
	
	return hasUpper && hasLower && hasDigit
}

// validateUsername 用户名验证
func validateUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	
	if len(username) < 3 || len(username) > 50 {
		return false
	}
	
	for _, char := range username {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9') || 
			 char == '_' || char == '-') {
			return false
		}
	}
	
	return true
}

// validatePhone 手机号验证
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	
	// 简单的手机号验证（可根据需要扩展）
	if len(phone) < 10 || len(phone) > 15 {
		return false
	}
	
	// 检查是否都是数字（可能包含+和-）
	for i, char := range phone {
		if i == 0 && char == '+' {
			continue
		}
		if char == '-' || char == ' ' {
			continue
		}
		if char < '0' || char > '9' {
			return false
		}
	}
	
	return true
}

// ValidateStruct 结构体验证辅助函数
func ValidateStruct(s interface{}) error {
	return validate.Struct(s)
}

// Paginate 分页参数验证和处理
func Paginate(c *gin.Context) (page, limit, offset int, err error) {
	// 获取页码
	pageStr := c.DefaultQuery("page", "1")
	page, err = strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		return 0, 0, 0, fmt.Errorf("页码必须是大于0的整数")
	}
	
	// 获取每页数量
	limitStr := c.DefaultQuery("limit", "20")
	limit, err = strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		return 0, 0, 0, fmt.Errorf("每页数量必须是1-100之间的整数")
	}
	
	// 计算偏移量
	offset = (page - 1) * limit
	
	return page, limit, offset, nil
}

// getCurrentTimestamp 获取当前时间戳
func getCurrentTimestamp() string {
	return time.Now().UTC().Format(time.RFC3339)
}

// ValidateJSON 验证JSON请求体
func ValidateJSON(target interface{}) gin.HandlerFunc {
	return func(c *gin.Context) {
		if err := c.ShouldBindJSON(target); err != nil {
			requestID := c.GetString("request_id")
			
			// 检查是否为验证错误
			if ve, ok := err.(validator.ValidationErrors); ok {
				var validationErrors []ValidationError
				for _, fe := range ve {
					validationErrors = append(validationErrors, ValidationError{
						Field:   getFieldName(fe),
						Tag:     fe.Tag(),
						Value:   fmt.Sprintf("%v", fe.Value()),
						Message: getValidationMessage(fe),
					})
				}
				
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success:   false,
					Error:     "请求参数验证失败",
					Code:      "VALIDATION_ERROR",
					Details:   validationErrors,
					RequestID: requestID,
					Timestamp: getCurrentTimestamp(),
					Path:      c.Request.URL.Path,
				})
			} else {
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success:   false,
					Error:     "请求格式错误",
					Code:      "INVALID_JSON",
					Message:   err.Error(),
					RequestID: requestID,
					Timestamp: getCurrentTimestamp(),
					Path:      c.Request.URL.Path,
				})
			}
			
			c.Abort()
			return
		}
		
		// 将验证通过的数据设置到上下文
		c.Set("validated_data", target)
		c.Next()
	}
}

// ValidateQueryParams 验证查询参数
func ValidateQueryParams(params map[string]string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var errors []ValidationError
		
		for param, validation := range params {
			value := c.Query(param)
			if value == "" {
				continue // 可选参数跳过
			}
			
			switch validation {
			case "uuid":
				if !isValidUUID(value) {
					errors = append(errors, ValidationError{
						Field:   param,
						Tag:     "uuid",
						Value:   value,
						Message: fmt.Sprintf("%s 必须是有效的UUID格式", param),
					})
				}
			case "bool":
				if value != "true" && value != "false" {
					errors = append(errors, ValidationError{
						Field:   param,
						Tag:     "bool",
						Value:   value,
						Message: fmt.Sprintf("%s 必须是true或false", param),
					})
				}
			case "int":
				if _, err := strconv.Atoi(value); err != nil {
					errors = append(errors, ValidationError{
						Field:   param,
						Tag:     "int",
						Value:   value,
						Message: fmt.Sprintf("%s 必须是整数", param),
					})
				}
			}
		}
		
		if len(errors) > 0 {
			requestID := c.GetString("request_id")
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Success:   false,
				Error:     "查询参数验证失败",
				Code:      "QUERY_VALIDATION_ERROR",
				Details:   errors,
				RequestID: requestID,
				Timestamp: getCurrentTimestamp(),
				Path:      c.Request.URL.Path,
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// isValidUUID 检查是否为有效的UUID
func isValidUUID(str string) bool {
	if len(str) != 36 {
		return false
	}
	
	for i, char := range str {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if char != '-' {
				return false
			}
		} else {
			if !((char >= '0' && char <= '9') ||
				 (char >= 'a' && char <= 'f') ||
				 (char >= 'A' && char <= 'F')) {
				return false
			}
		}
	}
	
	return true
}

// ErrorHandler 统一错误处理中间件
func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		
		// 检查是否有错误
		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			requestID := c.GetString("request_id")
			
			// 根据错误类型返回相应的状态码和消息
			switch err.Type {
			case gin.ErrorTypeBind:
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success:   false,
					Error:     "请求参数绑定失败",
					Code:      "BIND_ERROR",
					Message:   err.Error(),
					RequestID: requestID,
					Timestamp: getCurrentTimestamp(),
					Path:      c.Request.URL.Path,
				})
			case gin.ErrorTypePublic:
				c.JSON(http.StatusBadRequest, ErrorResponse{
					Success:   false,
					Error:     "请求处理失败",
					Code:      "REQUEST_ERROR",
					Message:   err.Error(),
					RequestID: requestID,
					Timestamp: getCurrentTimestamp(),
					Path:      c.Request.URL.Path,
				})
			default:
				c.JSON(http.StatusInternalServerError, ErrorResponse{
					Success:   false,
					Error:     "服务器内部错误",
					Code:      "INTERNAL_ERROR",
					RequestID: requestID,
					Timestamp: getCurrentTimestamp(),
					Path:      c.Request.URL.Path,
				})
			}
		}
	}
}