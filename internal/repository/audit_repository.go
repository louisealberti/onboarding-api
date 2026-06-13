package repository

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/louisealberti/onboarding-api/internal/domain"
)

// AuditRepository handles persistence of audit log entries.
type AuditRepository struct {
	DB *sql.DB
}

func NewAuditRepository(db *sql.DB) *AuditRepository {
	return &AuditRepository{DB: db}
}

// Save inserts a new audit log entry.
func (r *AuditRepository) Save(ctx context.Context, log *domain.AuditLog) error {
	query := `
		INSERT INTO audit_logs (id, customer_id, action, old_value, new_value, changed_by, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`
	_, err := r.DB.ExecContext(ctx, query,
		log.ID, log.CustomerID, log.Action,
		log.OldValue, log.NewValue,
		log.ChangedBy, log.CreatedAt,
	)
	return err
}

// ListByCustomer returns all audit entries for a customer, newest first.
func (r *AuditRepository) ListByCustomer(ctx context.Context, customerID uuid.UUID) ([]domain.AuditLog, error) {
	query := `
		SELECT id, customer_id, action, old_value, new_value, changed_by, created_at
		FROM audit_logs
		WHERE customer_id = $1
		ORDER BY created_at DESC
	`
	rows, err := r.DB.QueryContext(ctx, query, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []domain.AuditLog
	for rows.Next() {
		var l domain.AuditLog
		var oldValue []byte
		if err := rows.Scan(
			&l.ID, &l.CustomerID, &l.Action,
			&oldValue, &l.NewValue,
			&l.ChangedBy, &l.CreatedAt,
		); err != nil {
			return nil, err
		}
		if oldValue != nil {
			l.OldValue = json.RawMessage(oldValue)
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}
