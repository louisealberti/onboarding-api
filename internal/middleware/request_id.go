package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const RequestIDHeader = "X-Request-ID"

// RequestID generates a unique ID for each request and attaches it to:
// - the request context (for use in logs)
// - the response header (so clients can reference it when reporting errors)
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader(RequestIDHeader)
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(RequestIDHeader, requestID)
		c.Header(RequestIDHeader, requestID)

		c.Next()
	}
}
