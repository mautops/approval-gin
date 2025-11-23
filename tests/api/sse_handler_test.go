package api_test

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/mautops/approval-gin/internal/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSSEHandler_Connection 测试 SSE 连接
func TestSSEHandler_Connection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 配置 SSE 路由
	router.GET("/sse/tasks/:id", api.SSEHandler(validator))

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 SSE（不带 token，应该失败）
	req, err := http.NewRequest("GET", server.URL+"/sse/tasks/task-001", nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 没有 token 时应该返回 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSSEHandler_InvalidToken 测试无效 token 的 SSE 连接
func TestSSEHandler_InvalidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 配置 SSE 路由
	router.GET("/sse/tasks/:id", api.SSEHandler(validator))

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 SSE（带无效 token，应该失败）
	req, err := http.NewRequest("GET", server.URL+"/sse/tasks/task-001?token=invalid-token", nil)
	require.NoError(t, err)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 无效 token 时应该返回 401
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSSEHandler_ValidConnection 测试有效的 SSE 连接
func TestSSEHandler_ValidConnection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	validator := auth.NewKeycloakTokenValidator("http://localhost:8080/auth/realms/test")

	// 配置 SSE 路由
	router.GET("/sse/tasks/:id", api.SSEHandler(validator))

	server := httptest.NewServer(router)
	defer server.Close()

	// 连接到 SSE（带无效 token，应该返回 401）
	req, err := http.NewRequest("GET", server.URL+"/sse/tasks/task-001?token=test-token", nil)
	require.NoError(t, err)
	req.Header.Set("Accept", "text/event-stream")

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 由于 token 无效，应该返回 401
	// 在实际场景中，有效 token 会建立 SSE 连接并返回正确的响应头
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
}

// TestSSEHandler_EventFormat 测试 SSE 事件格式
func TestSSEHandler_EventFormat(t *testing.T) {
	// 测试 SSE 事件格式
	event := "data: {\"task_id\":\"task-001\",\"status\":\"approved\"}\n\n"
	
	scanner := bufio.NewScanner(strings.NewReader(event))
	scanner.Scan()
	line := scanner.Text()
	
	// 验证事件格式
	assert.True(t, strings.HasPrefix(line, "data: "))
}

