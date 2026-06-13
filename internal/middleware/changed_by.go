package middleware

import (
	"context"

	"github.com/gin-gonic/gin"
)

const ChangedByHeader = "X-Changed-By"

// ChangedBy reads the X-Changed-By header and injects it into the request context.
// The service layer reads it via ctx.Value("changed_by") to attribute audit log entries.
// When JWT is introduced, this middleware will be replaced by token subject extraction.
func ChangedBy() gin.HandlerFunc {
	return func(c *gin.Context) {
		changedBy := c.GetHeader(ChangedByHeader)
		if changedBy == "" {
			changedBy = "system"
		}
		ctx := context.WithValue(c.Request.Context(), "changed_by", changedBy)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
