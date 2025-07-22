package service

import (
	"fmt"
	"strings"
)

const (
	// LexorankBase 基数系统，使用0-9a-z (36进制)
	LexorankBase = 36
	
	// LexorankMinChar 最小字符
	LexorankMinChar = '0'
	
	// LexorankMaxChar 最大字符  
	LexorankMaxChar = 'z'
	
	// LexorankMidChar 中间字符
	LexorankMidChar = 'U' // 大约在中间位置
	
	// LexorankScale 精度位数
	LexorankScale = 6
)

// Lexorank Lexorank算法实现
type Lexorank struct {
	value string
}

// NewLexorank 创建Lexorank实例
func NewLexorank(value string) *Lexorank {
	if value == "" {
		value = strings.Repeat(string(LexorankMidChar), LexorankScale)
	}
	return &Lexorank{value: value}
}

// NewInitialLexorank 创建初始Lexorank
func NewInitialLexorank() *Lexorank {
	return NewLexorank(strings.Repeat(string(LexorankMidChar), LexorankScale))
}

// GetValue 获取Lexorank值
func (l *Lexorank) GetValue() string {
	return l.value
}

// GenNext 生成下一个Lexorank（在当前值之后）
func (l *Lexorank) GenNext() *Lexorank {
	return l.generateNext(false)
}

// GenPrev 生成上一个Lexorank（在当前值之前）
func (l *Lexorank) GenPrev() *Lexorank {
	return l.generateNext(true)
}

// GenBetween 在两个Lexorank之间生成新值
func GenBetween(prev, next *Lexorank) (*Lexorank, error) {
	if prev != nil && next != nil {
		cmp := strings.Compare(prev.value, next.value)
		if cmp >= 0 {
			return nil, fmt.Errorf("prev rank must be less than next rank")
		}
	}
	
	var prevVal, nextVal string
	
	if prev == nil {
		prevVal = strings.Repeat(string(LexorankMinChar), LexorankScale)
	} else {
		prevVal = prev.value
	}
	
	if next == nil {
		nextVal = strings.Repeat(string(LexorankMaxChar), LexorankScale)
	} else {
		nextVal = next.value
	}
	
	// 确保两个值长度一致
	maxLen := len(prevVal)
	if len(nextVal) > maxLen {
		maxLen = len(nextVal)
	}
	
	prevVal = padRight(prevVal, maxLen, LexorankMinChar)
	nextVal = padRight(nextVal, maxLen, LexorankMaxChar)
	
	// 生成中间值
	middleVal := generateMiddle(prevVal, nextVal)
	
	return NewLexorank(middleVal), nil
}

// generateNext 生成相邻的下一个或上一个值
func (l *Lexorank) generateNext(isPrev bool) *Lexorank {
	value := l.value
	result := make([]rune, len(value))
	
	for i, char := range value {
		result[i] = char
	}
	
	if isPrev {
		// 生成上一个值：向前递减
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] > LexorankMinChar {
				result[i]--
				// 后面的位数设为最大值
				for j := i + 1; j < len(result); j++ {
					result[j] = LexorankMaxChar
				}
				break
			}
		}
	} else {
		// 生成下一个值：向后递增
		for i := len(result) - 1; i >= 0; i-- {
			if result[i] < LexorankMaxChar {
				result[i]++
				// 后面的位数设为最小值
				for j := i + 1; j < len(result); j++ {
					result[j] = LexorankMinChar
				}
				break
			}
		}
	}
	
	return NewLexorank(string(result))
}

