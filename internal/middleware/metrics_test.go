package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/louisealberti/onboarding-api/internal/metrics"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestMetrics_RecordsRequestByRoutePattern(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/v1/customers/:id", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	before := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/v1/customers/:id", "200"))

	req := httptest.NewRequest(http.MethodGet, "/v1/customers/abc-123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	after := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "/v1/customers/:id", "200"))

	if after != before+1 {
		t.Errorf("expected counter for route pattern to increment by 1, went from %v to %v", before, after)
	}
}

func TestMetrics_GroupsUnmatchedRoutesUnderSingleLabel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	// No routes registered — every request is unmatched (404).

	before := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "unmatched", "404"))

	req := httptest.NewRequest(http.MethodGet, "/this/path/does/not/exist", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	after := testutil.ToFloat64(metrics.HTTPRequestsTotal.WithLabelValues("GET", "unmatched", "404"))

	if after != before+1 {
		t.Errorf("expected unmatched-route counter to increment by 1, went from %v to %v", before, after)
	}
}

func TestMetrics_RecordsDurationHistogram(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Metrics())
	r.GET("/v1/health-check-route", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	beforeCount := testutil.CollectAndCount(metrics.HTTPRequestDuration)

	req := httptest.NewRequest(http.MethodGet, "/v1/health-check-route", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	afterCount := testutil.CollectAndCount(metrics.HTTPRequestDuration)

	if afterCount <= beforeCount {
		t.Errorf("expected at least one new histogram sample, before=%d after=%d", beforeCount, afterCount)
	}
}
