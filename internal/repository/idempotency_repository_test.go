package repository_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

func setupIdempotencyDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts(
			"../../db/migrations/000001_create_customers_addresses_phones_tables.up.sql",
			"../../db/migrations/000003_create_idempotency_keys_table.up.sql",
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

func TestIntegration_Idempotency_SaveAndGet(t *testing.T) {
	db := setupIdempotencyDB(t)
	repo := repository.NewIdempotencyRepository(db)
	ctx := context.Background()

	key := "test-key-123"
	response := json.RawMessage(`{"id":"abc","status":"pending"}`)

	err := repo.Save(ctx, key, 201, response)
	require.NoError(t, err)

	record, err := repo.Get(ctx, key)
	require.NoError(t, err)
	assert.Equal(t, key, record.Key)
	assert.Equal(t, 201, record.StatusCode)
	assert.JSONEq(t, string(response), string(record.Response))
}

func TestIntegration_Idempotency_NotFound(t *testing.T) {
	db := setupIdempotencyDB(t)
	repo := repository.NewIdempotencyRepository(db)
	ctx := context.Background()

	record, err := repo.Get(ctx, "nonexistent-key")

	assert.Nil(t, record)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegration_Idempotency_DuplicateSaveIsNoop(t *testing.T) {
	db := setupIdempotencyDB(t)
	repo := repository.NewIdempotencyRepository(db)
	ctx := context.Background()

	key := "duplicate-key"
	first := json.RawMessage(`{"id":"first"}`)
	second := json.RawMessage(`{"id":"second"}`)

	require.NoError(t, repo.Save(ctx, key, 201, first))
	require.NoError(t, repo.Save(ctx, key, 201, second)) // ON CONFLICT DO NOTHING

	record, err := repo.Get(ctx, key)
	require.NoError(t, err)
	assert.JSONEq(t, string(first), string(record.Response)) // original preserved
}
