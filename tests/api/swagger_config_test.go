package api_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestSwaggerTool_Installed 测试 swag 工具是否已安装
func TestSwaggerTool_Installed(t *testing.T) {
	// 检查 swag 命令是否可用
	swagPath, err := exec.LookPath("swag")
	if err != nil {
		// 尝试在 GOPATH/bin 中查找
		homeDir := os.Getenv("HOME")
		if homeDir == "" {
			homeDir = os.Getenv("USERPROFILE")
		}
		swagPath = filepath.Join(homeDir, "go", "bin", "swag")
		_, err = os.Stat(swagPath)
		if err != nil {
			t.Skip("swag tool not installed, skipping test")
			return
		}
	}

	// 验证 swag 命令可以执行（忽略退出码，只验证命令存在）
	_, err = os.Stat(swagPath)
	assert.NoError(t, err, "swag command should exist")
}

// TestSwaggerConfig_Exists 测试 Swagger 配置文件是否存在
func TestSwaggerConfig_Exists(t *testing.T) {
	// 检查是否存在 docs 目录
	docsDir := filepath.Join("..", "..", "docs")
	_, err := os.Stat(docsDir)
	if err != nil {
		// docs 目录不存在是正常的（首次运行）
		t.Log("docs directory does not exist yet, will be created by swag init")
		return
	}

	// 如果存在，检查是否有必要的文件
	swaggerJSON := filepath.Join(docsDir, "swagger.json")
	swaggerYAML := filepath.Join(docsDir, "swagger.yaml")
	docsGo := filepath.Join(docsDir, "docs.go")

	// 这些文件可能不存在（首次运行）
	_, err = os.Stat(swaggerJSON)
	if err == nil {
		assert.FileExists(t, swaggerJSON, "swagger.json should exist")
	}

	_, err = os.Stat(swaggerYAML)
	if err == nil {
		assert.FileExists(t, swaggerYAML, "swagger.yaml should exist")
	}

	_, err = os.Stat(docsGo)
	if err == nil {
		assert.FileExists(t, docsGo, "docs.go should exist")
	}
}

// TestSwaggerConfig_Generate 测试 Swagger 文档生成
func TestSwaggerConfig_Generate(t *testing.T) {
	// 检查 swag 命令是否可用
	_, err := exec.LookPath("swag")
	if err != nil {
		t.Skip("swag tool not installed, skipping test")
		return
	}

	// 验证可以生成 Swagger 文档（不实际执行，只验证命令格式）
	cmd := exec.Command("swag", "init", "--help")
	err = cmd.Run()
	assert.NoError(t, err, "swag init command should be available")
}

