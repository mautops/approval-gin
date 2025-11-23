package tests

import (
	"os"
	"path/filepath"
	"testing"
)

// TestProjectStructure 验证项目目录结构
func TestProjectStructure(t *testing.T) {
	projectRoot := ".."
	
	requiredDirs := []string{
		"cmd",
		"internal",
		"tests",
		"docs",
		"migrations",
	}
	
	for _, dir := range requiredDirs {
		dirPath := filepath.Join(projectRoot, dir)
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Errorf("Required directory %s does not exist: %v", dir, err)
			continue
		}
		if !info.IsDir() {
			t.Errorf("%s exists but is not a directory", dir)
		}
	}
}

// TestProjectFiles 验证必要的项目文件存在
func TestProjectFiles(t *testing.T) {
	projectRoot := ".."
	
	requiredFiles := []string{
		"go.mod",
		"go.sum",
		"README.md",
		"main.go",
	}
	
	for _, file := range requiredFiles {
		filePath := filepath.Join(projectRoot, file)
		info, err := os.Stat(filePath)
		if err != nil {
			t.Errorf("Required file %s does not exist: %v", file, err)
			continue
		}
		if info.IsDir() {
			t.Errorf("%s exists but is a directory, not a file", file)
		}
	}
}

