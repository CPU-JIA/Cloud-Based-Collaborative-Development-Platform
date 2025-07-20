package auth

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// PasswordPolicy 密码策略
type PasswordPolicy struct {
	MinLength                int           `json:"min_length" yaml:"min_length"`
	MaxLength                int           `json:"max_length" yaml:"max_length"`
	RequireUppercase         bool          `json:"require_uppercase" yaml:"require_uppercase"`
	RequireLowercase         bool          `json:"require_lowercase" yaml:"require_lowercase"`
	RequireNumbers           bool          `json:"require_numbers" yaml:"require_numbers"`
	RequireSpecialChars      bool          `json:"require_special_chars" yaml:"require_special_chars"`
	MinSpecialChars          int           `json:"min_special_chars" yaml:"min_special_chars"`
	ForbiddenPatterns        []string      `json:"forbidden_patterns" yaml:"forbidden_patterns"`
	ForbiddenWords           []string      `json:"forbidden_words" yaml:"forbidden_words"`
	MaxConsecutiveChars      int           `json:"max_consecutive_chars" yaml:"max_consecutive_chars"`
	MaxRepeatingChars        int           `json:"max_repeating_chars" yaml:"max_repeating_chars"`
	PreventUsernameInclusion bool          `json:"prevent_username_inclusion" yaml:"prevent_username_inclusion"`
	PreventEmailInclusion    bool          `json:"prevent_email_inclusion" yaml:"prevent_email_inclusion"`
	PasswordExpiry           time.Duration `json:"password_expiry" yaml:"password_expiry"`
	PasswordHistoryCount     int           `json:"password_history_count" yaml:"password_history_count"`
	AccountLockoutThreshold  int           `json:"account_lockout_threshold" yaml:"account_lockout_threshold"`
	AccountLockoutDuration   time.Duration `json:"account_lockout_duration" yaml:"account_lockout_duration"`
	PasswordComplexityScore  int           `json:"password_complexity_score" yaml:"password_complexity_score"`
}

// PasswordValidationResult 密码验证结果
type PasswordValidationResult struct {
	IsValid             bool           `json:"is_valid"`
	Score               int            `json:"score"`
	MaxScore            int            `json:"max_score"`
	Strength            string         `json:"strength"`
	Errors              []string       `json:"errors"`
	Warnings            []string       `json:"warnings"`
	Suggestions         []string       `json:"suggestions"`
	ComplexityBreakdown map[string]int `json:"complexity_breakdown"`
}

// PasswordPolicyService 密码策略服务
type PasswordPolicyService struct {
	policy PasswordPolicy
}

// NewPasswordPolicyService 创建密码策略服务
func NewPasswordPolicyService(policy PasswordPolicy) *PasswordPolicyService {
	// 设置默认策略
	if policy.MinLength == 0 {
		policy.MinLength = 8
	}
	if policy.MaxLength == 0 {
		policy.MaxLength = 128
	}
	if policy.MaxConsecutiveChars == 0 {
		policy.MaxConsecutiveChars = 3
	}
	if policy.MaxRepeatingChars == 0 {
		policy.MaxRepeatingChars = 3
	}
	if policy.PasswordComplexityScore == 0 {
		policy.PasswordComplexityScore = 60 // 60分为最低要求
	}
	if policy.AccountLockoutThreshold == 0 {
		policy.AccountLockoutThreshold = 5
	}
	if policy.AccountLockoutDuration == 0 {
		policy.AccountLockoutDuration = 30 * time.Minute
	}
	if policy.PasswordExpiry == 0 {
		policy.PasswordExpiry = 90 * 24 * time.Hour // 90天
	}
	if policy.PasswordHistoryCount == 0 {
		policy.PasswordHistoryCount = 5
	}

	// 默认禁用词汇
	if len(policy.ForbiddenWords) == 0 {
		policy.ForbiddenWords = []string{
			"password", "123456", "qwerty", "admin", "root", "user",
			"guest", "test", "demo", "login", "pass", "secret",
		}
	}

	// 默认禁用模式
	if len(policy.ForbiddenPatterns) == 0 {
		policy.ForbiddenPatterns = []string{
			`^(.)\1{2,}$`,             // 重复字符 (aaa, 111)
			`^(.)(?:(?!\1).){0,2}\1`,  // 模式重复 (aba, 121)
			`(?i)^(qwerty|asdf|zxcv)`, // 键盘模式
			`^\d+$`,                   // 纯数字
			`^[a-zA-Z]+$`,             // 纯字母
		}
	}

	return &PasswordPolicyService{policy: policy}
}

