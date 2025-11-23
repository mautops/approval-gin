package api

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFConfig CSRF 配置
type CSRFConfig struct {
	SecretKey      string        // 密钥
	TokenLength    int           // Token 长度
	TokenTTL       time.Duration // Token 有效期
	HeaderName     string        // Token 请求头名称
	CookieName     string        // Cookie 名称
	CookieSecure   bool          // Cookie 是否仅 HTTPS
	CookieSameSite http.SameSite // Cookie SameSite 属性
}

// DefaultCSRFConfig 默认 CSRF 配置
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		SecretKey:      generateRandomKey(32),
		TokenLength:    32,
		TokenTTL:       24 * time.Hour,
		HeaderName:     "X-CSRF-Token",
		CookieName:     "csrf_token",
		CookieSecure:   false, // 开发环境默认 false
		CookieSameSite: http.SameSiteStrictMode,
	}
}

// CSRFStore CSRF Token 存储
type CSRFStore struct {
	tokens map[string]*csrfToken
	mu     sync.RWMutex
	config *CSRFConfig
}

// csrfToken CSRF Token 信息
type csrfToken struct {
	token     string
	expiresAt time.Time
}

// NewCSRFStore 创建 CSRF 存储
func NewCSRFStore(config *CSRFConfig) *CSRFStore {
	store := &CSRFStore{
		tokens: make(map[string]*csrfToken),
		config: config,
	}

	// 启动清理过期 token 的 goroutine
	go store.cleanupExpiredTokens()

	return store
}

// GenerateToken 生成 CSRF Token
func (s *CSRFStore) GenerateToken() (string, error) {
	token, err := generateRandomToken(s.config.TokenLength)
	if err != nil {
		return "", err
	}

	s.mu.Lock()
	s.tokens[token] = &csrfToken{
		token:     token,
		expiresAt: time.Now().Add(s.config.TokenTTL),
	}
	s.mu.Unlock()

	return token, nil
}

// ValidateToken 验证 CSRF Token
func (s *CSRFStore) ValidateToken(token string) bool {
	if token == "" {
		return false
	}

	s.mu.RLock()
	csrfToken, exists := s.tokens[token]
	s.mu.RUnlock()

	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(csrfToken.expiresAt) {
		s.mu.Lock()
		delete(s.tokens, token)
		s.mu.Unlock()
		return false
	}

	return true
}

// cleanupExpiredTokens 清理过期的 token
func (s *CSRFStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for token, csrfToken := range s.tokens {
			if now.After(csrfToken.expiresAt) {
				delete(s.tokens, token)
			}
		}
		s.mu.Unlock()
	}
}

// CSRFMiddleware CSRF 保护中间件
func CSRFMiddleware(config *CSRFConfig) gin.HandlerFunc {
	if config == nil {
		config = DefaultCSRFConfig()
	}

	store := NewCSRFStore(config)

	return func(c *gin.Context) {
		// GET、HEAD、OPTIONS 请求不需要 CSRF 保护
		if c.Request.Method == http.MethodGet ||
			c.Request.Method == http.MethodHead ||
			c.Request.Method == http.MethodOptions {
			// 将 store 存储到上下文，供后续使用（如生成新 token）
			c.Set("csrf_store", store)
			c.Next()
			return
		}

		// 从请求头获取 token
		token := c.GetHeader(config.HeaderName)

		// 如果请求头没有，尝试从 Cookie 获取
		if token == "" {
			cookie, err := c.Cookie(config.CookieName)
			if err == nil {
				token = cookie
			}
		}

		// 验证 token
		if !store.ValidateToken(token) {
			c.JSON(http.StatusForbidden, gin.H{
				"code":    403,
				"message": "invalid csrf token",
			})
			c.Abort()
			return
		}

		// 将 store 存储到上下文，供后续使用（如生成新 token）
		c.Set("csrf_store", store)
		c.Next()
	}
}

// GetCSRFToken 获取 CSRF Token（用于返回给客户端）
func GetCSRFToken(c *gin.Context) (string, error) {
	store, exists := c.Get("csrf_store")
	if !exists {
		// 如果没有 store，创建一个临时的
		config := DefaultCSRFConfig()
		store = NewCSRFStore(config)
	}

	csrfStore, ok := store.(*CSRFStore)
	if !ok {
		config := DefaultCSRFConfig()
		csrfStore = NewCSRFStore(config)
	}

	token, err := csrfStore.GenerateToken()
	if err != nil {
		return "", err
	}

	// 设置 Cookie
	config := csrfStore.config
	c.SetCookie(
		config.CookieName,
		token,
		int(config.TokenTTL.Seconds()),
		"/",
		"",
		config.CookieSecure,
		true, // HttpOnly
	)

	return token, nil
}

// generateRandomToken 生成随机 token
func generateRandomToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// generateRandomKey 生成随机密钥
func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}


