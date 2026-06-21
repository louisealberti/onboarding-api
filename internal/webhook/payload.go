// Package webhook notifies external systems when a customer's status
// changes (the only domain event the API currently emits). Delivery is
// best-effort: a failing or slow webhook destination must never affect the
// outcome of the API request that triggered it. See Notifier.Notify for the
// fire-and-forget contract.
package webhook

import (
	"time"

	"github.com/google/uuid"
)

// EventType identifies the kind of domain event a webhook payload carries.
// "customer.approved" / "customer.blocked" style names, as anticipated in
// the project roadmap, are derived from the status transition itself rather
// than hardcoded per status — see eventTypeForStatus.
type EventType string

// StatusChangedPayload is the JSON body POSTed to the configured webhook URL
// whenever a customer transitions status via PATCH /v1/customers/:id/status.
type StatusChangedPayload struct {
	Event      EventType `json:"event"`
	CustomerID uuid.UUID `json:"customerId"`
	OldStatus  string    `json:"oldStatus"`
	NewStatus  string    `json:"newStatus"`
	ChangedBy  string    `json:"changedBy"`
	OccurredAt time.Time `json:"occurredAt"`
}

// eventTypeForStatus maps a resulting status to a domain event name, e.g.
// "approved" -> "customer.approved". This mirrors the event names already
// referenced in the project roadmap (customer.approved, customer.blocked).
func eventTypeForStatus(status string) EventType {
	return EventType("customer." + status)
}
