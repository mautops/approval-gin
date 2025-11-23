package auth_test

import (
	"context"
	"testing"
	"time"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestPermissionCache_GetSet 测试权限缓存的基本操作
func TestPermissionCache_GetSet(t *testing.T) {
	ttl := 5 * time.Minute
	cache := auth.NewPermissionCache(ttl)
	
	// 设置缓存
	key := "user:user-001:viewer:template:tpl-001"
	cache.Set(key, true)
	
	// 获取缓存
	value, found := cache.Get(key)
	assert.True(t, found)
	assert.True(t, value)
	
	// 获取不存在的 key
	_, found = cache.Get("non-existent-key")
	assert.False(t, found)
}

// TestPermissionCache_Expiration 测试缓存过期
func TestPermissionCache_Expiration(t *testing.T) {
	ttl := 100 * time.Millisecond
	cache := auth.NewPermissionCache(ttl)
	
	key := "user:user-001:viewer:template:tpl-001"
	cache.Set(key, true)
	
	// 立即获取应该存在
	_, found := cache.Get(key)
	assert.True(t, found)
	
	// 等待过期
	time.Sleep(150 * time.Millisecond)
	
	// 过期后应该不存在
	_, found = cache.Get(key)
	assert.False(t, found)
}

// TestCachedOpenFGAClient 测试带缓存的 OpenFGA 客户端
func TestCachedOpenFGAClient(t *testing.T) {
	apiURL := "http://localhost:8080"
	storeID := "test-store"
	modelID := "test-model"
	fgaClient, err := auth.NewOpenFGAClient(apiURL, storeID, modelID)
	if err != nil {
		t.Skip("OpenFGA client creation failed, skipping test")
		return
	}
	
	ttl := 5 * time.Minute
	cache := auth.NewPermissionCache(ttl)
	cachedClient := auth.NewCachedOpenFGAClient(fgaClient, cache)
	
	ctx := context.Background()
	// 测试：第一次调用应该查询 OpenFGA
	_, err = cachedClient.CheckPermission(ctx, "user-001", "viewer", "template", "tpl-001")
	// 注意: 完整的测试需要真实的 OpenFGA 服务器
	// 这里只验证方法存在且可调用
	_ = err
}