// generateMiddle 生成两个字符串之间的中间值
func generateMiddle(prev, next string) string {
	if len(prev) != len(next) {
		maxLen := len(prev)
		if len(next) > maxLen {
			maxLen = len(next)
		}
		prev = padRight(prev, maxLen, LexorankMinChar)
		next = padRight(next, maxLen, LexorankMaxChar)
	}
	
	result := make([]rune, len(prev))
	carry := 0
	
	for i := len(prev) - 1; i >= 0; i-- {
		prevChar := rune(prev[i])
		nextChar := rune(next[i])
		
		prevVal := charToValue(prevChar)
		nextVal := charToValue(nextChar)
		
		sum := prevVal + nextVal + carry
		mid := sum / 2
		carry = sum % 2
		
		result[i] = valueToChar(mid)
		
		// 如果差值大于1，找到了合适的中间值
		if nextVal-prevVal > 1 {
			carry = 0
			break
		}
	}
	
	// 处理进位，如果需要增加精度
	if carry > 0 {
		// 向右延伸一位
		newResult := make([]rune, len(result)+1)
		copy(newResult, result)
		newResult[len(result)] = LexorankMidChar
		result = newResult
	}
	
	return string(result)
}

// charToValue 将字符转换为数值
func charToValue(char rune) int {
	if char >= '0' && char <= '9' {
		return int(char - '0')
	}
	if char >= 'A' && char <= 'Z' {
		return int(char-'A') + 10
	}
	if char >= 'a' && char <= 'z' {
		return int(char-'a') + 10
	}
	return 0
}

// valueToChar 将数值转换为字符
func valueToChar(val int) rune {
	if val >= 0 && val <= 9 {
		return rune('0' + val)
	}
	if val >= 10 && val <= 35 {
		return rune('A' + val - 10)
	}
	if val >= 36 && val <= 61 {
		return rune('a' + val - 36)
	}
	return '0'
}

// padRight 右填充字符串
func padRight(str string, length int, padChar rune) string {
	if len(str) >= length {
		return str
	}
	
	padding := strings.Repeat(string(padChar), length-len(str))
	return str + padding
}

// Compare 比较两个Lexorank值
func (l *Lexorank) Compare(other *Lexorank) int {
	return strings.Compare(l.value, other.value)
}

// IsValid 验证Lexorank值是否有效
func (l *Lexorank) IsValid() bool {
	if len(l.value) == 0 {
		return false
	}
	
	for _, char := range l.value {
		if !isValidLexorankChar(char) {
			return false
		}
	}
	
	return true
}

// isValidLexorankChar 检查字符是否为有效的Lexorank字符
func isValidLexorankChar(char rune) bool {
	return (char >= '0' && char <= '9') ||
		   (char >= 'A' && char <= 'Z') ||
		   (char >= 'a' && char <= 'z')
}

// LexorankManager Lexorank管理器
type LexorankManager struct{}

// NewLexorankManager 创建Lexorank管理器
func NewLexorankManager() *LexorankManager {
	return &LexorankManager{}
}

// CalculateRankForPosition 为指定位置计算Lexorank
func (lm *LexorankManager) CalculateRankForPosition(prevRank, nextRank *string) (string, error) {
	var prev, next *Lexorank
	
	if prevRank != nil {
		prev = NewLexorank(*prevRank)
		if !prev.IsValid() {
			return "", fmt.Errorf("invalid previous rank: %s", *prevRank)
		}
	}
	
	if nextRank != nil {
		next = NewLexorank(*nextRank)
		if !next.IsValid() {
			return "", fmt.Errorf("invalid next rank: %s", *nextRank)
		}
	}
	
	// 在两个值之间生成新的rank
	newRank, err := GenBetween(prev, next)
	if err != nil {
		return "", fmt.Errorf("failed to generate rank between %v and %v: %w", prevRank, nextRank, err)
	}
	
	return newRank.GetValue(), nil
}

// CalculateRankForInsertion 为插入操作计算Lexorank
func (lm *LexorankManager) CalculateRankForInsertion(position int, existingRanks []string) (string, error) {
	if len(existingRanks) == 0 {
		// 如果没有现有排名，返回初始值
		return NewInitialLexorank().GetValue(), nil
	}
	
	// 验证位置
	if position < 0 {
		position = 0
	}
	if position > len(existingRanks) {
		position = len(existingRanks)
	}
	
	var prevRank, nextRank *string
	
	if position > 0 {
		prevRank = &existingRanks[position-1]
	}
	
	if position < len(existingRanks) {
		nextRank = &existingRanks[position]
	}
	
	return lm.CalculateRankForPosition(prevRank, nextRank)
}

