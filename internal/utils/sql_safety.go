package utils

import (
	"errors"
	"regexp"
	"strings"
)

// ValidateSortField 验证排序字段，防止 SQL 注入
func ValidateSortField(field string) error {
	if field == "" {
		return errors.New("sort field cannot be empty")
	}
	
	// 只允许字母、数字、下划线和点（用于表名.字段名）
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_.]+$`, field)
	if !matched {
		return errors.New("invalid sort field format")
	}
	
	// 检查是否包含 SQL 关键字（防止注入）
	// 注意：只检查完整的单词，避免误判（如 "created_at" 包含 "AT"）
	sqlKeywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "CREATE",
		"EXEC", "EXECUTE", "UNION", "SCRIPT", "DECLARE", "CAST", "CONVERT",
		"FROM", "WHERE", "ORDER", "BY", "GROUP", "HAVING", "JOIN", "INNER",
		"OUTER", "LEFT", "RIGHT", "ON", "AS", "AND", "OR", "NOT", "IN",
	}
	
	upperField := strings.ToUpper(field)
	// 使用单词边界检查，避免误判
	for _, keyword := range sqlKeywords {
		// 检查关键字是否作为完整单词出现（前后是边界或非字母数字字符）
		pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(keyword) + `\b`)
		if pattern.MatchString(upperField) {
			return errors.New("sort field contains SQL keyword")
		}
	}
	
	return nil
}

// ValidateSortOrder 验证排序方向
func ValidateSortOrder(order string) error {
	upperOrder := strings.ToUpper(strings.TrimSpace(order))
	if upperOrder != "ASC" && upperOrder != "DESC" {
		return errors.New("sort order must be ASC or DESC")
	}
	return nil
}

// SanitizeSortField 清理排序字段
func SanitizeSortField(field string) string {
	// 移除所有非字母数字、下划线和点的字符
	reg := regexp.MustCompile(`[^a-zA-Z0-9_.]`)
	return reg.ReplaceAllString(field, "")
}

// SanitizeSortOrder 清理排序方向
func SanitizeSortOrder(order string) string {
	upperOrder := strings.ToUpper(strings.TrimSpace(order))
	if upperOrder == "ASC" || upperOrder == "DESC" {
		return upperOrder
	}
	return "DESC" // 默认降序
}

