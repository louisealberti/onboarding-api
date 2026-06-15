package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// limiterEntry holds a token bucket limiter and the last time it was seen.
// Entries that haven't been accessed in a while are cleaned up periodically.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimiterStore manages per-key token bucket limiters with periodic cleanup.
type RateLimiterStore struct {
	mu       sync.Mutex
	limiters map[string]*limiterEntry
	r        rate.Limit // tokens per second
	b        int        // burst size
}

func NewRateLimiterStore(r rate.Limit, b int) *RateLimiterStore {
	store := &RateLimiterStore{
		limiters: make(map[string]*limiterEntry),
		r:        r,
		b:        b,
	}
	go store.cleanup()
	return store
}

func (s *RateLimiterStore) get(key string) *rate.Limiter {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.limiters[key]
	if !ok {
		entry = &limiterEntry{limiter: rate.NewLimiter(s.r, s.b)}
		s.limiters[key] = entry
	}
	entry.lastSeen = time.Now()
	return entry.limiter
}

// cleanup removes limiters that haven't been used in the last 5 minutes.
func (s *RateLimiterStore) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.mu.Lock()
		for key, entry := range s.limiters {
			if time.Since(entry.lastSeen) > 5*time.Minute {
				delete(s.limiters, key)
			}
		}
		s.mu.Unlock()
	}
}

// RateLimitByIP limits requests by client IP address.
// Suitable for public routes where no authentication is required.
// r = requests per second, b = burst size.
func RateLimitByIP(r rate.Limit, b int) gin.HandlerFunc {
	store := NewRateLimiterStore(r, b)
	return func(c *gin.Context) {
		key := c.ClientIP()
		if !store.get(key).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests — please slow down",
			})
			return
		}
		c.Next()
	}
}

// RateLimitBySubject limits requests by the JWT subject claim.
// Falls back to IP if no subject is available in the context.
// Must be used after Auth middleware.
func RateLimitBySubject(r rate.Limit, b int) gin.HandlerFunc {
	store := NewRateLimiterStore(r, b)
	return func(c *gin.Context) {
		key, _ := c.Request.Context().Value("changed_by").(string)
		if key == "" {
			key = c.ClientIP() // fallback
		}
		if !store.get(key).Allow() {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "too many requests — please slow down",
			})
			return
		}
		c.Next()
	}
}
