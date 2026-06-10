package repository_test

import (
	"context"
	"database/sql"
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

// setupDB sobe um container Postgres, aplica as migrations e retorna o *sql.DB.
// O container é encerrado via t.Cleanup ao fim de cada teste.
func setupDB(t *testing.T) *sql.DB {
	t.Helper()
	ctx := context.Background()

	pgc, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		postgres.WithInitScripts("../../db/migrations/000001_create_customers_addresses_phones_tables.up.sql"),
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

// ── helpers ────────────────────────────────────────────────────────────────

func insertCustomer(t *testing.T, db *sql.DB, c *domain.Customer) {
	t.Helper()
	_, err := db.Exec(
		`INSERT INTO customers (id, first_name, last_name, email, tax_id, country_code, status, version, created_at, updated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.FirstName, c.LastName, c.Email, c.TaxID,
		c.CountryCode, c.Status, c.Version, c.CreatedAt, c.UpdatedAt,
	)
	require.NoError(t, err)
}

func newPersistedCustomer() *domain.Customer {
	now := time.Now().UTC().Truncate(time.Millisecond)
	return &domain.Customer{
		ID:          uuid.New(),
		FirstName:   "Ana",
		LastName:    "Ferreira",
		Email:       "ana@example.com",
		TaxID:       "52998224725",
		CountryCode: "BR",
		Status:      "pending",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

// ── CreateCustomer ─────────────────────────────────────────────────────────

func TestIntegration_CreateCustomer_Success(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	c.Address = &domain.Address{
		ID:         uuid.New(),
		CustomerID: c.ID,
		Street:     "Rua das Flores, 42",
		City:       "Curitiba",
		State:      "PR",
		PostalCode: "80000-000",
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	c.Phones = []domain.Phone{
		{ID: uuid.New(), CustomerID: c.ID, CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
	}

	err := repo.CreateCustomer(ctx, c)

	require.NoError(t, err)

	// Confirma que os dados chegaram no banco
	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM customers WHERE id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM addresses WHERE customer_id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM phones WHERE customer_id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestIntegration_CreateCustomer_WithoutAddressAndPhones(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()

	err := repo.CreateCustomer(ctx, c)

	require.NoError(t, err)

	var count int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM customers WHERE id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 1, count)

	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM addresses WHERE customer_id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 0, count)

	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM phones WHERE customer_id = $1`, c.ID).Scan(&count))
	assert.Equal(t, 0, count)
}

func TestIntegration_CreateCustomer_DuplicateEmail(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c1 := newPersistedCustomer()
	require.NoError(t, repo.CreateCustomer(ctx, c1))

	c2 := newPersistedCustomer()
	c2.ID = uuid.New()
	c2.TaxID = "11122233344" // tax_id diferente para isolar o erro no email

	err := repo.CreateCustomer(ctx, c2)

	assert.Error(t, err) // UNIQUE constraint em email
}

// ── GetByEmail ─────────────────────────────────────────────────────────────

func TestIntegration_GetByEmail_Success(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)

	result, err := repo.GetByEmail(ctx, c.Email)

	require.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.Email, result.Email)
}