// ValidatePassword 验证密码是否符合策略
func (s *PasswordPolicyService) ValidatePassword(password, username, email string) *PasswordValidationResult {
	result := &PasswordValidationResult{
		ComplexityBreakdown: make(map[string]int),
		MaxScore:            100,
	}

	// 基本长度检查
	if len(password) < s.policy.MinLength {
		result.Errors = append(result.Errors, fmt.Sprintf("密码长度至少需要%d个字符", s.policy.MinLength))
	}
	if len(password) > s.policy.MaxLength {
		result.Errors = append(result.Errors, fmt.Sprintf("密码长度不能超过%d个字符", s.policy.MaxLength))
	}

	// 字符类型检查
	var hasUpper, hasLower, hasNumber, hasSpecial bool
	var upperCount, lowerCount, numberCount, specialCount int

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
			upperCount++
		case unicode.IsLower(char):
			hasLower = true
			lowerCount++
		case unicode.IsDigit(char):
			hasNumber = true
			numberCount++
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
			specialCount++
		}
	}

	if s.policy.RequireUppercase && !hasUpper {
		result.Errors = append(result.Errors, "密码必须包含大写字母")
	}
	if s.policy.RequireLowercase && !hasLower {
		result.Errors = append(result.Errors, "密码必须包含小写字母")
	}
	if s.policy.RequireNumbers && !hasNumber {
		result.Errors = append(result.Errors, "密码必须包含数字")
	}
	if s.policy.RequireSpecialChars && !hasSpecial {
		result.Errors = append(result.Errors, "密码必须包含特殊字符")
	}
	if s.policy.MinSpecialChars > 0 && specialCount < s.policy.MinSpecialChars {
		result.Errors = append(result.Errors, fmt.Sprintf("密码至少需要%d个特殊字符", s.policy.MinSpecialChars))
	}

	// 复杂度评分
	result.ComplexityBreakdown["length"] = s.calculateLengthScore(len(password))
	result.ComplexityBreakdown["character_variety"] = s.calculateCharVarietyScore(hasUpper, hasLower, hasNumber, hasSpecial)
	result.ComplexityBreakdown["uniqueness"] = s.calculateUniquenessScore(password)
	result.ComplexityBreakdown["patterns"] = s.calculatePatternScore(password)

	// 计算总分
	for _, score := range result.ComplexityBreakdown {
		result.Score += score
	}

	// 检查连续字符
	if s.policy.MaxConsecutiveChars > 0 {
		consecutiveCount := s.findMaxConsecutiveChars(password)
		if consecutiveCount > s.policy.MaxConsecutiveChars {
			result.Errors = append(result.Errors, fmt.Sprintf("密码不能包含超过%d个连续字符", s.policy.MaxConsecutiveChars))
		}
	}

	// 检查重复字符
	if s.policy.MaxRepeatingChars > 0 {
		repeatingCount := s.findMaxRepeatingChars(password)
		if repeatingCount > s.policy.MaxRepeatingChars {
			result.Errors = append(result.Errors, fmt.Sprintf("密码不能包含超过%d个重复字符", s.policy.MaxRepeatingChars))
		}
	}

	// 检查禁用词汇
	lowerPassword := strings.ToLower(password)
	for _, word := range s.policy.ForbiddenWords {
		if strings.Contains(lowerPassword, strings.ToLower(word)) {
			result.Errors = append(result.Errors, fmt.Sprintf("密码不能包含常见词汇: %s", word))
		}
	}

	// 检查禁用模式
	for _, pattern := range s.policy.ForbiddenPatterns {
		if matched, _ := regexp.MatchString(pattern, password); matched {
			result.Errors = append(result.Errors, "密码包含不允许的模式")
			break
		}
	}

	// 检查用户名和邮箱包含
	if s.policy.PreventUsernameInclusion && username != "" {
		if strings.Contains(lowerPassword, strings.ToLower(username)) {
			result.Errors = append(result.Errors, "密码不能包含用户名")
		}
	}
	if s.policy.PreventEmailInclusion && email != "" {
		emailPart := strings.Split(email, "@")[0]
		if strings.Contains(lowerPassword, strings.ToLower(emailPart)) {
			result.Errors = append(result.Errors, "密码不能包含邮箱地址")
		}
	}

	// 确定密码强度
	result.Strength = s.getPasswordStrength(result.Score)

	// 检查是否达到最低复杂度要求
	if result.Score < s.policy.PasswordComplexityScore {
		result.Errors = append(result.Errors, fmt.Sprintf("密码复杂度不足，当前分数%d，要求%d", result.Score, s.policy.PasswordComplexityScore))
	}

	// 生成建议
	result.Suggestions = s.generateSuggestions(password, result)

	result.IsValid = len(result.Errors) == 0

	return result
}

// calculateLengthScore 计算长度分数
func (s *PasswordPolicyService) calculateLengthScore(length int) int {
	if length < s.policy.MinLength {
		return 0
	}
	// 基础分数 + 额外长度奖励（每个字符1分，最多20分）
	baseScore := 15
	extraScore := length - s.policy.MinLength
	if extraScore > 20 {
		extraScore = 20
	}
	return baseScore + extraScore
}

