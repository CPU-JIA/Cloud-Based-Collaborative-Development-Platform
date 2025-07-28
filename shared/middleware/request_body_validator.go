package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"github.com/cloud-platform/collaborative-dev/shared/logger"
	"github.com/gin-gonic/gin"
)

// RequestBodyValidator JSON请求体深度验证器
type RequestBodyValidator struct {
	config *ValidatorConfig
	logger logger.Logger
}

// NewRequestBodyValidator 创建请求体验证器
func NewRequestBodyValidator(config *ValidatorConfig) *RequestBodyValidator {
	if config == nil {
		cfg := DefaultValidatorConfig
		config = &cfg
	}

	return &RequestBodyValidator{
		config: config,
		logger: config.Logger,
	}
}

// ValidateJSON Gin中间件 - 深度验证JSON请求体
func (v *RequestBodyValidator) ValidateJSON() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否为JSON内容类型
		if !isJSONContent(c.ContentType()) {
			c.Next()
			return
		}

		// 读取请求体
		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			v.handleError(c, fmt.Errorf("读取请求体失败: %w", err))
			c.Abort()
			return
		}

		// 恢复请求体以供后续使用
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		// 如果请求体为空，继续处理
		if len(body) == 0 {
			c.Next()
			return
		}

		// 解析JSON
		var data interface{}
		decoder := json.NewDecoder(bytes.NewBuffer(body))
		decoder.UseNumber() // 使用json.Number以保持数字精度

		if err := decoder.Decode(&data); err != nil {
			v.handleError(c, fmt.Errorf("无效的JSON格式: %w", err))
			c.Abort()
			return
		}

		// 验证JSON深度
		if v.config.MaxJSONDepth > 0 {
			if depth := v.getJSONDepth(data, 0); depth > v.config.MaxJSONDepth {
				v.handleError(c, fmt.Errorf("JSON嵌套深度超过限制: %d > %d", depth, v.config.MaxJSONDepth))
				c.Abort()
				return
			}
		}

		// 深度扫描JSON数据
		if v.config.EnableDeepJSONValidation {
			if err := v.validateJSONData(data, ""); err != nil {
				v.handleError(c, err)
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// validateJSONData 递归验证JSON数据
func (v *RequestBodyValidator) validateJSONData(data interface{}, path string) error {
	switch value := data.(type) {
	case string:
		return v.validateString(value, path)

	case map[string]interface{}:
		for key, val := range value {
			// 验证键名
			if err := v.validateString(key, fmt.Sprintf("%s.%s(key)", path, key)); err != nil {
				return err
			}

			// 递归验证值
			newPath := key
			if path != "" {
				newPath = fmt.Sprintf("%s.%s", path, key)
			}

			if err := v.validateJSONData(val, newPath); err != nil {
				return err
			}
		}

	case []interface{}:
		for i, val := range value {
			newPath := fmt.Sprintf("%s[%d]", path, i)
			if path == "" {
				newPath = fmt.Sprintf("[%d]", i)
			}

			if err := v.validateJSONData(val, newPath); err != nil {
				return err
			}
		}

	case json.Number:
		// 数字类型不需要安全验证
		return nil

	case bool, nil:
		// 布尔值和null不需要验证
		return nil

	default:
		// 其他类型转换为字符串验证
		if strVal := fmt.Sprintf("%v", value); strVal != "" {
			return v.validateString(strVal, path)
		}
	}

	return nil
}

// validateString 验证字符串值
func (v *RequestBodyValidator) validateString(value string, path string) error {
	location := "JSON字段"
	if path != "" {
		location = fmt.Sprintf("JSON字段 '%s'", path)
	}

	// SQL注入检测
	if v.config.EnableSQLInjectionProtection {
		if isSQLInjection(value) {
			return fmt.Errorf("检测到SQL注入攻击：%s", location)
		}
	}

	// XSS攻击检测
	if v.config.EnableXSSProtection {
		if isXSSAttack(value) {
			return fmt.Errorf("检测到XSS攻击：%s", location)
		}
	}

	// 路径遍历检测
	if v.config.EnablePathTraversalProtection {
		// 只对可能是文件路径的字段进行检测
		if looksLikeFilePath(value) && isPathTraversal(value) {
			return fmt.Errorf("检测到路径遍历攻击：%s", location)
		}
	}

	// 命令注入检测
	if v.config.EnableCommandInjectionProtection {
		if isCommandInjection(value) {
			return fmt.Errorf("检测到命令注入攻击：%s", location)
		}
	}

	// LDAP注入检测
	if v.config.EnableLDAPInjectionProtection {
		// 只对可能是LDAP查询的字段进行检测
		if looksLikeLDAPQuery(value) && isLDAPInjection(value) {
			return fmt.Errorf("检测到LDAP注入攻击：%s", location)
		}
	}

	return nil
}

// getJSONDepth 计算JSON数据的最大深度
func (v *RequestBodyValidator) getJSONDepth(data interface{}, currentDepth int) int {
	maxDepth := currentDepth

	switch value := data.(type) {
	case map[string]interface{}:
		currentDepth++
		if currentDepth > maxDepth {
			maxDepth = currentDepth
		}

		for _, val := range value {
			if depth := v.getJSONDepth(val, currentDepth); depth > maxDepth {
				maxDepth = depth
			}
		}

	case []interface{}:
		currentDepth++
		if currentDepth > maxDepth {
			maxDepth = currentDepth
		}

		for _, val := range value {
			if depth := v.getJSONDepth(val, currentDepth); depth > maxDepth {
				maxDepth = depth
			}
		}
	}

	return maxDepth
}

// handleError 处理验证错误
func (v *RequestBodyValidator) handleError(c *gin.Context, err error) {
	if v.logger != nil {
		v.logger.Warn("JSON验证失败",
			"path", c.Request.URL.Path,
			"method", c.Request.Method,
			"error", err.Error(),
			"ip", c.ClientIP(),
		)
	}

	if v.config.ErrorHandler != nil {
		v.config.ErrorHandler(c, err)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "输入验证失败",
			"code":    "JSON_VALIDATION_FAILED",
			"details": err.Error(),
		})
	}
}