// RebalanceRanks 重新平衡排名（当排名过于接近时）
func (lm *LexorankManager) RebalanceRanks(ranks []string) ([]string, error) {
	if len(ranks) <= 1 {
		return ranks, nil
	}
	
	// 检查是否需要重新平衡
	needRebalance := false
	for i := 0; i < len(ranks)-1; i++ {
		rank1 := NewLexorank(ranks[i])
		rank2 := NewLexorank(ranks[i+1])
		
		if !rank1.IsValid() || !rank2.IsValid() {
			needRebalance = true
			break
		}
		
		// 如果两个排名过于接近，需要重新平衡
		if isRanksTooClose(rank1, rank2) {
			needRebalance = true
			break
		}
	}
	
	if !needRebalance {
		return ranks, nil
	}
	
	// 重新生成平衡的排名
	newRanks := make([]string, len(ranks))
	
	// 生成均匀分布的排名
	minRank := NewLexorank(strings.Repeat(string(LexorankMinChar), LexorankScale))
	maxRank := NewLexorank(strings.Repeat(string(LexorankMaxChar), LexorankScale))
	
	for i := 0; i < len(ranks); i++ {
		ratio := float64(i+1) / float64(len(ranks)+1)
		newRank, err := lm.generateRankByRatio(minRank, maxRank, ratio)
		if err != nil {
			return nil, fmt.Errorf("failed to generate balanced rank at position %d: %w", i, err)
		}
		newRanks[i] = newRank.GetValue()
	}
	
	return newRanks, nil
}

// generateRankByRatio 根据比例生成排名
func (lm *LexorankManager) generateRankByRatio(min, max *Lexorank, ratio float64) (*Lexorank, error) {
	if ratio <= 0.0 {
		return min.GenNext(), nil
	}
	if ratio >= 1.0 {
		return max.GenPrev(), nil
	}
	
	minVal := min.GetValue()
	maxVal := max.GetValue()
	
	// 简化实现：根据比例插值
	result := make([]rune, len(minVal))
	
	for i := 0; i < len(minVal) && i < len(maxVal); i++ {
		minCharVal := charToValue(rune(minVal[i]))
		maxCharVal := charToValue(rune(maxVal[i]))
		
		interpolated := minCharVal + int(float64(maxCharVal-minCharVal)*ratio)
		result[i] = valueToChar(interpolated)
	}
	
	return NewLexorank(string(result)), nil
}

// isRanksTooClose 检查两个排名是否过于接近
func isRanksTooClose(rank1, rank2 *Lexorank) bool {
	val1 := rank1.GetValue()
	val2 := rank2.GetValue()
	
	if len(val1) != len(val2) {
		return false
	}
	
	// 检查是否只有最后一位不同，且差值为1
	differences := 0
	lastDiffIndex := -1
	
	for i := 0; i < len(val1); i++ {
		if val1[i] != val2[i] {
			differences++
			lastDiffIndex = i
		}
	}
	
	if differences == 1 && lastDiffIndex == len(val1)-1 {
		char1 := rune(val1[lastDiffIndex])
		char2 := rune(val2[lastDiffIndex])
		
		val1Num := charToValue(char1)
		val2Num := charToValue(char2)
		
		return abs(val2Num-val1Num) <= 1
	}
	
	return false
}

// abs 返回整数的绝对值
func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// ValidateLexorankSequence 验证Lexorank序列是否有效
func ValidateLexorankSequence(ranks []string) error {
	if len(ranks) <= 1 {
		return nil
	}
	
	for i := 0; i < len(ranks); i++ {
		rank := NewLexorank(ranks[i])
		if !rank.IsValid() {
			return fmt.Errorf("invalid rank at position %d: %s", i, ranks[i])
		}
		
		if i > 0 {
			prevRank := NewLexorank(ranks[i-1])
			if rank.Compare(prevRank) <= 0 {
				return fmt.Errorf("rank at position %d (%s) is not greater than previous rank (%s)", 
					i, ranks[i], ranks[i-1])
			}
		}
	}
	
	return nil
}