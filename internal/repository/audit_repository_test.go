package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/louisealberti/onboarding-api/internal/domain"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupAuditDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts(
			"../../db/migrations/000001_create_customers_addresses_phones_tables.up.sql",
			"../../db/migrations/000004_create_audit_logs_table.up.sql",
		),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = pgc.Terminate(ctx) })

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := sql.Open("pgx", connStr)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	require.NoError(t, db.Ping())
	return db
}

func insertCustomerForAudit(t *testing.T, db *sql.DB) uuid.UUID {
	t.Helper()
	id := uuid.New()
	_, err := db.Exec(`
		INSERT INTO customers (id, first_name, last_name, email, tax_id, country_code, status, version, created_at, updated_at)
		VALUES ($1, 'Ana', 'Ferreira', 'ana@example.com', '52998224725', 'BR', 'pending', 1, NOW(), NOW())
	`, id)
	require.NoError(t, err)
	return id
}

func TestIntegration_AuditLog_SaveAndList(t *testing.T) {
	db := setupAuditDB(t)
	repo := repository.NewAuditRepository(db)
	ctx := context.Background()

	customerID := insertCustomerForAudit(t, db)
	newValue, _ := json.Marshal(map[string]string{"status": "approved"})

	log := &domain.AuditLog{
		ID:         uuid.New(),
		CustomerID: customerID,
		Action:     domain.AuditActionStatusChanged,
		NewValue:   newValue,
		ChangedBy:  "operator@fintech.com",
		CreatedAt:  time.Now().UTC(),
	}

	err := repo.Save(ctx, log)
	require.NoError(t, err)

	logs, err := repo.ListByCustomer(ctx, customerID)
	require.NoError(t, err)
	require.Len(t, logs, 1)
	assert.Equal(t, domain.AuditActionStatusChanged, logs[0].Action)
	assert.Equal(t, "operator@fintech.com", logs[0].ChangedBy)
}

func TestIntegration_AuditLog_OldValuePreserved(t *testing.T) {
	db := setupAuditDB(t)
	repo := repository.NewAuditRepository(db)
	ctx := context.Background()

	customerID := insertCustomerForAudit(t, db)
	oldValue, _ := json.Marshal(map[string]string{"status": "pending"})
	newValue, _ := json.Marshal(map[string]string{"status": "approved"})

	log := &domain.AuditLog{
		ID:         uuid.New(),
		CustomerID: customerID,
		Action:     domain.AuditActionStatusChanged,
		OldValue:   oldValue,
		NewValue:   newValue,
		ChangedBy:  "system",
		CreatedAt:  time.Now().UTC(),
	}

	require.NoError(t, repo.Save(ctx, log))

	logs, err := repo.ListByCustomer(ctx, customerID)
	require.NoError(t, err)
	assert.JSONEq(t, string(oldValue), string(logs[0].OldValue))
	assert.JSONEq(t, string(newValue), string(logs[0].NewValue))
}

func TestIntegration_AuditLog_OrderedNewestFirst(t *testing.T) {
	db := setupAuditDB(t)
	repo := repository.NewAuditRepository(db)
	ctx := context.Background()

	customerID := insertCustomerForAudit(t, db)

	actions := []domain.AuditAction{
		domain.AuditActionCreated,
		domain.AuditActionStatusChanged,
		domain.AuditActionUpdated,
	}

	for _, action := range actions {
		newValue, _ := json.Marshal(map[string]string{"action": string(action)})
		require.NoError(t, repo.Save(ctx, &domain.AuditLog{
			ID:         uuid.New(),
			CustomerID: customerID,
			Action:     action,
			NewValue:   newValue,
			ChangedBy:  "system",
			CreatedAt:  time.Now().UTC(),
		}))
		time.Sleep(10 * time.Millisecond) // ensure distinct timestamps
	}

	logs, err := repo.ListByCustomer(ctx, customerID)
	require.NoError(t, err)
	require.Len(t, logs, 3)
	assert.Equal(t, domain.AuditActionUpdated, logs[0].Action) // newest
	assert.Equal(t, domain.AuditActionStatusChanged, logs[1].Action)
	assert.Equal(t, domain.AuditActionCreated, logs[2].Action) // oldest
}

func TestIntegration_AuditLog_EmptyForUnknownCustomer(t *testing.T) {
	db := setupAuditDB(t)
	repo := repository.NewAuditRepository(db)
	ctx := context.Background()

	logs, err := repo.ListByCustomer(ctx, uuid.New())

	require.NoError(t, err)
	assert.Empty(t, logs)
}
