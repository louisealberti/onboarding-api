package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestAcceptance_RateLimit(t *testing.T) {
	db := setupDB(t)

	// Override startServer with a very restrictive rate limiter for this test
	srv := startServerWithRateLimit(t, db, rate.Limit(1), 1)

	t.Run("primeira request passa", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("requests em excesso retornam 429", func(t *testing.T) {
		// Burst já foi consumido — próximas requests devem ser limitadas
		var got429 bool
		for i := 0; i < 20; i++ {
			resp := apiGet(t, srv, "/v1/customers")
			resp.Body.Close()
			if resp.StatusCode == http.StatusTooManyRequests {
				got429 = true
				break
			}
		}
		assert.True(t, got429, "esperava receber 429 após exceder o rate limit")
	})
}
