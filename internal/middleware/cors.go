package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// CORS configures Cross-Origin Resource Sharing headers.
// origins is a list of allowed origins; pass []string{"*"} to allow all.
// For production, pass explicit origins (e.g. backoffice domain).
func CORS(origins []string) gin.HandlerFunc {
	allowedOrigins := make(map[string]bool, len(origins))
	allowAll := false
	for _, o := range origins {
		if o == "*" {
			allowAll = true
			break
		}
		allowedOrigins[o] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		if allowAll || allowedOrigins[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID, Idempotency-Key")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		// Preflight request — respond immediately without hitting handlers
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
