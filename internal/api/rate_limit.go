package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// RateLimitMiddleware 限流中间件
func RateLimitMiddleware(rps float64, burst int) gin.HandlerFunc {
	limiter := rate.NewLimiter(rate.Limit(rps), burst)

	return func(c *gin.Context) {
		if !limiter.Allow() {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Code:    429,
				Message: "too many requests",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}


