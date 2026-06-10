package middleware

import (
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
)

// Logger logs each request as a structured JSON line with method, path,
// status, latency, and the request ID set by the RequestID middleware.
// It must be registered after RequestID so the request ID is available.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		requestID, _ := c.Get(RequestIDHeader)

		logger.Info("request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.FullPath()),
			slog.String("query", c.Request.URL.RawQuery),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("latency", time.Since(start)),
			slog.String("request_id", requestID.(string)),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
