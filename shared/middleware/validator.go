package middleware

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// ValidatorConfig 验证器配置
type ValidatorConfig struct {
	// EnableSQLInjectionProtection 启用SQL注入防护
	EnableSQLInjectionProtection bool

	// EnableXSSProtection 启用XSS防护
	EnableXSSProtection bool

	// EnablePathTraversalProtection 启用路径遍历防护
	EnablePathTraversalProtection bool

	// EnableCommandInjectionProtection 启用命令注入防护
	EnableCommandInjectionProtection bool

	// EnableLDAPInjectionProtection 启用LDAP注入防护
	EnableLDAPInjectionProtection bool

	// EnableDeepJSONValidation 启用JSON深度验证
	EnableDeepJSONValidation bool

	// MaxRequestSize 最大请求大小（字节）
	MaxRequestSize int64

	// MaxJSONDepth JSON最大嵌套深度
	MaxJSONDepth int

	// AllowedContentTypes 允许的内容类型
	AllowedContentTypes []string

	// CustomValidators 自定义验证器
	CustomValidators map[string]validator.Func

	// ErrorHandler 错误处理函数
	ErrorHandler func(c *gin.Context, err error)

	// Logger 日志记录器
	Logger logger.Logger

	// SanitizeInput 是否自动清理输入
	SanitizeInput bool

	// BlockOnSuspiciousInput 检测到可疑输入时是否阻止请求
	BlockOnSuspiciousInput bool
}

// DefaultValidatorConfig 默认验证器配置
var DefaultValidatorConfig = ValidatorConfig{
	EnableSQLInjectionProtection:     true,
	EnableXSSProtection:              true,
	EnablePathTraversalProtection:    true,
	EnableCommandInjectionProtection: true,
	EnableLDAPInjectionProtection:    true,
	EnableDeepJSONValidation:         true,
	MaxRequestSize:                   10 * 1024 * 1024, // 10MB
	MaxJSONDepth:                     10,
	AllowedContentTypes: []string{
		"application/json",
		"application/x-www-form-urlencoded",
		"multipart/form-data",
		"text/plain",
	},
	SanitizeInput:          false, // 默认不自动清理，避免破坏数据
	BlockOnSuspiciousInput: true,  // 默认阻止可疑输入
}

// InputValidatorMiddleware 输入验证中间件
type InputValidatorMiddleware struct {
	config    *ValidatorConfig
	validator *validator.Validate
}

// NewInputValidatorMiddleware 创建输入验证中间件
func NewInputValidatorMiddleware(config *ValidatorConfig) *InputValidatorMiddleware {
	if config == nil {
		cfg := DefaultValidatorConfig
		config = &cfg
	}

	if config.ErrorHandler == nil {
		config.ErrorHandler = defaultValidationErrorHandler
	}

	v := validator.New()

	// 注册自定义验证器
	if config.CustomValidators != nil {
		for name, fn := range config.CustomValidators {
			v.RegisterValidation(name, fn)
		}
	}

	// 注册内置安全验证器
	registerSecurityValidators(v)

	return &InputValidatorMiddleware{
		config:    config,
		validator: v,
	}
}

