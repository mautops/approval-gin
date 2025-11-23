package auth

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// PermissionCache 权限缓存
type PermissionCache struct {
	cache *sync.Map
	ttl   time.Duration
}

// cacheEntry 缓存条目
type cacheEntry struct {
	value     bool
	expiresAt time.Time
}

// NewPermissionCache 创建权限缓存
func NewPermissionCache(ttl time.Duration) *PermissionCache {
	return &PermissionCache{
		cache: &sync.Map{},
		ttl:   ttl,
	}
}

// Get 获取缓存
func (c *PermissionCache) Get(key string) (bool, bool) {
	val, found := c.cache.Load(key)
	if !found {
		return false, false
	}

	entry := val.(*cacheEntry)
	if time.Now().After(entry.expiresAt) {
		// 已过期，删除
		c.cache.Delete(key)
		return false, false
	}

	return entry.value, true
}

// Set 设置缓存
func (c *PermissionCache) Set(key string, value bool) {
	entry := &cacheEntry{
		value:     value,
		expiresAt: time.Now().Add(c.ttl),
	}
	c.cache.Store(key, entry)
}

// Clear 清空缓存
func (c *PermissionCache) Clear() {
	c.cache.Range(func(key, value interface{}) bool {
		c.cache.Delete(key)
		return true
	})
}

// CachedOpenFGAClient 带缓存的 OpenFGA 客户端
type CachedOpenFGAClient struct {
	client *OpenFGAClient
	cache  *PermissionCache
}

// NewCachedOpenFGAClient 创建带缓存的 OpenFGA 客户端
func NewCachedOpenFGAClient(client *OpenFGAClient, cache *PermissionCache) *CachedOpenFGAClient {
	return &CachedOpenFGAClient{
		client: client,
		cache:  cache,
	}
}

// CheckPermission 检查权限（带缓存）
func (c *CachedOpenFGAClient) CheckPermission(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) (bool, error) {
	// 生成缓存 key
	cacheKey := fmt.Sprintf("user:%s:%s:%s:%s", userID, relation, objectType, objectID)

	// 从缓存获取
	if value, found := c.cache.Get(cacheKey); found {
		return value, nil
	}

	// 缓存未命中，查询 OpenFGA
	allowed, err := c.client.CheckPermission(ctx, userID, relation, objectType, objectID)
	if err != nil {
		return false, err
	}

	// 写入缓存
	c.cache.Set(cacheKey, allowed)

	return allowed, nil
}

// SetRelation 设置权限关系（清除相关缓存）
func (c *CachedOpenFGAClient) SetRelation(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) error {
	err := c.client.SetRelation(ctx, userID, relation, objectType, objectID)
	if err != nil {
		return err
	}

	// 清除相关缓存
	cacheKey := fmt.Sprintf("user:%s:%s:%s:%s", userID, relation, objectType, objectID)
	c.cache.cache.Delete(cacheKey)

	return nil
}

// DeleteRelation 删除权限关系（清除相关缓存）
func (c *CachedOpenFGAClient) DeleteRelation(
	ctx context.Context,
	userID string,
	relation string,
	objectType string,
	objectID string,
) error {
	err := c.client.DeleteRelation(ctx, userID, relation, objectType, objectID)
	if err != nil {
		return err
	}

	// 清除相关缓存
	cacheKey := fmt.Sprintf("user:%s:%s:%s:%s", userID, relation, objectType, objectID)
	c.cache.cache.Delete(cacheKey)

	return nil
}


