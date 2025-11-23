package utils

import (
	"html"
	"regexp"
	"strings"
	"unicode"
)

// SanitizeString 清理字符串，移除或转义危险字符
func SanitizeString(input string) string {
	// 1. HTML 转义，防止 XSS
	sanitized := html.EscapeString(input)
	
	// 2. 移除控制字符（除了换行符和制表符）
	var result strings.Builder
	for _, r := range sanitized {
		if unicode.IsControl(r) && r != '\n' && r != '\t' {
			continue
		}
		result.WriteRune(r)
	}
	
	return result.String()
}

// ValidateTemplateName 验证模板名称
func ValidateTemplateName(name string) error {
	// 1. 检查是否为空或仅包含空白字符
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ErrEmptyName
	}
	
	// 2. 检查长度（最大 255 字符）
	if len(trimmed) > 255 {
		return ErrNameTooLong
	}
	
	// 3. 检查是否包含危险字符（XSS、SQL 注入等）
	if containsDangerousChars(trimmed) {
		return ErrDangerousChars
	}
	
	return nil
}

// ValidateTaskID 验证任务 ID 格式
func ValidateTaskID(id string) error {
	// 1. 检查是否为空
	if id == "" {
		return ErrEmptyID
	}
	
	// 2. 检查格式（只允许字母、数字、连字符、下划线）
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, id)
	if !matched {
		return ErrInvalidIDFormat
	}
	
	// 3. 检查长度（最大 64 字符）
	if len(id) > 64 {
		return ErrIDTooLong
	}
	
	return nil
}

// ValidateTemplateID 验证模板 ID 格式
func ValidateTemplateID(id string) error {
	return ValidateTaskID(id) // 使用相同的验证规则
}

// containsDangerousChars 检查字符串是否包含危险字符
func containsDangerousChars(s string) bool {
	// 检查常见的 XSS 和 SQL 注入模式
	dangerousPatterns := []string{
		"<script",
		"</script>",
		"javascript:",
		"onerror=",
		"onload=",
		"';",
		"'; --",
		"DROP TABLE",
		"DELETE FROM",
		"INSERT INTO",
		"UPDATE SET",
		"UNION SELECT",
		"<iframe",
		"<img",
		"<svg",
	}
	
	lower := strings.ToLower(s)
	for _, pattern := range dangerousPatterns {
		if strings.Contains(lower, pattern) {
			return true
		}
	}
	
	return false
}

// TrimAndValidate 清理并验证字符串
func TrimAndValidate(s string, maxLen int) (string, error) {
	// 1. 去除首尾空白字符
	trimmed := strings.TrimSpace(s)
	
	// 2. 检查是否为空
	if trimmed == "" {
		return "", ErrEmptyString
	}
	
	// 3. 检查长度
	if maxLen > 0 && len(trimmed) > maxLen {
		return "", ErrStringTooLong
	}
	
	// 4. 清理危险字符
	sanitized := SanitizeString(trimmed)
	
	return sanitized, nil
}

// 错误定义
var (
	ErrEmptyName       = &ValidationError{Code: "EMPTY_NAME", Message: "name cannot be empty"}
	ErrNameTooLong     = &ValidationError{Code: "NAME_TOO_LONG", Message: "name exceeds maximum length"}
	ErrDangerousChars  = &ValidationError{Code: "DANGEROUS_CHARS", Message: "name contains dangerous characters"}
	ErrEmptyID         = &ValidationError{Code: "EMPTY_ID", Message: "id cannot be empty"}
	ErrInvalidIDFormat = &ValidationError{Code: "INVALID_ID_FORMAT", Message: "id contains invalid characters"}
	ErrIDTooLong       = &ValidationError{Code: "ID_TOO_LONG", Message: "id exceeds maximum length"}
	ErrEmptyString     = &ValidationError{Code: "EMPTY_STRING", Message: "string cannot be empty"}
	ErrStringTooLong   = &ValidationError{Code: "STRING_TOO_LONG", Message: "string exceeds maximum length"}
)

// ValidationError 验证错误
type ValidationError struct {
	Code    string
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}


