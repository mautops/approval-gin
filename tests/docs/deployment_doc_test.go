package docs_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestDeploymentDoc_Exists 测试部署文档是否存在
func TestDeploymentDoc_Exists(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "DEPLOYMENT.md")
	_, err := os.Stat(docPath)
	assert.NoError(t, err, "DEPLOYMENT.md should exist")
}

// TestDeploymentDoc_RequiredSections 测试部署文档是否包含必需的章节
func TestDeploymentDoc_RequiredSections(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "DEPLOYMENT.md")
	content, err := os.ReadFile(docPath)
	assert.NoError(t, err)

	docContent := string(content)
	requiredSections := []string{
		"系统要求",
		"快速开始",
		"配置说明",
		"部署方式",
		"数据库迁移",
		"环境变量",
		"健康检查",
		"故障排查",
		"生产环境",
		"监控",
		"日志",
		"备份",
	}

	for _, section := range requiredSections {
		assert.Contains(t, docContent, section, "should contain section: %s", section)
	}
}

// TestDeploymentDoc_DockerComposeExample 测试是否包含 Docker Compose 示例
func TestDeploymentDoc_DockerComposeExample(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "DEPLOYMENT.md")
	content, err := os.ReadFile(docPath)
	assert.NoError(t, err)

	docContent := string(content)
	assert.Contains(t, docContent, "docker-compose", "should contain docker-compose example")
}

// TestDeploymentDoc_EnvironmentVariables 测试是否包含环境变量说明
func TestDeploymentDoc_EnvironmentVariables(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "DEPLOYMENT.md")
	content, err := os.ReadFile(docPath)
	assert.NoError(t, err)

	docContent := string(content)
	assert.Contains(t, docContent, "APP_", "should contain environment variables")
}

// TestDeploymentDoc_ProductionConfig 测试是否包含生产环境配置说明
func TestDeploymentDoc_ProductionConfig(t *testing.T) {
	docPath := filepath.Join("..", "..", "docs", "DEPLOYMENT.md")
	content, err := os.ReadFile(docPath)
	assert.NoError(t, err)

	docContent := string(content)
	productionKeywords := []string{
		"生产环境",
		"production",
		"连接池",
		"日志级别",
	}

	hasProduction := false
	for _, keyword := range productionKeywords {
		if strings.Contains(docContent, keyword) {
			hasProduction = true
			break
		}
	}
	assert.True(t, hasProduction, "should contain production environment configuration")
}

