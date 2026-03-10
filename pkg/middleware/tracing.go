package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type contextKey string

const TraceIDKey contextKey = "trace_id"

// Tracing automatically reads X-Trace-Id from headers or generates a new one.
// It stores it into the Gin context, the Request context, and writes it to the response header.
func Tracing() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = uuid.New().String()
		}

		// Set in Gin Context for handlers
		c.Set("trace_id", traceID)

		// Set in request.Context() so standard library and loggers can see it
		ctx := context.WithValue(c.Request.Context(), TraceIDKey, traceID)
		c.Request = c.Request.WithContext(ctx)

		// Always return it to the client/downstream
		c.Header("X-Trace-Id", traceID)

		c.Next()
	}
}
