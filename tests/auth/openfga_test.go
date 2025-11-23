package auth_test

import (
	"context"
	"testing"

	"github.com/mautops/approval-gin/internal/auth"
)

// TestOpenFGAClient_New 测试创建 OpenFGA 客户端
func TestOpenFGAClient_New(t *testing.T) {
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	
	client, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	// 注意: 完整的测试需要真实的 OpenFGA 服务器
	// 这里只验证方法存在且可调用
	_ = err
	_ = client
}

// TestOpenFGAClient_CheckPermission 测试权限检查
func TestOpenFGAClient_CheckPermission(t *testing.T) {
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	
	client, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		t.Skip("OpenFGA client creation failed, skipping test")
		return
	}
	
	ctx := context.Background()
	allowed, err := client.CheckPermission(ctx, "user-001", "viewer", "template", "tpl-001")
	// 注意: 完整的测试需要真实的 OpenFGA 服务器和权限模型
	// 这里只验证方法存在且可调用
	_ = err
	_ = allowed
}

