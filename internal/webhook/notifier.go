package webhook

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const (
	// maxAttempts caps retries for a single delivery. Webhooks are
	// best-effort notifications, not a guaranteed-delivery queue (that's
	// what Kafka would be for, per the roadmap) — a handful of quick
	// retries is enough to absorb a transient blip without holding
	// resources for a destination that's genuinely down.
	maxAttempts = 3

	// requestTimeout bounds each individual delivery attempt. Generous
	// enough for a slow-but-alive endpoint, short enough not to pile up
	// goroutines if the destination hangs.
	requestTimeout = 5 * time.Second
)

// Notifier sends domain-event notifications to an external HTTP endpoint.
type Notifier struct {
	url        string
	secret     string
	httpClient *http.Client
	logger     *slog.Logger
	backoff    time.Duration // base backoff between retries; configurable for tests
}

// NewNotifier builds a Notifier for the given destination URL and HMAC
// signing secret. If url is empty, callers should not construct a Notifier
// at all — use config.WebhookEnabled() to decide whether to wire one up.
func NewNotifier(url, secret string, logger *slog.Logger) *Notifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &Notifier{
		url:    url,
		secret: secret,
		httpClient: &http.Client{
			Timeout: requestTimeout,
		},
		logger:  logger,
		backoff: 500 * time.Millisecond,
	}
}

// WithBackoff overrides the base retry backoff. Intended for tests that
// exercise the retry path without waiting on real-world delays; production
// callers should rely on the default set by NewNotifier.
func (n *Notifier) WithBackoff(d time.Duration) *Notifier {
	n.backoff = d
	return n
}

// NotifyStatusChanged sends a "customer.<newStatus>" event to the configured
// webhook URL.
//
// This method is fire-and-forget by contract: it is expected to be called
// from a separate goroutine (see service.CustomerService.UpdateStatus), and
// it deliberately swallows delivery failures after exhausting retries —
// logging them — rather than returning an error to a caller who, by the
// time delivery is attempted, has likely already responded to the original
// HTTP request. A webhook destination being unreachable must never fail or
// roll back the status transition that already succeeded in the database.
func (n *Notifier) NotifyStatusChanged(ctx context.Context, customerID uuid.UUID, oldStatus, newStatus, changedBy string) {
	payload := StatusChangedPayload{
		Event:      eventTypeForStatus(newStatus),
		CustomerID: customerID,
		OldStatus:  oldStatus,
		NewStatus:  newStatus,
		ChangedBy:  changedBy,
		OccurredAt: time.Now().UTC(),
	}

	body, err := json.Marshal(payload)
	if err != nil {
		n.logger.Error("webhook: failed to marshal payload",
			slog.String("customerId", customerID.String()),
			slog.Any("error", err))
		return
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if err := n.deliver(ctx, body); err != nil {
			lastErr = err
			n.logger.Warn("webhook: delivery attempt failed",
				slog.String("customerId", customerID.String()),
				slog.Int("attempt", attempt),
				slog.Any("error", err))

			if attempt < maxAttempts {
				// Fixed short backoff is enough here: this is a best-effort
				// notification with a small retry budget, not a queue
				// worth tuning exponential backoff for.
				time.Sleep(time.Duration(attempt) * n.backoff)
			}
			continue
		}
		return
	}

	n.logger.Error("webhook: delivery failed after all retries, giving up",
		slog.String("customerId", customerID.String()),
		slog.Int("attempts", maxAttempts),
		slog.Any("lastError", lastErr))
}

// deliver performs a single POST attempt and returns an error for any
// non-2xx response or transport failure.
func (n *Notifier) deliver(ctx context.Context, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	if n.secret != "" {
		req.Header.Set("X-Webhook-Signature", sign(body, n.secret))
	}

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("destination returned status %d", resp.StatusCode)
	}
	return nil
}

// sign computes an HMAC-SHA256 signature of body using secret, hex-encoded.
// Receivers verify by recomputing this over the raw request body — the same
// pattern used by GitHub, Stripe, and most webhook providers — to confirm
// the request actually came from this API and wasn't forged or tampered
// with in transit.
func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}
