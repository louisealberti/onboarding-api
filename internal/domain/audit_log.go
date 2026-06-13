package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of change recorded in the audit log.
type AuditAction string

const (
	AuditActionCreated       AuditAction = "created"
	AuditActionUpdated       AuditAction = "updated"
	AuditActionStatusChanged AuditAction = "status_changed"
)

// AuditLog records a change to a customer for compliance and traceability.
type AuditLog struct {
	ID         uuid.UUID       `json:"id"`
	CustomerID uuid.UUID       `json:"customerId"`
	Action     AuditAction     `json:"action"`
	OldValue   json.RawMessage `json:"oldValue,omitempty"`
	NewValue   json.RawMessage `json:"newValue"`
	ChangedBy  string          `json:"changedBy"`
	CreatedAt  time.Time       `json:"createdAt"`
}
