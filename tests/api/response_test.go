package api_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/mautops/approval-gin/internal/api"
	"github.com/stretchr/testify/assert"
)

// TestResponseFormat 测试统一响应格式
func TestResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		api.Success(c, gin.H{"message": "test"})
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.Response
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, "success", response.Message)
	assert.NotNil(t, response.Data)
}

// TestErrorResponseFormat 测试错误响应格式
func TestErrorResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		api.Error(c, 400, "invalid request", "missing required field")
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusBadRequest, w.Code)
	
	var response api.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 400, response.Code)
	assert.Equal(t, "invalid request", response.Message)
	assert.Equal(t, "missing required field", response.Detail)
}

// TestPaginatedResponseFormat 测试分页响应格式
func TestPaginatedResponseFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		data := []string{"item1", "item2", "item3"}
		pagination := api.PaginationInfo{
			Page:      1,
			PageSize:  20,
			Total:     100,
			TotalPage: 5,
		}
		api.Paginated(c, data, pagination)
	})
	
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	assert.Equal(t, http.StatusOK, w.Code)
	
	var response api.PaginatedResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, 0, response.Code)
	assert.Equal(t, "success", response.Message)
	assert.NotNil(t, response.Data)
	assert.Equal(t, 1, response.Pagination.Page)
	assert.Equal(t, 20, response.Pagination.PageSize)
	assert.Equal(t, int64(100), response.Pagination.Total)
	assert.Equal(t, 5, response.Pagination.TotalPage)
}


