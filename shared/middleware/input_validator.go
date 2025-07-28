package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

var (
	// 预编译的正则表达式
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)
	uuidRegex     = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	pathRegex     = regexp.MustCompile(`^[a-zA-Z0-9._/\-]+$`)
	
	// SQL注入关键词
	sqlKeywords = []string{"select", "insert", "update", "delete", "drop", "union", "exec", "script"}
)

// InputValidator 输入验证器
type InputValidator struct {
	validator *validator.Validate
}

// NewInputValidator 创建输入验证器
func NewInputValidator() *InputValidator {
	v := validator.New()
	
	// 注册自定义验证规则
	v.RegisterValidation("safestring", validateSafeString)
	v.RegisterValidation("safepath", validateSafePath)
	v.RegisterValidation("strongpassword", validateStrongPassword)
	
	return &InputValidator{
		validator: v,
	}
}

// ValidateInput 输入验证中间件
func ValidateInput() gin.HandlerFunc {
	validator := NewInputValidator()
	
	return func(c *gin.Context) {
		// 验证查询参数
		if err := validator.ValidateQueryParams(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid query parameters",
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		
		// 验证路径参数
		if err := validator.ValidatePathParams(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid path parameters", 
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		
		// 验证请求头
		if err := validator.ValidateHeaders(c); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "Invalid headers",
				"message": err.Error(),
			})
			c.Abort()
			return
		}
		
		c.Next()
	}
}

// ValidateQueryParams 验证查询参数
func (v *InputValidator) ValidateQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		// 验证参数名
		if !isValidParamName(key) {
			return fmt.Errorf("invalid parameter name: %s", key)
		}
		
		// 验证参数值
		for _, value := range values {
			if containsSQLInjection(value) {
				return fmt.Errorf("potential SQL injection detected in parameter: %s", key)
			}
			
			// 检查特殊参数
			switch key {
			case "email":
				if !isValidEmail(value) {
					return fmt.Errorf("invalid email format")
				}
			case "page", "limit", "offset":
				if !isValidNumber(value) {
					return fmt.Errorf("invalid number format for %s", key)
				}
			}
		}
	}
	
	return nil
}

// ValidatePathParams 验证路径参数
func (v *InputValidator) ValidatePathParams(c *gin.Context) error {
	// 验证常见的路径参数
	if id := c.Param("id"); id != "" {
		if !isValidUUID(id) && !isValidNumber(id) {
			return fmt.Errorf("invalid ID format")
		}
	}
	
	if username := c.Param("username"); username != "" {
		if !isValidUsername(username) {
			return fmt.Errorf("invalid username format")
		}
	}
	
	if path := c.Param("path"); path != "" {
		if !isValidPath(path) {
			return fmt.Errorf("invalid path format")
		}
	}
	
	return nil
}

// ValidateHeaders 验证请求头
func (v *InputValidator) ValidateHeaders(c *gin.Context) error {
	// 验证Content-Type
	contentType := c.GetHeader("Content-Type")
	if c.Request.Method == "POST" || c.Request.Method == "PUT" {
		if contentType == "" {
			return fmt.Errorf("Content-Type header is required")
		}
		
		// 限制允许的Content-Type
		allowedTypes := []string{
			"application/json",
			"application/x-www-form-urlencoded",
			"multipart/form-data",
		}
		
		validType := false
		for _, allowed := range allowedTypes {
			if strings.HasPrefix(contentType, allowed) {
				validType = true
				break
			}
		}
		
		if !validType {
			return fmt.Errorf("unsupported Content-Type: %s", contentType)
		}
	}
	
	// 验证自定义头
	for key, values := range c.Request.Header {
		if strings.HasPrefix(key, "X-") {
			for _, value := range values {
				if containsSQLInjection(value) {
					return fmt.Errorf("potential injection in header: %s", key)
				}
			}
		}
	}
	
	return nil
}

// ValidateStruct 验证结构体
func (v *InputValidator) ValidateStruct(s interface{}) error {
	return v.validator.Struct(s)
}

// 辅助函数

func isValidParamName(name string) bool {
	// 只允许字母、数字、下划线和连字符
	return regexp.MustCompile(`^[a-zA-Z0-9_-]+$`).MatchString(name)
}

func isValidEmail(email string) bool {
	return emailRegex.MatchString(email)
}

func isValidUsername(username string) bool {
	return usernameRegex.MatchString(username)
}

func isValidUUID(id string) bool {
	return uuidRegex.MatchString(id)
}

func isValidNumber(str string) bool {
	return regexp.MustCompile(`^\d+$`).MatchString(str)
}

func isValidPath(path string) bool {
	// 防止路径遍历
	if strings.Contains(path, "..") {
		return false
	}
	return pathRegex.MatchString(path)
}

func containsSQLInjection(input string) bool {
	lowered := strings.ToLower(input)
	
	// 检查SQL关键词
	for _, keyword := range sqlKeywords {
		if strings.Contains(lowered, keyword) {
			// 检查是否在单词边界
			pattern := fmt.Sprintf(`\b%s\b`, keyword)
			if regexp.MustCompile(pattern).MatchString(lowered) {
				return true
			}
		}
	}
	
	// 检查特殊字符组合
	dangerousPatterns := []string{
		`--`,      // SQL注释
		`/*`,      // SQL注释
		`*/`,      // SQL注释
		`;`,       // 语句分隔符
		`'='`,     // 常见注入模式
		`" or "`,  // 常见注入模式
		`' or '`,  // 常见注入模式
	}
	
	for _, pattern := range dangerousPatterns {
		if strings.Contains(input, pattern) {
			return true
		}
	}
	
	return false
}

// 自定义验证规则

func validateSafeString(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return !containsSQLInjection(value)
}

func validateSafePath(fl validator.FieldLevel) bool {
	value := fl.Field().String()
	return isValidPath(value)
}

func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()
	
	// 至少8个字符
	if len(password) < 8 {
		return false
	}
	
	// 必须包含大写字母、小写字母、数字和特殊字符
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)
	
	return hasUpper && hasLower && hasNumber && hasSpecial
}