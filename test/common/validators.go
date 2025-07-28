package common

import (
	"fmt"
	"regexp"
	"strings"
)

// Common validation functions used across multiple test files

// ValidateEmail 验证邮箱格式
func ValidateEmail(email string) error {
	if email == "" {
		return fmt.Errorf("邮箱不能为空")
	}

	emailPattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailPattern.MatchString(email) {
		return fmt.Errorf("邮箱格式不正确")
	}

	if len(email) > 100 {
		return fmt.Errorf("邮箱长度不能超过100字符")
	}

	return nil
}

// ValidateBranchName 验证Git分支名称
func ValidateBranchName(branch string) error {
	if branch == "" {
		return fmt.Errorf("分支名称不能为空")
	}

	if len(branch) > 255 {
		return fmt.Errorf("分支名称长度不能超过255字符")
	}

	// Git分支名称规则：允许字母、数字、斜杠、横线、下划线、点号
	// 不能以点开头或结尾，不能以斜杠结尾
	if strings.HasPrefix(branch, ".") || strings.HasSuffix(branch, ".") || strings.HasSuffix(branch, "/") {
		return fmt.Errorf("分支名称格式不正确")
	}

	// 检查不允许的字符
	invalidChars := []string{" ", "~", "^", ":", "?", "*", "[", "\\", "@{", ".."}
	for _, char := range invalidChars {
		if strings.Contains(branch, char) {
			return fmt.Errorf("分支名称不能包含字符: %s", char)
		}
	}

	// 基本字符检查
	branchPattern := regexp.MustCompile(`^[a-zA-Z0-9/_\-\.]+$`)
	if !branchPattern.MatchString(branch) {
		return fmt.Errorf("分支名称格式不正确")
	}

	return nil
}

// ValidateCommitSHA 验证Git提交SHA
func ValidateCommitSHA(sha string) error {
	if sha == "" {
		return fmt.Errorf("提交SHA不能为空")
	}

	// Git SHA可以是短SHA(7-40位)或完整SHA(40位)
	if len(sha) < 7 || len(sha) > 40 {
		return fmt.Errorf("提交SHA长度必须在7-40字符之间")
	}

	// SHA只能包含十六进制字符
	shaPattern := regexp.MustCompile(`^[a-fA-F0-9]+$`)
	if !shaPattern.MatchString(sha) {
		return fmt.Errorf("提交SHA只能包含十六进制字符")
	}

	return nil
}

// ValidatePermission 验证权限
func ValidatePermission(permission string) error {
	if permission == "" {
		return fmt.Errorf("权限不能为空")
	}

	validPermissions := []string{"read", "write", "admin", "owner"}
	for _, validPerm := range validPermissions {
		if permission == validPerm {
			return nil
		}
	}

	return fmt.Errorf("无效的权限: %s", permission)
}
