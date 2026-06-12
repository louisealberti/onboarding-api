package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/louisealberti/onboarding-api/internal/repository"
)

const IdempotencyKeyHeader = "Idempotency-Key"

// Idempotency intercepts POST requests carrying an Idempotency-Key header.
// If the key was already processed (within 24h), it replays the original response.
// If not, it processes the request normally and stores the result for future replays.
//
// Usage: apply only to POST routes that create resources.
func Idempotency(repo *repository.IdempotencyRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := c.GetHeader(IdempotencyKeyHeader)

		// No key provided — proceed normally, idempotency is opt-in
		if key == "" {
			c.Next()
			return
		}

		// Check if we've seen this key before
		record, err := repo.Get(c.Request.Context(), key)
		if err == nil {
			// Key exists and is not expired — replay the original response
			c.Header(IdempotencyKeyHeader, key)
			c.Header("X-Idempotency-Replayed", "true")
			c.Data(record.StatusCode, "application/json", record.Response)
			c.Abort()
			return
		}

		// Key not found — capture the response so we can store it
		writer := &responseCapture{ResponseWriter: c.Writer, body: &bytes.Buffer{}}
		c.Writer = writer

		c.Next()

		// Only store successful creations (2xx)
		if writer.status >= 200 && writer.status < 300 {
			var raw json.RawMessage = writer.body.Bytes()
			_ = repo.Save(c.Request.Context(), key, writer.status, raw)
		}

		c.Header(IdempotencyKeyHeader, key)
	}
}

// responseCapture wraps gin.ResponseWriter to capture the response body and status.
type responseCapture struct {
	gin.ResponseWriter
	body   *bytes.Buffer
	status int
}

func (r *responseCapture) Write(b []byte) (int, error) {
	r.body.Write(b)
	return r.ResponseWriter.Write(b)
}

func (r *responseCapture) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *responseCapture) WriteHeaderNow() {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	r.ResponseWriter.WriteHeaderNow()
}