// calculateCharVarietyScore 计算字符多样性分数
func (s *PasswordPolicyService) calculateCharVarietyScore(hasUpper, hasLower, hasNumber, hasSpecial bool) int {
	score := 0
	if hasLower {
		score += 5
	}
	if hasUpper {
		score += 5
	}
	if hasNumber {
		score += 10
	}
	if hasSpecial {
		score += 15
	}
	return score
}

// calculateUniquenessScore 计算唯一性分数
func (s *PasswordPolicyService) calculateUniquenessScore(password string) int {
	uniqueChars := make(map[rune]bool)
	for _, char := range password {
		uniqueChars[char] = true
	}
	uniqueRatio := float64(len(uniqueChars)) / float64(len(password))
	return int(uniqueRatio * 20) // 最多20分
}

// calculatePatternScore 计算模式分数（反向评分，越少模式越高分）
func (s *PasswordPolicyService) calculatePatternScore(password string) int {
	score := 15 // 基础分数

	// 检查常见模式并扣分
	patterns := []string{
		`(.)\1{2,}`, // 重复字符
		`0123|1234|2345|3456|4567|5678|6789|7890`, // 连续数字
		`abcd|bcde|cdef|defg|efgh|fghi|ghij`,      // 连续字母
		`qwer|wert|erty|rtyu|tyui|yuio|uiop`,      // 键盘模式
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(`(?i)`+pattern, password); matched {
			score -= 3
		}
	}

	if score < 0 {
		score = 0
	}
	return score
}

// findMaxConsecutiveChars 查找最大连续字符数
func (s *PasswordPolicyService) findMaxConsecutiveChars(password string) int {
	if len(password) <= 1 {
		return len(password)
	}

	maxCount := 1
	currentCount := 1

	for i := 1; i < len(password); i++ {
		if password[i] == password[i-1]+1 || password[i] == password[i-1]-1 {
			currentCount++
		} else {
			if currentCount > maxCount {
				maxCount = currentCount
			}
			currentCount = 1
		}
	}

	if currentCount > maxCount {
		maxCount = currentCount
	}

	return maxCount
}

// findMaxRepeatingChars 查找最大重复字符数
func (s *PasswordPolicyService) findMaxRepeatingChars(password string) int {
	if len(password) <= 1 {
		return len(password)
	}

	maxCount := 1
	currentCount := 1

	for i := 1; i < len(password); i++ {
		if password[i] == password[i-1] {
			currentCount++
		} else {
			if currentCount > maxCount {
				maxCount = currentCount
			}
			currentCount = 1
		}
	}

	if currentCount > maxCount {
		maxCount = currentCount
	}

	return maxCount
}

// getPasswordStrength 根据分数确定密码强度
func (s *PasswordPolicyService) getPasswordStrength(score int) string {
	switch {
	case score >= 90:
		return "很强"
	case score >= 70:
		return "强"
	case score >= 50:
		return "中等"
	case score >= 30:
		return "弱"
	default:
		return "很弱"
	}
}

// generateSuggestions 生成密码改进建议
func (s *PasswordPolicyService) generateSuggestions(password string, result *PasswordValidationResult) []string {
	var suggestions []string

	if len(password) < s.policy.MinLength {
		suggestions = append(suggestions, fmt.Sprintf("增加密码长度至%d个字符", s.policy.MinLength))
	}

	if result.ComplexityBreakdown["character_variety"] < 20 {
		suggestions = append(suggestions, "使用大写字母、小写字母、数字和特殊字符的组合")
	}

	if result.Score < s.policy.PasswordComplexityScore {
		suggestions = append(suggestions, "避免使用常见词汇、连续字符或重复字符")
		suggestions = append(suggestions, "考虑使用密码短语或随机生成的密码")
	}

	return suggestions
}

// GetPolicy 获取当前密码策略
func (s *PasswordPolicyService) GetPolicy() PasswordPolicy {
	return s.policy
}

// IsPasswordExpired 检查密码是否过期
func (s *PasswordPolicyService) IsPasswordExpired(lastPasswordChange time.Time) bool {
	if s.policy.PasswordExpiry == 0 {
		return false
	}
	return time.Since(lastPasswordChange) > s.policy.PasswordExpiry
}

// ShouldLockAccount 检查是否应该锁定账户
func (s *PasswordPolicyService) ShouldLockAccount(failedAttempts int) bool {
	return failedAttempts >= s.policy.AccountLockoutThreshold
}

// GetLockoutDuration 获取账户锁定时长
func (s *PasswordPolicyService) GetLockoutDuration() time.Duration {
	return s.policy.AccountLockoutDuration
}
