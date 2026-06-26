package metrics

import (
	"context"
	"database/sql"
	"time"
)

// dbStatsPollInterval is how often the connection pool gauges are
// refreshed. sql.DB.Stats() is cheap (in-memory counters, no query), so a
// short interval costs nothing meaningful; 5s gives Grafana dashboards
// responsive-feeling data without polling so often it shows up in profiles.
const dbStatsPollInterval = 5 * time.Second

// StartDBStatsCollector polls db.Stats() every dbStatsPollInterval and
// updates the DB connection pool gauges. It runs until ctx is canceled —
// callers should pass a context tied to the application's lifetime and let
// it stop naturally on shutdown, no separate cleanup call needed.
//
// WaitCount is cumulative (it only grows), so it's exposed as a counter:
// each tick adds the delta since the last poll rather than re-setting an
// absolute value, which is what a Prometheus counter expects.
func StartDBStatsCollector(ctx context.Context, db *sql.DB) {
	ticker := time.NewTicker(dbStatsPollInterval)
	defer ticker.Stop()

	var lastWaitCount int64

	collect := func() {
		stats := db.Stats()

		DBConnectionsOpen.Set(float64(stats.OpenConnections))
		DBConnectionsInUse.Set(float64(stats.InUse))
		DBConnectionsIdle.Set(float64(stats.Idle))

		if delta := stats.WaitCount - lastWaitCount; delta > 0 {
			DBWaitCountTotal.Add(float64(delta))
		}
		lastWaitCount = stats.WaitCount
	}

	collect() // first sample immediately, don't wait a full interval to populate gauges

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			collect()
		}
	}
}
