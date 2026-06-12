package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

// IdempotencyRecord holds a previously processed request result.
type IdempotencyRecord struct {
	Key        string
	StatusCode int
	Response   json.RawMessage
	CreatedAt  time.Time
	ExpiresAt  time.Time
}

// IdempotencyRepository handles persistence of idempotency keys.
type IdempotencyRepository struct {
	DB *sql.DB
}

func NewIdempotencyRepository(db *sql.DB) *IdempotencyRepository {
	return &IdempotencyRepository{DB: db}
}

// Get returns the record for the given key, or sql.ErrNoRows if not found or expired.
func (r *IdempotencyRepository) Get(ctx context.Context, key string) (*IdempotencyRecord, error) {
	query := `
		SELECT key, status_code, response, created_at, expires_at
		FROM idempotency_keys
		WHERE key = $1 AND expires_at > NOW()
	`
	record := &IdempotencyRecord{}
	err := r.DB.QueryRowContext(ctx, query, key).Scan(
		&record.Key,
		&record.StatusCode,
		&record.Response,
		&record.CreatedAt,
		&record.ExpiresAt,
	)
	if err != nil {
		return nil, err
	}
	return record, nil
}

// Save persists a new idempotency record with a 24h TTL.
func (r *IdempotencyRepository) Save(ctx context.Context, key string, statusCode int, response json.RawMessage) error {
	query := `
		INSERT INTO idempotency_keys (key, status_code, response, expires_at)
		VALUES ($1, $2, $3, NOW() + INTERVAL '24 hours')
		ON CONFLICT (key) DO NOTHING
	`
	_, err := r.DB.ExecContext(ctx, query, key, statusCode, response)
	return err
}