// Handler 返回Gin中间件处理函数
func (m *InputValidatorMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 验证请求大小
		if c.Request.ContentLength > m.config.MaxRequestSize {
			m.config.ErrorHandler(c, fmt.Errorf("请求大小超过限制：%d bytes", c.Request.ContentLength))
			c.Abort()
			return
		}

		// 验证内容类型（仅在配置了允许的内容类型列表时才检查）
		if len(m.config.AllowedContentTypes) > 0 && !m.isAllowedContentType(c.ContentType()) && c.Request.ContentLength > 0 {
			m.config.ErrorHandler(c, fmt.Errorf("不支持的内容类型：%s", c.ContentType()))
			c.Abort()
			return
		}

		// 验证路径参数
		if err := m.validatePathParams(c); err != nil {
			m.config.ErrorHandler(c, err)
			c.Abort()
			return
		}

		// 验证查询参数
		if err := m.validateQueryParams(c); err != nil {
			m.config.ErrorHandler(c, err)
			c.Abort()
			return
		}

		// 验证请求体
		if c.Request.ContentLength > 0 {
			if err := m.validateRequestBody(c); err != nil {
				m.config.ErrorHandler(c, err)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// validatePathParams 验证路径参数
func (m *InputValidatorMiddleware) validatePathParams(c *gin.Context) error {
	for _, param := range c.Params {
		if m.config.EnableSQLInjectionProtection {
			if isSQLInjection(param.Value) {
				return fmt.Errorf("检测到SQL注入攻击：路径参数 '%s'", param.Key)
			}
		}

		if m.config.EnablePathTraversalProtection {
			if isPathTraversal(param.Value) {
				return fmt.Errorf("检测到路径遍历攻击：路径参数 '%s'", param.Key)
			}
		}

		if m.config.EnableXSSProtection {
			if isXSSAttack(param.Value) {
				return fmt.Errorf("检测到XSS攻击：路径参数 '%s'", param.Key)
			}
		}

		if m.config.EnableCommandInjectionProtection {
			if isCommandInjection(param.Value) {
				return fmt.Errorf("检测到命令注入攻击：路径参数 '%s'", param.Key)
			}
		}

		if m.config.EnableLDAPInjectionProtection {
			if isLDAPInjection(param.Value) {
				return fmt.Errorf("检测到LDAP注入攻击：路径参数 '%s'", param.Key)
			}
		}
	}
	return nil
}

// validateQueryParams 验证查询参数
func (m *InputValidatorMiddleware) validateQueryParams(c *gin.Context) error {
	for key, values := range c.Request.URL.Query() {
		for _, value := range values {
			if m.config.EnableSQLInjectionProtection {
				if isSQLInjection(value) {
					return fmt.Errorf("检测到SQL注入攻击：查询参数 '%s'", key)
				}
			}

			if m.config.EnableXSSProtection {
				if isXSSAttack(value) {
					return fmt.Errorf("检测到XSS攻击：查询参数 '%s'", key)
				}
			}

			if m.config.EnableCommandInjectionProtection {
				if isCommandInjection(value) {
					return fmt.Errorf("检测到命令注入攻击：查询参数 '%s'", key)
				}
			}

			if m.config.EnableLDAPInjectionProtection {
				if isLDAPInjection(value) {
					return fmt.Errorf("检测到LDAP注入攻击：查询参数 '%s'", key)
				}
			}
		}
	}
	return nil
}

// validateRequestBody 验证请求体
func (m *InputValidatorMiddleware) validateRequestBody(c *gin.Context) error {
	// 对于JSON请求，Gin会自动处理绑定和验证
	// 这里主要处理其他类型的请求

	if strings.Contains(c.ContentType(), "application/x-www-form-urlencoded") ||
		strings.Contains(c.ContentType(), "multipart/form-data") {

		// 解析表单
		if err := c.Request.ParseForm(); err != nil {
			return fmt.Errorf("解析表单失败：%w", err)
		}

		// 验证表单字段
		for key, values := range c.Request.Form {
			for _, value := range values {
				if m.config.EnableSQLInjectionProtection {
					if isSQLInjection(value) {
						return fmt.Errorf("检测到SQL注入攻击：表单字段 '%s'", key)
					}
				}

				if m.config.EnableXSSProtection {
					if isXSSAttack(value) {
						return fmt.Errorf("检测到XSS攻击：表单字段 '%s'", key)
					}
				}

				if m.config.EnableCommandInjectionProtection {
					if isCommandInjection(value) {
						return fmt.Errorf("检测到命令注入攻击：表单字段 '%s'", key)
					}
				}

				if m.config.EnableLDAPInjectionProtection {
					if isLDAPInjection(value) {
						return fmt.Errorf("检测到LDAP注入攻击：表单字段 '%s'", key)
					}
				}
			}
		}
	}

	return nil
}

// isAllowedContentType 检查是否为允许的内容类型
func (m *InputValidatorMiddleware) isAllowedContentType(contentType string) bool {
	if len(m.config.AllowedContentTypes) == 0 {
		return true
	}

	contentType = strings.ToLower(strings.TrimSpace(contentType))
	for _, allowed := range m.config.AllowedContentTypes {
		if strings.Contains(contentType, strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}

// SQL注入检测模式 - 增强版
var sqlInjectionPatterns = []*regexp.Regexp{
	// 基本SQL关键字
	regexp.MustCompile(`(?i)\b(union|select|insert|update|delete|drop|create|alter|truncate)\b`),
	// SQL函数和系统表
	regexp.MustCompile(`(?i)\b(exec|execute|xp_|sp_|@@|char|nchar|varchar|nvarchar|cast|convert|cursor|declare|fetch|sys|sysobjects|syscolumns|information_schema)\b`),
	// SQL操作符和注释
	regexp.MustCompile(`(?i)(;--|#|\/\*|\*\/|\|\||&&)`),
	// 条件注入模式
	regexp.MustCompile(`(?i)(\b(and|or)\b\s*(\d+\s*=\s*\d+|'[^']*'\s*=\s*'[^']*'))`),
	// 时间盲注
	regexp.MustCompile(`(?i)\b(sleep|waitfor|delay|benchmark|pg_sleep)\b`),
	// 堆叠查询
	regexp.MustCompile(`(?i);\s*(select|insert|update|delete|drop|create)`),
	// 联合查询注入
	regexp.MustCompile(`(?i)\bunion\s+(all\s+)?select\b`),
	// 布尔盲注
	regexp.MustCompile(`(?i)\b(and|or)\b\s*\d+\s*(>|<|=|!=)\s*\d+`),
	// HEX编码的SQL注入
	regexp.MustCompile(`(?i)(0x[0-9a-f]+)`),
}

// isSQLInjection 检测SQL注入
func isSQLInjection(input string) bool {
	// 空输入不检测
	if input == "" {
		return false
	}

	// 检查常见的SQL注入模式
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// 检查特殊字符组合
	dangerousCombos := []string{
		"' or '",
		"' OR '",
		"1=1",
		"1' or '1'='1",
		"admin'--",
		"') or ('1'='1",
	}

	lowerInput := strings.ToLower(input)
	for _, combo := range dangerousCombos {
		if strings.Contains(lowerInput, strings.ToLower(combo)) {
			return true
		}
	}

	return false
}

// XSS攻击检测模式 - 增强版
var xssPatterns = []*regexp.Regexp{
	// 脚本标签
	regexp.MustCompile(`(?i)<\s*script[^>]*>.*?</\s*script\s*>`),
	// iframe和框架
	regexp.MustCompile(`(?i)<\s*(iframe|frame|frameset)[^>]*>.*?</\s*(iframe|frame|frameset)\s*>`),
	// 对象和嵌入
	regexp.MustCompile(`(?i)<\s*(object|embed|applet)[^>]*>`),
	// 链接和导入
	regexp.MustCompile(`(?i)<\s*(link|import|meta)[^>]*>`),
	// 事件处理器
	regexp.MustCompile(`(?i)\bon\w+\s*=\s*["']?[^"']*["']?`),
	// JavaScript协议
	regexp.MustCompile(`(?i)(javascript|vbscript|livescript|mocha)\s*:`),
	// data URI scheme
	regexp.MustCompile(`(?i)data\s*:\s*text/html`),
	// SVG攻击向量
	regexp.MustCompile(`(?i)<\s*svg[^>]*>.*?</\s*svg\s*>`),
	// 样式注入
	regexp.MustCompile(`(?i)<\s*style[^>]*>.*?</\s*style\s*>`),
	// Expression和行为
	regexp.MustCompile(`(?i)(expression\s*\(|behavior\s*:|moz-binding)`),
	// Base64编码的脚本
	regexp.MustCompile(`(?i)base64\s*,\s*[a-zA-Z0-9+/=]+`),
	// HTML5新增的危险属性
	regexp.MustCompile(`(?i)(formaction|onfocus|onblur|onload|onerror|onclick|onmouseover)`),
}

// isXSSAttack 检测XSS攻击
func isXSSAttack(input string) bool {
	// 空输入不检测
	if input == "" {
		return false
	}

	// 检查XSS模式
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	// 检查HTML实体编码的攻击
	htmlEntities := []string{
		"&lt;script",
		"&#60;script",
		"&#x3C;script",
		"\\u003cscript",
		"\\x3cscript",
	}

	lowerInput := strings.ToLower(input)
	for _, entity := range htmlEntities {
		if strings.Contains(lowerInput, entity) {
			return true
		}
	}

	return false
}

// 路径遍历检测模式 - 增强版
var pathTraversalPatterns = []*regexp.Regexp{
	// 基本路径遍历
	regexp.MustCompile(`\.\.[\\/\\]`),
	// URL编码的路径遍历
	regexp.MustCompile(`(?i)(%2e%2e|%252e%252e)(%2f|%5c|%252f|%255c)`),
	// Unicode编码
	regexp.MustCompile(`(?i)(\\u002e\\u002e|\\u00252e\\u00252e)`),
	// 双重编码
	regexp.MustCompile(`(?i)%25%32%65%25%32%65`),
	// 绝对路径
	regexp.MustCompile(`(?i)^[a-zA-Z]:\\|^/etc/|^/var/|^/usr/|^/proc/|^/sys/`),
	// UNC路径
	regexp.MustCompile(`(?i)^\\\\[^\\]+\\`),
	// 特殊文件
	regexp.MustCompile(`(?i)(passwd|shadow|hosts|sudoers|web\.config|applicationHost\.config)`),
}

// 命令注入检测模式
var commandInjectionPatterns = []*regexp.Regexp{
	// Shell命令分隔符
	regexp.MustCompile(`[;&|]\s*[\w/]+`),
	// 命令替换
	regexp.MustCompile(`\$\([^)]+\)`),
	// 常见危险命令
	regexp.MustCompile(`(?i)\b(bash|sh|cmd|powershell|nc|netcat|wget|curl|telnet|eval|exec)\b`),
	// 管道和重定向
	regexp.MustCompile(`[<>]\s*[\w/.]+|\|\s*[\w/]+`),
	// 特殊字符组合
	regexp.MustCompile(`[\r\n]+[\w/]+`),
	// Windows命令
	regexp.MustCompile(`(?i)\b(net\s+user|net\s+localgroup|reg\s+add|schtasks)\b`),
}

// LDAP注入检测模式
var ldapInjectionPatterns = []*regexp.Regexp{
	// LDAP过滤器字符
	regexp.MustCompile(`[()&|!*]`),
	// LDAP属性操作
	regexp.MustCompile(`(?i)(objectClass|cn|sn|uid|userPassword|mail)\s*[=<>~]`),
	// LDAP通配符
	regexp.MustCompile(`\*\w*\*`),
	// NULL字节注入
	regexp.MustCompile(`\x00|%00`),
	// LDAP转义序列
	regexp.MustCompile(`\\[0-9a-fA-F]{2}`),
}

// isPathTraversal 检测路径遍历攻击
func isPathTraversal(input string) bool {
	// 空输入不检测
	if input == "" {
		return false
	}

	for _, pattern := range pathTraversalPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// isCommandInjection 检测命令注入攻击
func isCommandInjection(input string) bool {
	// 空输入不检测
	if input == "" {
		return false
	}

	for _, pattern := range commandInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// isLDAPInjection 检测LDAP注入攻击
func isLDAPInjection(input string) bool {
	// 空输入不检测
	if input == "" {
		return false
	}

	for _, pattern := range ldapInjectionPatterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}

// registerSecurityValidators 注册安全验证器
func registerSecurityValidators(v *validator.Validate) {
	// 注册无SQL注入验证器
	v.RegisterValidation("nosqlinjection", func(fl validator.FieldLevel) bool {
		return !isSQLInjection(fl.Field().String())
	})

	// 注册无XSS验证器
	v.RegisterValidation("noxss", func(fl validator.FieldLevel) bool {
		return !isXSSAttack(fl.Field().String())
	})

	// 注册安全文件名验证器
	v.RegisterValidation("safefilename", func(fl validator.FieldLevel) bool {
		filename := fl.Field().String()
		// 只允许字母、数字、下划线、连字符和点
		matched, _ := regexp.MatchString(`^[a-zA-Z0-9_\-\.]+$`, filename)
		return matched && !isPathTraversal(filename)
	})

	// 注册安全路径验证器
	v.RegisterValidation("safepath", func(fl validator.FieldLevel) bool {
		path := fl.Field().String()
		return !isPathTraversal(path)
	})

	// 注册强密码验证器
	v.RegisterValidation("strongpassword", func(fl validator.FieldLevel) bool {
		password := fl.Field().String()
		if len(password) < 8 {
			return false
		}

		hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
		hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
		hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
		hasSpecial := regexp.MustCompile(`[!@#$%^&*(),.?":{}|<>]`).MatchString(password)

		return hasUpper && hasLower && hasNumber && hasSpecial
	})

	// 注册安全URL验证器
	v.RegisterValidation("safeurl", func(fl validator.FieldLevel) bool {
		url := fl.Field().String()
		// 防止javascript:, data:, vbscript: 等危险协议
		dangerousProtocols := []string{"javascript:", "data:", "vbscript:", "file:", "about:"}
		urlLower := strings.ToLower(url)
		for _, protocol := range dangerousProtocols {
			if strings.HasPrefix(urlLower, protocol) {
				return false
			}
		}
		return true
	})

	// 注册无命令注入验证器
	v.RegisterValidation("nocommandinjection", func(fl validator.FieldLevel) bool {
		return !isCommandInjection(fl.Field().String())
	})

	// 注册无LDAP注入验证器
	v.RegisterValidation("noldapinjection", func(fl validator.FieldLevel) bool {
		return !isLDAPInjection(fl.Field().String())
	})

	// 注册安全的JSON字段验证器
	v.RegisterValidation("safejson", func(fl validator.FieldLevel) bool {
		value := fl.Field().String()
		return !isSQLInjection(value) && !isXSSAttack(value) && !isCommandInjection(value)
	})
}

// defaultValidationErrorHandler 默认验证错误处理
func defaultValidationErrorHandler(c *gin.Context, err error) {
	c.JSON(http.StatusBadRequest, gin.H{
		"error":   "输入验证失败",
		"code":    "VALIDATION_FAILED",
		"details": err.Error(),
	})
}

// ValidateStruct 验证结构体
func (m *InputValidatorMiddleware) ValidateStruct(s interface{}) error {
	return m.validator.Struct(s)
}

// InputValidation 便捷函数，使用默认配置创建输入验证中间件
func InputValidation() gin.HandlerFunc {
	middleware := NewInputValidatorMiddleware(nil)
	return middleware.Handler()
}

// InputValidationWithConfig 使用自定义配置创建输入验证中间件
func InputValidationWithConfig(config ValidatorConfig) gin.HandlerFunc {
	middleware := NewInputValidatorMiddleware(&config)
	return middleware.Handler()
}
