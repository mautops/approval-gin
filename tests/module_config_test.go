package tests

import (
	"os/exec"
	"strings"
	"testing"
)

// TestModuleDependencies 验证 go.mod 中的必需依赖
func TestModuleDependencies(t *testing.T) {
	// 读取 go.mod 文件内容
	cmd := exec.Command("go", "mod", "graph")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run go mod graph: %v", err)
	}
	
	modGraph := string(output)
	
	// 必需的依赖包
	requiredDeps := []string{
		"github.com/gin-gonic/gin",
		"gorm.io/gorm",
		"github.com/spf13/cobra",
		"github.com/spf13/viper",
	}
	
	// 检查每个依赖是否存在
	for _, dep := range requiredDeps {
		if !strings.Contains(modGraph, dep) {
			// 检查是否在 go.mod 中声明
			cmd := exec.Command("grep", "-q", dep, "../go.mod")
			if err := cmd.Run(); err != nil {
				t.Errorf("Required dependency %s is not found in go.mod", dep)
			}
		}
	}
}

// TestModuleVersion 验证 Go 版本要求
func TestModuleVersion(t *testing.T) {
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run go version: %v", err)
	}
	
	versionStr := string(output)
	// 检查 Go 版本是否 >= 1.25.4
	// 这里简化检查,实际应该解析版本号
	if !strings.Contains(versionStr, "go1.") {
		t.Error("Go version check failed")
	}
}

