// Package metrics defines this API's Prometheus metrics and exposes them at
// GET /metrics for scraping. Three groups are tracked:
//
//   - HTTP: request count and latency, labeled by method/path/status —
//     wired in via middleware.Metrics, the same pattern as middleware.Logger.
//   - Business: domain events that matter operationally (customers created,
//     status transitions, webhook delivery outcomes, idempotent replays) —
//     called directly from the service layer at the point each event happens.
//   - Database connection pool: gauges sourced from sql.DB.Stats(), polled
//     periodically rather than pushed, since the pool's state isn't an
//     "event" — see StartDBStatsCollector.
//
// Go runtime metrics (goroutines, memory, GC) are not defined here: the
// Prometheus client library's default registry already exposes them
// (go_goroutines, go_memstats_*, etc.) for free via promhttp.Handler().
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ---- HTTP ----

var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests, labeled by method, route, and status code.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "http_request_duration_seconds",
			Help: "HTTP request latency in seconds, labeled by method and route.",
			// Default buckets (prometheus.DefBuckets) span 5ms–10s, a good
			// fit for a typical CRUD API; no need for custom buckets here.
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// ---- Business ----

var (
	CustomersCreatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "customers_created_total",
			Help: "Total number of customers successfully created.",
		},
	)

	CustomerStatusTransitionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "customer_status_transitions_total",
			Help: "Total number of customer status transitions, labeled by origin and destination status.",
		},
		[]string{"from", "to"},
	)

	// WebhookDeliveriesTotal's "outcome" label is "success" or "failure" —
	// failure meaning all retries in webhook.Notifier were exhausted, not
	// each individual attempt (those would be noisy and aren't actionable
	// on their own; see webhook.Notifier.NotifyStatusChanged).
	WebhookDeliveriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "webhook_deliveries_total",
			Help: "Total number of webhook delivery attempts that ran to completion (all retries), labeled by outcome.",
		},
		[]string{"outcome"},
	)

	IdempotentReplaysTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "idempotent_replays_total",
			Help: "Total number of requests short-circuited by the idempotency middleware because their Idempotency-Key was already seen.",
		},
	)
)

// ---- Database connection pool ----

var (
	DBConnectionsOpen = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_open",
			Help: "Number of established connections to the database, both in use and idle (sql.DBStats.OpenConnections).",
		},
	)

	DBConnectionsInUse = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_in_use",
			Help: "Number of connections currently in use (sql.DBStats.InUse).",
		},
	)

	DBConnectionsIdle = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "db_connections_idle",
			Help: "Number of idle connections in the pool (sql.DBStats.Idle).",
		},
	)

	DBWaitCountTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "db_wait_count_total",
			Help: "Total number of connections that had to wait for a free slot in the pool (sql.DBStats.WaitCount, cumulative since process start).",
		},
	)
)
