package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Response 统一响应格式
// @Description 统一响应格式,包含状态码、消息和数据
type Response struct {
	Code    int         `json:"code" example:"0"`    // 状态码: 0 表示成功,非 0 表示失败
	Message string      `json:"message" example:"success"` // 响应消息
	Data    interface{} `json:"data"`    // 响应数据
}

// ErrorResponse 错误响应格式
// @Description 错误响应格式,包含错误码、错误消息和错误详情
type ErrorResponse struct {
	Code    int    `json:"code" example:"400"`    // 错误码
	Message string `json:"message" example:"invalid request"` // 错误消息
	Detail  string `json:"detail,omitempty" example:"validation failed"`  // 错误详情(可选)
}

// PaginatedResponse 分页响应
// @Description 分页响应格式,包含数据列表和分页信息
type PaginatedResponse struct {
	Code       int         `json:"code" example:"0"`
	Message    string      `json:"message" example:"success"`
	Data       interface{} `json:"data"`    // 数据列表
	Pagination PaginationInfo `json:"pagination"` // 分页信息
}

// PaginationInfo 分页信息
// @Description 分页信息,包含当前页码、每页数量、总记录数和总页数
type PaginationInfo struct {
	Page      int   `json:"page" example:"1"`       // 当前页码
	PageSize  int   `json:"page_size" example:"20"`  // 每页数量
	Total     int64 `json:"total" example:"100"`      // 总记录数
	TotalPage int   `json:"total_page" example:"5"` // 总页数
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

// Error 错误响应
func Error(c *gin.Context, code int, message string, detail string) {
	statusCode := http.StatusInternalServerError
	if code >= 400 && code < 600 {
		statusCode = code
	}
	
	c.JSON(statusCode, ErrorResponse{
		Code:    code,
		Message: message,
		Detail:  detail,
	})
}

// Paginated 分页响应
func Paginated(c *gin.Context, data interface{}, pagination PaginationInfo) {
	c.JSON(http.StatusOK, PaginatedResponse{
		Code:       0,
		Message:    "success",
		Data:       data,
		Pagination: pagination,
	})
}

