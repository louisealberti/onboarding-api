package webhook

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

func discardLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestNotifyStatusChanged_DeliversOnFirstAttempt(t *testing.T) {
	var received StatusChangedPayload
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, "", discardLogger())
	customerID := uuid.New()

	n.NotifyStatusChanged(context.Background(), customerID, "pending", "approved", "admin@fintech.com")

	if calls != 1 {
		t.Fatalf("expected exactly 1 delivery attempt, got %d", calls)
	}
	if received.Event != "customer.approved" {
		t.Errorf("expected event 'customer.approved', got %q", received.Event)
	}
	if received.CustomerID != customerID {
		t.Errorf("expected customerId %s, got %s", customerID, received.CustomerID)
	}
	if received.OldStatus != "pending" || received.NewStatus != "approved" {
		t.Errorf("unexpected status transition in payload: %+v", received)
	}
	if received.ChangedBy != "admin@fintech.com" {
		t.Errorf("expected changedBy 'admin@fintech.com', got %q", received.ChangedBy)
	}
}

func TestNotifyStatusChanged_SignsPayloadWhenSecretSet(t *testing.T) {
	var gotSignature string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature = r.Header.Get("X-Webhook-Signature")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, "test-secret", discardLogger())
	n.NotifyStatusChanged(context.Background(), uuid.New(), "pending", "approved", "admin@fintech.com")

	if gotSignature == "" {
		t.Error("expected X-Webhook-Signature header to be set when secret is configured")
	}
}

func TestNotifyStatusChanged_NoSignatureHeaderWhenSecretEmpty(t *testing.T) {
	var gotSignature string
	headerSet := false

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotSignature, headerSet = r.Header.Get("X-Webhook-Signature"), r.Header.Get("X-Webhook-Signature") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, "", discardLogger())
	n.NotifyStatusChanged(context.Background(), uuid.New(), "pending", "approved", "admin@fintech.com")

	if headerSet {
		t.Errorf("expected no signature header when secret is empty, got %q", gotSignature)
	}
}

func TestNotifyStatusChanged_RetriesOnFailureThenSucceeds(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&calls, 1)
		if n < 2 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, "", discardLogger()).WithBackoff(time.Millisecond)
	n.NotifyStatusChanged(context.Background(), uuid.New(), "pending", "approved", "admin@fintech.com")

	if calls != 2 {
		t.Fatalf("expected delivery to succeed on the 2nd attempt, got %d calls", calls)
	}
}

func TestNotifyStatusChanged_GivesUpAfterMaxAttempts(t *testing.T) {
	var calls int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	n := NewNotifier(srv.URL, "", discardLogger()).WithBackoff(time.Millisecond)
	// Must not panic, hang, or otherwise propagate an error: this is the
	// fire-and-forget contract callers rely on.
	n.NotifyStatusChanged(context.Background(), uuid.New(), "pending", "approved", "admin@fintech.com")

	if calls != maxAttempts {
		t.Fatalf("expected exactly %d attempts, got %d", maxAttempts, calls)
	}
}

func TestNotifyStatusChanged_UnreachableDestinationDoesNotPanic(t *testing.T) {
	// Port 0 on localhost — nothing listening, connection refused immediately.
	n := NewNotifier("http://127.0.0.1:0", "", discardLogger())

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expected no panic for an unreachable destination, got: %v", r)
		}
	}()

	n.NotifyStatusChanged(context.Background(), uuid.New(), "pending", "approved", "admin@fintech.com")
}

func TestEventTypeForStatus(t *testing.T) {
	cases := map[string]EventType{
		"approved":   "customer.approved",
		"blocked":    "customer.blocked",
		"active":     "customer.active",
		"terminated": "customer.terminated",
	}
	for status, want := range cases {
		if got := eventTypeForStatus(status); got != want {
			t.Errorf("eventTypeForStatus(%q) = %q, want %q", status, got, want)
		}
	}
}

func TestSign_IsDeterministicAndSecretSensitive(t *testing.T) {
	body := []byte(`{"event":"customer.approved"}`)

	sigA := sign(body, "secret-a")
	sigAAgain := sign(body, "secret-a")
	sigB := sign(body, "secret-b")

	if sigA != sigAAgain {
		t.Error("expected sign() to be deterministic for the same body and secret")
	}
	if sigA == sigB {
		t.Error("expected different secrets to produce different signatures")
	}
}
