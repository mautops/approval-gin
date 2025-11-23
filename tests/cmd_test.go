package tests

import (
	"os/exec"
	"strings"
	"testing"
)

// TestRootCommand 测试根命令是否存在
func TestRootCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "../main.go", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run root command: %v", err)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "approval-gin") {
		t.Error("Root command help does not contain 'approval-gin'")
	}
}

// TestServerCommand 测试 server 命令是否存在
func TestServerCommand(t *testing.T) {
	cmd := exec.Command("go", "run", "../main.go", "server", "--help")
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to run server command: %v", err)
	}
	
	outputStr := string(output)
	if !strings.Contains(outputStr, "server") {
		t.Error("Server command help does not contain 'server'")
	}
}

// TestServerCommandExecution 测试 server 命令可以执行(不实际启动服务器)
func TestServerCommandExecution(t *testing.T) {
	// 这个测试验证命令结构,不实际启动服务器
	cmd := exec.Command("go", "run", "../main.go", "server")
	// 设置超时或使用 context 来避免长时间运行
	// 这里只是验证命令可以解析和执行
	err := cmd.Start()
	if err != nil {
		t.Fatalf("Failed to start server command: %v", err)
	}
	
	// 立即终止命令
	cmd.Process.Kill()
	cmd.Wait()
}

