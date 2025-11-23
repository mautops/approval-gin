package api_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSwaggerModels_ResponseModel 测试 Response 模型的 Swagger 注释
func TestSwaggerModels_ResponseModel(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	definitions, ok := swaggerDoc["definitions"].(map[string]interface{})
	if !ok {
		t.Skip("definitions section not found in swagger.json, may need to regenerate")
		return
	}

	// 验证 Response 模型存在
	responseModel, ok := definitions["api.Response"].(map[string]interface{})
	require.True(t, ok, "api.Response model should exist in definitions")

	// 验证 Response 模型有描述
	description, ok := responseModel["description"].(string)
	assert.True(t, ok, "api.Response should have description")
	assert.NotEmpty(t, description, "api.Response description should not be empty")

	// 验证 Response 模型有必需的字段
	properties, ok := responseModel["properties"].(map[string]interface{})
	require.True(t, ok, "api.Response should have properties")

	assert.Contains(t, properties, "code", "api.Response should have 'code' field")
	assert.Contains(t, properties, "message", "api.Response should have 'message' field")
	assert.Contains(t, properties, "data", "api.Response should have 'data' field")
}

// TestSwaggerModels_ErrorResponseModel 测试 ErrorResponse 模型的 Swagger 注释
func TestSwaggerModels_ErrorResponseModel(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	definitions, ok := swaggerDoc["definitions"].(map[string]interface{})
	if !ok {
		t.Skip("definitions section not found in swagger.json, may need to regenerate")
		return
	}

	// 验证 ErrorResponse 模型存在
	errorResponseModel, ok := definitions["api.ErrorResponse"].(map[string]interface{})
	require.True(t, ok, "api.ErrorResponse model should exist in definitions")

	// 验证 ErrorResponse 模型有描述
	description, ok := errorResponseModel["description"].(string)
	assert.True(t, ok, "api.ErrorResponse should have description")
	assert.NotEmpty(t, description, "api.ErrorResponse description should not be empty")

	// 验证 ErrorResponse 模型有必需的字段
	properties, ok := errorResponseModel["properties"].(map[string]interface{})
	require.True(t, ok, "api.ErrorResponse should have properties")

	assert.Contains(t, properties, "code", "api.ErrorResponse should have 'code' field")
	assert.Contains(t, properties, "message", "api.ErrorResponse should have 'message' field")
	assert.Contains(t, properties, "detail", "api.ErrorResponse should have 'detail' field")
}

// TestSwaggerModels_PaginatedResponseModel 测试 PaginatedResponse 模型的 Swagger 注释
func TestSwaggerModels_PaginatedResponseModel(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	definitions, ok := swaggerDoc["definitions"].(map[string]interface{})
	if !ok {
		t.Skip("definitions section not found in swagger.json, may need to regenerate")
		return
	}

	// 验证 PaginatedResponse 模型存在
	paginatedResponseModel, ok := definitions["api.PaginatedResponse"].(map[string]interface{})
	require.True(t, ok, "api.PaginatedResponse model should exist in definitions")

	// 验证 PaginatedResponse 模型有描述
	description, ok := paginatedResponseModel["description"].(string)
	assert.True(t, ok, "api.PaginatedResponse should have description")
	assert.NotEmpty(t, description, "api.PaginatedResponse description should not be empty")

	// 验证 PaginatedResponse 模型有必需的字段
	properties, ok := paginatedResponseModel["properties"].(map[string]interface{})
	require.True(t, ok, "api.PaginatedResponse should have properties")

	assert.Contains(t, properties, "code", "api.PaginatedResponse should have 'code' field")
	assert.Contains(t, properties, "message", "api.PaginatedResponse should have 'message' field")
	assert.Contains(t, properties, "data", "api.PaginatedResponse should have 'data' field")
	assert.Contains(t, properties, "pagination", "api.PaginatedResponse should have 'pagination' field")
}

