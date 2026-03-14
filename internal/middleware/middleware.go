package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/smap-hcmut/shared-libs/go/response"
)

func (m Middleware) InternalAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Read internal API key from X-Internal-Key header
		// This is used for service-to-service communication
		tokenString := c.GetHeader("X-Internal-Key")

		if tokenString == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if tokenString != m.InternalKey {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Next()
	}
}