func TestIntegration_GetByEmail_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	result, err := repo.GetByEmail(ctx, "naoexiste@example.com")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegration_GetByEmail_SoftDeletedIsInvisible(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)
	_, err := db.Exec(`UPDATE customers SET deleted_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	result, err := repo.GetByEmail(ctx, c.Email)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ── GetByID ────────────────────────────────────────────────────────────────

func TestIntegration_GetByID_Success_WithAddressAndPhones(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	c.Address = &domain.Address{
		ID:         uuid.New(),
		CustomerID: c.ID,
		Street:     "Rua B, 100",
		City:       "Curitiba",
		State:      "PR",
		PostalCode: "80010-000",
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	c.Phones = []domain.Phone{
		{ID: uuid.New(), CustomerID: c.ID, CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
		{ID: uuid.New(), CustomerID: c.ID, CountryCode: "55", AreaCode: "41", Number: "33334444", Type: "landline", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	result, err := repo.GetByID(ctx, c.ID)

	require.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.Email, result.Email)
	require.NotNil(t, result.Address)
	assert.Equal(t, "Rua B, 100", result.Address.Street)
	assert.Len(t, result.Phones, 2)
}

func TestIntegration_GetByID_Success_WithoutAddress(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)

	result, err := repo.GetByID(ctx, c.ID)

	require.NoError(t, err)
	assert.Nil(t, result.Address)
	assert.Empty(t, result.Phones)
}

func TestIntegration_GetByID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	result, err := repo.GetByID(ctx, uuid.New())

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegration_GetByID_SoftDeletedIsInvisible(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)
	_, err := db.Exec(`UPDATE customers SET deleted_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	result, err := repo.GetByID(ctx, c.ID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ── UpdateCustomer ─────────────────────────────────────────────────────────

func TestIntegration_UpdateCustomer_BasicFields(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	require.NoError(t, repo.CreateCustomer(ctx, c))

	c.FirstName = "Ana Paula"
	c.Version = 2
	c.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)

	err := repo.UpdateCustomer(ctx, c)

	require.NoError(t, err)

	var firstName string
	var version int
	require.NoError(t, db.QueryRow(`SELECT first_name, version FROM customers WHERE id = $1`, c.ID).Scan(&firstName, &version))
	assert.Equal(t, "Ana Paula", firstName)
	assert.Equal(t, 2, version)
}

func TestIntegration_UpdateCustomer_UpdatesAddress(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	c.Address = &domain.Address{
		ID: uuid.New(), CustomerID: c.ID,
		Street: "Rua A, 1", City: "Curitiba", State: "PR", PostalCode: "80000-000",
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt,
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	c.Address.Street = "Rua B, 200"
	c.Address.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)
	c.Version = 2

	err := repo.UpdateCustomer(ctx, c)

	require.NoError(t, err)

	var street string
	require.NoError(t, db.QueryRow(`SELECT street FROM addresses WHERE customer_id = $1`, c.ID).Scan(&street))
	assert.Equal(t, "Rua B, 200", street)
}

func TestIntegration_UpdateCustomer_ReplacesPhones(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	phoneID := uuid.New()
	c.Phones = []domain.Phone{
		{ID: phoneID, CustomerID: c.ID, CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	// Troca por um telefone novo (sem ID — será inserido)
	c.Phones = []domain.Phone{
		{CountryCode: "55", AreaCode: "11", Number: "999999999", Type: "mobile", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
	}
	c.Version = 2

	err := repo.UpdateCustomer(ctx, c)

	require.NoError(t, err)

	// O antigo foi soft-deleted, o novo foi inserido
	var active int
	require.NoError(t, db.QueryRow(`SELECT COUNT(*) FROM phones WHERE customer_id = $1 AND deleted_at IS NULL`, c.ID).Scan(&active))
	assert.Equal(t, 1, active)

	var number string
	require.NoError(t, db.QueryRow(`SELECT number FROM phones WHERE customer_id = $1 AND deleted_at IS NULL`, c.ID).Scan(&number))
	assert.Equal(t, "999999999", number)
}

func TestIntegration_UpdateCustomer_ReactivatesExistingPhone(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	phoneID := uuid.New()
	c.Phones = []domain.Phone{
		{ID: phoneID, CustomerID: c.ID, CountryCode: "55", AreaCode: "41", Number: "991112233", Type: "mobile", CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt},
	}
	require.NoError(t, repo.CreateCustomer(ctx, c))

	// Reenvia o mesmo phone com ID — deve reativar (deleted_at = NULL)
	c.Version = 2
	c.Phones[0].UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)

	err := repo.UpdateCustomer(ctx, c)

	require.NoError(t, err)

	var deletedAt *time.Time
	require.NoError(t, db.QueryRow(`SELECT deleted_at FROM phones WHERE id = $1`, phoneID).Scan(&deletedAt))
	assert.Nil(t, deletedAt)
}

// ── SoftDelete ─────────────────────────────────────────────────────────────

func TestIntegration_SoftDelete_Success(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)

	err := repo.SoftDelete(ctx, c.ID)

	require.NoError(t, err)

	var status string
	var deletedAt *time.Time
	require.NoError(t, db.QueryRow(`SELECT status, deleted_at FROM customers WHERE id = $1`, c.ID).Scan(&status, &deletedAt))
	assert.Equal(t, "terminated", status)
	assert.NotNil(t, deletedAt)
}

func TestIntegration_SoftDelete_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	err := repo.SoftDelete(ctx, uuid.New())

	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegration_SoftDelete_AlreadyDeleted(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)
	require.NoError(t, repo.SoftDelete(ctx, c.ID))

	// Segunda chamada — WHERE deleted_at IS NULL não bate, rowsAffected = 0
	err := repo.SoftDelete(ctx, c.ID)

	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ── GetByTaxID ─────────────────────────────────────────────────────────────

func TestIntegration_GetByTaxID_Success(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)

	result, err := repo.GetByTaxID(ctx, c.TaxID)

	require.NoError(t, err)
	assert.Equal(t, c.ID, result.ID)
	assert.Equal(t, c.TaxID, result.TaxID)
	assert.Equal(t, c.Email, result.Email)
}

func TestIntegration_GetByTaxID_NotFound(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	result, err := repo.GetByTaxID(ctx, "00000000000")

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

func TestIntegration_GetByTaxID_SoftDeletedIsInvisible(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)
	_, err := db.Exec(`UPDATE customers SET deleted_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	result, err := repo.GetByTaxID(ctx, c.TaxID)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}

// ── UpdateCustomer — status persistence ────────────────────────────────────

func TestIntegration_UpdateCustomer_PersistsStatus(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer() // status: "pending"
	require.NoError(t, repo.CreateCustomer(ctx, c))

	c.Status = "approved"
	c.Version = 2
	c.UpdatedAt = time.Now().UTC().Truncate(time.Millisecond)

	err := repo.UpdateCustomer(ctx, c)

	require.NoError(t, err)

	var status string
	require.NoError(t, db.QueryRow(`SELECT status FROM customers WHERE id = $1`, c.ID).Scan(&status))
	assert.Equal(t, "approved", status)
}

// ── GetByEmail — bug documentado ───────────────────────────────────────────

func TestIntegration_GetByEmail_SoftDeletedStillReturns(t *testing.T) {
	db := setupDB(t)
	repo := repository.NewCustomerRepository(db)
	ctx := context.Background()

	c := newPersistedCustomer()
	insertCustomer(t, db, c)
	_, err := db.Exec(`UPDATE customers SET deleted_at = NOW() WHERE id = $1`, c.ID)
	require.NoError(t, err)

	result, err := repo.GetByEmail(ctx, c.Email)

	assert.Nil(t, result)
	assert.ErrorIs(t, err, sql.ErrNoRows)
}