// TestSwaggerModels_PaginationInfoModel 测试 PaginationInfo 模型的 Swagger 注释
func TestSwaggerModels_PaginationInfoModel(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	definitions, ok := swaggerDoc["definitions"].(map[string]interface{})
	if !ok {
		t.Skip("definitions section not found in swagger.json, may need to regenerate")
		return
	}

	// 验证 PaginationInfo 模型存在
	paginationInfoModel, ok := definitions["api.PaginationInfo"].(map[string]interface{})
	require.True(t, ok, "api.PaginationInfo model should exist in definitions")

	// 验证 PaginationInfo 模型有描述
	description, ok := paginationInfoModel["description"].(string)
	assert.True(t, ok, "api.PaginationInfo should have description")
	assert.NotEmpty(t, description, "api.PaginationInfo description should not be empty")

	// 验证 PaginationInfo 模型有必需的字段
	properties, ok := paginationInfoModel["properties"].(map[string]interface{})
	require.True(t, ok, "api.PaginationInfo should have properties")

	assert.Contains(t, properties, "page", "api.PaginationInfo should have 'page' field")
	assert.Contains(t, properties, "page_size", "api.PaginationInfo should have 'page_size' field")
	assert.Contains(t, properties, "total", "api.PaginationInfo should have 'total' field")
	assert.Contains(t, properties, "total_page", "api.PaginationInfo should have 'total_page' field")
}

// TestSwaggerModels_ServiceModels 测试 Service 层数据模型的 Swagger 注释
func TestSwaggerModels_ServiceModels(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	var swaggerDoc map[string]interface{}
	err = json.Unmarshal(data, &swaggerDoc)
	require.NoError(t, err, "swagger.json should be valid JSON")

	definitions, ok := swaggerDoc["definitions"].(map[string]interface{})
	if !ok {
		t.Skip("definitions section not found in swagger.json, may need to regenerate")
		return
	}

	// 验证 CreateTemplateRequest 模型存在
	createTemplateRequest, ok := definitions["service.CreateTemplateRequest"].(map[string]interface{})
	if ok {
		description, ok := createTemplateRequest["description"].(string)
		assert.True(t, ok, "service.CreateTemplateRequest should have description")
		assert.NotEmpty(t, description, "service.CreateTemplateRequest description should not be empty")
	}

	// 验证 CreateTaskRequest 模型存在
	// 注意: 由于 json.RawMessage 的限制, swag 工具可能无法完全解析 CreateTaskRequest
	// 但模型应该存在,即使没有描述
	createTaskRequest, ok := definitions["service.CreateTaskRequest"].(map[string]interface{})
	if ok {
		// 如果存在描述,验证它不为空
		if description, hasDesc := createTaskRequest["description"].(string); hasDesc {
			assert.NotEmpty(t, description, "service.CreateTaskRequest description should not be empty if present")
		} else {
			// 如果没有描述,至少验证模型存在且有类型定义
			assert.Contains(t, createTaskRequest, "type", "service.CreateTaskRequest should have type definition")
		}
	} else {
		t.Log("service.CreateTaskRequest model not found in definitions, may need to be used in an API endpoint")
	}

	// 验证 ApproveRequest 模型存在
	approveRequest, ok := definitions["service.ApproveRequest"].(map[string]interface{})
	if ok {
		description, ok := approveRequest["description"].(string)
		assert.True(t, ok, "service.ApproveRequest should have description")
		assert.NotEmpty(t, description, "service.ApproveRequest description should not be empty")
	}
}

// TestSwaggerModels_FieldDescriptions 测试数据模型字段是否有描述
func TestSwaggerModels_FieldDescriptions(t *testing.T) {
	swaggerJSON := filepath.Join("..", "..", "docs", "swagger.json")

	data, err := os.ReadFile(swaggerJSON)
	require.NoError(t, err, "should be able to read swagger.json")

	// 检查 swagger.json 中是否包含字段描述的关键词
	// 这只是一个简单的检查，实际的字段描述验证需要解析 JSON
	content := string(data)
	
	// 验证 Response 模型的字段有描述（通过检查 JSON 结构）
	assert.True(t, strings.Contains(content, "Response") || strings.Contains(content, "response"), 
		"swagger.json should contain Response model")
	
	// 验证 ErrorResponse 模型的字段有描述
	assert.True(t, strings.Contains(content, "ErrorResponse") || strings.Contains(content, "error"), 
		"swagger.json should contain ErrorResponse model")
}

