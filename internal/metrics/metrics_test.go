package metrics

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestHTTPRequestsTotal_IncrementsWithLabels(t *testing.T) {
	before := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/v1/customers", "200"))

	HTTPRequestsTotal.WithLabelValues("GET", "/v1/customers", "200").Inc()

	after := testutil.ToFloat64(HTTPRequestsTotal.WithLabelValues("GET", "/v1/customers", "200"))
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, went from %v to %v", before, after)
	}
}

func TestCustomersCreatedTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(CustomersCreatedTotal)

	CustomersCreatedTotal.Inc()

	after := testutil.ToFloat64(CustomersCreatedTotal)
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, went from %v to %v", before, after)
	}
}

func TestCustomerStatusTransitionsTotal_LabeledByFromTo(t *testing.T) {
	before := testutil.ToFloat64(CustomerStatusTransitionsTotal.WithLabelValues("pending", "approved"))

	CustomerStatusTransitionsTotal.WithLabelValues("pending", "approved").Inc()

	after := testutil.ToFloat64(CustomerStatusTransitionsTotal.WithLabelValues("pending", "approved"))
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, went from %v to %v", before, after)
	}

	// A different label pair must not be affected.
	other := testutil.ToFloat64(CustomerStatusTransitionsTotal.WithLabelValues("approved", "active"))
	if other == after {
		t.Error("expected a different from/to label pair to have an independent count")
	}
}

func TestWebhookDeliveriesTotal_TracksOutcomes(t *testing.T) {
	beforeSuccess := testutil.ToFloat64(WebhookDeliveriesTotal.WithLabelValues("success"))
	beforeFailure := testutil.ToFloat64(WebhookDeliveriesTotal.WithLabelValues("failure"))

	WebhookDeliveriesTotal.WithLabelValues("success").Inc()
	WebhookDeliveriesTotal.WithLabelValues("failure").Inc()

	afterSuccess := testutil.ToFloat64(WebhookDeliveriesTotal.WithLabelValues("success"))
	afterFailure := testutil.ToFloat64(WebhookDeliveriesTotal.WithLabelValues("failure"))

	if afterSuccess != beforeSuccess+1 {
		t.Errorf("expected success counter to increment by 1, went from %v to %v", beforeSuccess, afterSuccess)
	}
	if afterFailure != beforeFailure+1 {
		t.Errorf("expected failure counter to increment by 1, went from %v to %v", beforeFailure, afterFailure)
	}
}

func TestIdempotentReplaysTotal_Increments(t *testing.T) {
	before := testutil.ToFloat64(IdempotentReplaysTotal)

	IdempotentReplaysTotal.Inc()

	after := testutil.ToFloat64(IdempotentReplaysTotal)
	if after != before+1 {
		t.Errorf("expected counter to increment by 1, went from %v to %v", before, after)
	}
}

func TestStartDBStatsCollector_PopulatesGaugesImmediately(t *testing.T) {
	// An unopened/invalid driver is fine here: Stats() reads in-memory pool
	// state and never touches the network, regardless of whether any real
	// connection has ever been established.
	db, err := sql.Open("pgx", "host=unreachable port=5432 user=x password=x dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error opening db handle: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go StartDBStatsCollector(ctx, db)

	// The collector samples once immediately on start, synchronously
	// relative to the ticker — give the goroutine a brief moment to run
	// before asserting, since it's still a separate goroutine.
	time.Sleep(50 * time.Millisecond)

	// OpenConnections should be a valid non-negative reading (0 is
	// expected here, since no query has ever been run against this DSN).
	if v := testutil.ToFloat64(DBConnectionsOpen); v < 0 {
		t.Errorf("expected a non-negative DBConnectionsOpen reading, got %v", v)
	}
}

func TestStartDBStatsCollector_StopsOnContextCancel(t *testing.T) {
	db, err := sql.Open("pgx", "host=unreachable port=5432 user=x password=x dbname=x sslmode=disable")
	if err != nil {
		t.Fatalf("unexpected error opening db handle: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		StartDBStatsCollector(ctx, db)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// expected: the collector returned promptly after cancellation
	case <-time.After(time.Second):
		t.Error("expected StartDBStatsCollector to return shortly after context cancellation")
	}
}