// isJSONContent 检查是否为JSON内容类型
func isJSONContent(contentType string) bool {
	return contentType == "application/json" ||
		contentType == "text/json" ||
		contentType == "application/json; charset=utf-8" ||
		contentType == "text/json; charset=utf-8"
}

// looksLikeFilePath 简单判断字符串是否像文件路径
func looksLikeFilePath(s string) bool {
	// 包含路径分隔符或文件扩展名
	return len(s) > 0 && (bytes.ContainsAny([]byte(s), "/\\") ||
		(len(s) > 4 && s[len(s)-4] == '.') ||
		(len(s) > 5 && s[len(s)-5] == '.'))
}

// looksLikeLDAPQuery 简单判断字符串是否像LDAP查询
func looksLikeLDAPQuery(s string) bool {
	// 包含LDAP特征字符
	return len(s) > 0 && (bytes.ContainsAny([]byte(s), "()&|=") ||
		bytes.Contains([]byte(s), []byte("cn=")) ||
		bytes.Contains([]byte(s), []byte("uid=")) ||
		bytes.Contains([]byte(s), []byte("objectClass=")))
}

// ValidateStructWithJSON 验证结构体并进行JSON字段深度验证
func (v *RequestBodyValidator) ValidateStructWithJSON(c *gin.Context, obj interface{}) error {
	// 首先进行标准的结构体验证
	if err := c.ShouldBindJSON(obj); err != nil {
		return err
	}

	// 如果启用了深度验证，对结构体进行反射验证
	if v.config.EnableDeepJSONValidation {
		return v.validateStructFields(obj, "")
	}

	return nil
}

// validateStructFields 使用反射验证结构体字段
func (v *RequestBodyValidator) validateStructFields(obj interface{}, path string) error {
	val := reflect.ValueOf(obj)

	// 处理指针
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return nil
		}
		val = val.Elem()
	}

	// 只处理结构体
	if val.Kind() != reflect.Struct {
		return nil
	}

	typ := val.Type()

	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 跳过私有字段
		if !field.CanInterface() {
			continue
		}

		// 获取字段名
		fieldName := fieldType.Name
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag != "" && jsonTag != "-" {
			fieldName = jsonTag
		}

		// 构建路径
		fieldPath := fieldName
		if path != "" {
			fieldPath = fmt.Sprintf("%s.%s", path, fieldName)
		}

		// 根据字段类型进行验证
		switch field.Kind() {
		case reflect.String:
			if err := v.validateString(field.String(), fieldPath); err != nil {
				return err
			}

		case reflect.Struct:
			if err := v.validateStructFields(field.Interface(), fieldPath); err != nil {
				return err
			}

		case reflect.Slice, reflect.Array:
			for j := 0; j < field.Len(); j++ {
				elemPath := fmt.Sprintf("%s[%d]", fieldPath, j)
				elem := field.Index(j)

				if elem.Kind() == reflect.String {
					if err := v.validateString(elem.String(), elemPath); err != nil {
						return err
					}
				} else if elem.Kind() == reflect.Struct {
					if err := v.validateStructFields(elem.Interface(), elemPath); err != nil {
						return err
					}
				}
			}

		case reflect.Map:
			for _, key := range field.MapKeys() {
				// 验证键
				if key.Kind() == reflect.String {
					if err := v.validateString(key.String(), fmt.Sprintf("%s.%s(key)", fieldPath, key.String())); err != nil {
						return err
					}
				}

				// 验证值
				value := field.MapIndex(key)
				if value.Kind() == reflect.String {
					if err := v.validateString(value.String(), fmt.Sprintf("%s.%s", fieldPath, key.String())); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// RequestBodyValidation 便捷函数，使用默认配置创建请求体验证中间件
func RequestBodyValidation() gin.HandlerFunc {
	validator := NewRequestBodyValidator(nil)
	return validator.ValidateJSON()
}

// RequestBodyValidationWithConfig 使用自定义配置创建请求体验证中间件
func RequestBodyValidationWithConfig(config ValidatorConfig) gin.HandlerFunc {
	validator := NewRequestBodyValidator(&config)
	return validator.ValidateJSON()
}
