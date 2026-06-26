package middleware

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/louisealberti/onboarding-api/internal/metrics"
)

// Metrics records HTTP request count and latency for every request,
// labeled by method, route, and status code. Register alongside Logger —
// both read from the same request lifecycle but serve different purposes
// (structured logs for debugging a specific request, metrics for
// aggregate trends over time).
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		// c.FullPath() returns the route pattern (e.g. "/v1/customers/:id"),
		// not the literal request path — using the pattern keeps cardinality
		// bounded regardless of how many distinct customer IDs are requested.
		// It's empty for unmatched routes (404s), which would otherwise
		// create one label series per garbage URL ever requested; group
		// those under "unmatched" instead.
		path := c.FullPath()
		if path == "" {
			path = "unmatched"
		}

		status := strconv.Itoa(c.Writer.Status())

		metrics.HTTPRequestsTotal.WithLabelValues(c.Request.Method, path, status).Inc()
		metrics.HTTPRequestDuration.WithLabelValues(c.Request.Method, path).Observe(time.Since(start).Seconds())
	}
}
