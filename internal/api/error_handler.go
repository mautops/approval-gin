package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// APIError API 错误
type APIError struct {
	Code    int
	Message string
	Detail  string
}

func (e *APIError) Error() string {
	return e.Message
}

// ErrorHandlerMiddleware 错误处理中间件
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()

			var apiErr *APIError
			if errors.As(err, &apiErr) {
				Error(c, apiErr.Code, apiErr.Message, apiErr.Detail)
			} else {
				Error(c, http.StatusInternalServerError, "internal server error", err.Error())
			}
		}
	}
}

// WrapError 包装错误
func WrapError(err error, code int, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
		Detail:  err.Error(),
	}
}

