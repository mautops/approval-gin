package api_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestInputValidation_TemplateName_XSS 测试模板名称 XSS 防护
func TestInputValidation_TemplateName_XSS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 使用验证工具验证输入
		if err := utils.ValidateTemplateName(req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 清理输入
		req.Name, _ = utils.TrimAndValidate(req.Name, 255)
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试 XSS 攻击输入
	xssPayload := `<script>alert('XSS')</script>`
	body, _ := json.Marshal(map[string]string{"name": xssPayload})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证输入被清理或拒绝
	assert.NotEqual(t, http.StatusOK, w.Code, "Should reject or sanitize XSS input")
}

// TestInputValidation_TemplateName_SQLInjection 测试模板名称 SQL 注入防护
func TestInputValidation_TemplateName_SQLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 使用验证工具验证输入
		if err := utils.ValidateTemplateName(req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 清理输入
		req.Name, _ = utils.TrimAndValidate(req.Name, 255)
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试 SQL 注入攻击输入
	sqlPayload := "'; DROP TABLE templates; --"
	body, _ := json.Marshal(map[string]string{"name": sqlPayload})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证输入被清理或拒绝
	// 注意：由于使用参数化查询，SQL 注入应该被防护，但输入验证可以进一步拒绝可疑输入
	assert.NotEqual(t, http.StatusOK, w.Code, "Should reject or sanitize SQL injection input")
}

// TestInputValidation_TemplateName_Length 测试模板名称长度限制
func TestInputValidation_TemplateName_Length(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required,max=255"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试超长输入
	longName := string(make([]byte, 300)) // 300 字节，超过 255 字符限制
	body, _ := json.Marshal(map[string]string{"name": longName})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证超长输入被拒绝
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject input exceeding length limit")
}

// TestInputValidation_TaskID_Format 测试任务 ID 格式验证
func TestInputValidation_TaskID_Format(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.GET("/api/v1/tasks/:id", func(c *gin.Context) {
		id := c.Param("id")
		// 验证 ID 格式（应该只包含字母、数字、连字符）
		// 这里应该验证 ID 格式，拒绝包含特殊字符的输入
		c.JSON(http.StatusOK, gin.H{"id": id})
	})

	// 测试包含特殊字符的 ID
	maliciousID := "../../etc/passwd"
	req := httptest.NewRequest("GET", "/api/v1/tasks/"+maliciousID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证恶意 ID 被拒绝或清理
	// 注意：Gin 的路由参数已经进行了一定程度的清理，但可以进一步验证
	assert.NotEqual(t, http.StatusOK, w.Code, "Should reject or sanitize malicious ID")
}

// TestInputValidation_JSON_Malformed 测试恶意 JSON 输入
func TestInputValidation_JSON_Malformed(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试恶意 JSON 输入
	malformedJSON := `{"name": "test", "extra": }` // 无效的 JSON
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBufferString(malformedJSON))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证恶意 JSON 被拒绝
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject malformed JSON")
}

// TestInputValidation_EmptyString 测试空字符串输入
func TestInputValidation_EmptyString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试空字符串输入
	body, _ := json.Marshal(map[string]string{"name": ""})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证空字符串被拒绝
	assert.Equal(t, http.StatusBadRequest, w.Code, "Should reject empty string")
}

// TestInputValidation_WhitespaceOnly 测试仅包含空白字符的输入
func TestInputValidation_WhitespaceOnly(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.POST("/api/v1/templates", func(c *gin.Context) {
		var req struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 使用验证工具验证输入
		if err := utils.ValidateTemplateName(req.Name); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		// 清理输入
		req.Name, _ = utils.TrimAndValidate(req.Name, 255)
		c.JSON(http.StatusOK, gin.H{"name": req.Name})
	})

	// 测试仅包含空白字符的输入
	body, _ := json.Marshal(map[string]string{"name": "   \t\n  "})
	req := httptest.NewRequest("POST", "/api/v1/templates", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 验证仅包含空白字符的输入被清理或拒绝
	// 注意：这取决于验证逻辑，可以清理空白字符或拒绝
	assert.NotEqual(t, http.StatusOK, w.Code, "Should reject or sanitize whitespace-only input")
}

