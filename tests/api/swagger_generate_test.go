package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwaggerGenerate_DocsExist 测试 Swagger 文档文件是否存在
func TestSwaggerGenerate_DocsExist(t *testing.T) {
	docsDir := filepath.Join("..", "..", "docs")

	// 检查 docs.go 是否存在
	docsGo := filepath.Join(docsDir, "docs.go")
	_, err := os.Stat(docsGo)
	require.NoError(t, err, "docs.go should exist")

	// 检查 swagger.json 是否存在
	swaggerJSON := filepath.Join(docsDir, "swagger.json")
	_, err = os.Stat(swaggerJSON)
	require.NoError(t, err, "swagger.json should exist")

	// 检查 swagger.yaml 是否存在
	swaggerYAML := filepath.Join(docsDir, "swagger.yaml")
	_, err = os.Stat(swaggerYAML)
	require.NoError(t, err, "swagger.yaml should exist")
}

// TestSwaggerGenerate_ValidJSON 测试 Swagger JSON 格式是否有效
func TestSwaggerGenerate_ValidJSON(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	assert.NoError(t, err, "swagger.json should be valid JSON")

	// 验证必需的字段
	assert.Equal(t, "2.0", swaggerDoc["swagger"], "swagger version should be 2.0")
	assert.NotNil(t, swaggerDoc["info"], "info should exist")
	assert.NotNil(t, swaggerDoc["paths"], "paths should exist")
}

// TestSwaggerGenerate_ContainsEndpoints 测试 Swagger 文档是否包含 API 端点
func TestSwaggerGenerate_ContainsEndpoints(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	paths, ok := swaggerDoc["paths"].(map[string]interface{})
	require.True(t, ok, "paths should be a map")

	// 验证包含模板管理端点
	assert.Contains(t, paths, "/templates", "should contain /templates endpoint")
	assert.Contains(t, paths, "/templates/{id}", "should contain /templates/{id} endpoint")

	// 验证包含任务管理端点
	assert.Contains(t, paths, "/tasks", "should contain /tasks endpoint")
	assert.Contains(t, paths, "/tasks/{id}", "should contain /tasks/{id} endpoint")
}


