package auth

import (
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// KeycloakClaims Keycloak JWT 声明
type KeycloakClaims struct {
	Sub               string   `json:"sub"`
	Email             string   `json:"email"`
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	RealmAccess       struct {
		Roles []string `json:"roles"`
	} `json:"realm_access"`
	jwt.RegisteredClaims
}

// KeycloakTokenValidator Keycloak Token 验证器
type KeycloakTokenValidator struct {
	issuer    string
	jwksURL   string
	jwksCache *sync.Map
	httpClient *http.Client
}

// NewKeycloakTokenValidator 创建 Keycloak Token 验证器
func NewKeycloakTokenValidator(issuer string) *KeycloakTokenValidator {
	jwksURL := fmt.Sprintf("%s/protocol/openid-connect/certs", issuer)
	return &KeycloakTokenValidator{
		issuer:     issuer,
		jwksURL:    jwksURL,
		jwksCache:  &sync.Map{},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Issuer 返回 Issuer URL
func (v *KeycloakTokenValidator) Issuer() string {
	return v.issuer
}

// ValidateToken 验证 Keycloak JWT Token
func (v *KeycloakTokenValidator) ValidateToken(tokenString string) (*KeycloakClaims, error) {
	// 1. 解析 token (不验证签名)
	token, err := jwt.ParseWithClaims(tokenString, &KeycloakClaims{}, func(token *jwt.Token) (interface{}, error) {
		// 验证签名算法
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return nil, nil // 先返回 nil,稍后获取公钥
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// 2. 获取 token 的 kid (Key ID)
	kid, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("missing kid in token header")
	}

	// 3. 获取公钥
	publicKey, err := v.GetPublicKey(kid)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// 4. 重新解析并验证 token
	token, err = jwt.ParseWithClaims(tokenString, &KeycloakClaims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to validate token: %w", err)
	}

	// 5. 验证 claims
	if claims, ok := token.Claims.(*KeycloakClaims); ok && token.Valid {
		// 验证 issuer
		if claims.Issuer != v.issuer {
			return nil, errors.New("invalid issuer")
		}

		// 验证过期时间
		if claims.ExpiresAt != nil && claims.ExpiresAt.Time.Before(time.Now()) {
			return nil, errors.New("token expired")
		}

		return claims, nil
	}

	return nil, errors.New("invalid token")
}

// GetPublicKey 获取公钥 (从 JWKS 或缓存)
func (v *KeycloakTokenValidator) GetPublicKey(kid string) (interface{}, error) {
	// 从缓存获取
	if cached, ok := v.jwksCache.Load(kid); ok {
		return cached, nil
	}

	// 从 Keycloak 获取 JWKS
	resp, err := v.httpClient.Get(v.jwksURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("JWKS endpoint returned status %d", resp.StatusCode)
	}

	var jwks struct {
		Keys []struct {
			Kid string `json:"kid"`
			Kty string `json:"kty"`
			Use string `json:"use"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return nil, fmt.Errorf("failed to decode JWKS: %w", err)
	}

	// 查找匹配的 key
	for _, key := range jwks.Keys {
		if key.Kid == kid {
			// 解析 RSA 公钥
			publicKey, err := parseRSAPublicKey(key.N, key.E)
			if err != nil {
				return nil, fmt.Errorf("failed to parse RSA public key: %w", err)
			}

			// 缓存公钥
			v.jwksCache.Store(kid, publicKey)
			return publicKey, nil
		}
	}

	return nil, fmt.Errorf("key not found in JWKS: %s", kid)
}

// parseRSAPublicKey 解析 RSA 公钥
func parseRSAPublicKey(nStr, eStr string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode n: %w", err)
	}

	eBytes, err := base64.RawURLEncoding.DecodeString(eStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode e: %w", err)
	}

	n := new(big.Int).SetBytes(nBytes)
	e := int(new(big.Int).SetBytes(eBytes).Int64())

	return &rsa.PublicKey{
		N: n,
		E: e,
	}, nil
}

// KeycloakAuthMiddleware Keycloak JWT 认证中间件
func KeycloakAuthMiddleware(validator *KeycloakTokenValidator) gin.HandlerFunc {
	return func(c *gin.Context) {
		token := c.GetHeader("Authorization")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "missing authorization header",
			})
			c.Abort()
			return
		}

		// 移除 "Bearer " 前缀
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}

		claims, err := validator.ValidateToken(token)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{
				"code":    401,
				"message": "invalid token",
				"detail":  err.Error(),
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", claims.Sub)
		c.Set("username", claims.PreferredUsername)
		c.Set("email", claims.Email)
		c.Set("name", claims.Name)
		c.Set("roles", claims.RealmAccess.Roles)

		c.Next()
	}
}

