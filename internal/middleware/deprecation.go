package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
)

// Deprecated marks a route group as deprecated by adding standard
// RFC 8594 headers. Apply to a route group when a newer version exists.
//
// Usage:
//
//	v1 := r.Group("/v1")
//	v1.Use(middleware.Deprecated("2027-01-01", "https://api.example.com/v2"))
func Deprecated(sunsetDate string, successor string) gin.HandlerFunc {
	// Validate the date at startup, not per-request
	_, err := time.Parse("2006-01-02", sunsetDate)
	if err != nil {
		panic("middleware.Deprecated: invalid sunsetDate format, expected YYYY-MM-DD")
	}

	return func(c *gin.Context) {
		c.Header("Deprecation", "true")
		c.Header("Sunset", sunsetDate)
		if successor != "" {
			c.Header("Link", "<"+successor+">; rel=\"successor-version\"")
		}
		c.Next()
	}
}
