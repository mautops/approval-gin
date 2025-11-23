package auth_test

import (
	"testing"

	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
)

// TestKeycloakTokenValidator_New 测试创建 Keycloak Token 验证器
func TestKeycloakTokenValidator_New(t *testing.T) {
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	
	assert.NotNil(t, validator)
	assert.Equal(t, issuer, validator.Issuer())
}

// TestKeycloakTokenValidator_ValidateToken 测试验证 Token
func TestKeycloakTokenValidator_ValidateToken(t *testing.T) {
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	
	// 测试：无效的 token
	invalidToken := "invalid.token.here"
	claims, err := validator.ValidateToken(invalidToken)
	assert.Error(t, err)
	assert.Nil(t, claims)
	
	// 注意: 完整的 token 验证需要真实的 Keycloak 服务器和有效的 token
	// 这里只验证方法存在且可调用
}

// TestKeycloakTokenValidator_GetPublicKey 测试获取公钥
func TestKeycloakTokenValidator_GetPublicKey(t *testing.T) {
	issuer := "https://keycloak.example.com/realms/test"
	validator := auth.NewKeycloakTokenValidator(issuer)
	
	// 测试：获取不存在的 key ID
	_, err := validator.GetPublicKey("non-existent-kid")
	assert.Error(t, err)
	
	// 注意: 完整的公钥获取需要真实的 Keycloak JWKS endpoint
	// 这里只验证方法存在且可调用
}

