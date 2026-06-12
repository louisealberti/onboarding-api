package acceptance_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/louisealberti/onboarding-api/internal/handler"
	"github.com/louisealberti/onboarding-api/internal/middleware"
	"github.com/louisealberti/onboarding-api/internal/repository"
	"github.com/louisealberti/onboarding-api/internal/service"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// setupDB sobe um container Postgres e aplica as migrations.
// O container é encerrado via t.Cleanup ao fim do teste pai.
// Compartilhe o mesmo db entre subtestes de um mesmo grupo para
// que o estado acumule propositalmente nos testes de lifecycle.
func setupDB(t *testing.T) *sql.DB {
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

// startServer monta a stack completa (repo → service → handler → gin)
// e devolve um httptest.Server real. O servidor é encerrado via t.Cleanup.
func startServer(t *testing.T, db *sql.DB) *httptest.Server {
	t.Helper()

	gin.SetMode(gin.TestMode)

	repo := repository.NewCustomerRepository(db)
	idempotencyRepo := repository.NewIdempotencyRepository(db)
	svc := service.NewCustomerService(repo)
	h := handler.NewCustomerHandler(svc)
	hh := handler.NewHealthHandler(db, handler.BuildInfo{Version: "test", BuildTime: "unknown"})

	r := gin.New()
	r.Use(middleware.RequestID())

	r.GET("/health", hh.Health)

	v1 := r.Group("/v1")
	v1.POST("/customers", middleware.Idempotency(idempotencyRepo), h.CreateCustomer)
	v1.GET("/customers/:id", h.GetCustomerByID)
	v1.GET("/customers", h.ListCustomers)
	v1.PUT("/customers/:id", h.UpdateCustomer)
	v1.PATCH("/customers/:id/status", h.UpdateStatus)
	v1.DELETE("/customers/:id", h.DeleteCustomer)

	srv := httptest.NewServer(r)
	t.Cleanup(srv.Close)
	return srv
}

// ── HTTP helpers ──────────────────────────────────────────────────────────
// Cada helper dispara uma request real contra o httptest.Server e devolve
// a *http.Response — o teste é responsável por fazer defer resp.Body.Close().

func apiPost(t *testing.T, srv *httptest.Server, path string, body map[string]any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	resp, err := http.Post(srv.URL+path, "application/json", bytes.NewReader(b))
	require.NoError(t, err)
	return resp
}

func apiPatch(t *testing.T, srv *httptest.Server, path string, body map[string]any) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPatch, srv.URL+path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

func apiGet(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	resp, err := http.Get(srv.URL + path)
	require.NoError(t, err)
	return resp
}

func apiDelete(t *testing.T, srv *httptest.Server, path string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodDelete, srv.URL+path, nil)
	require.NoError(t, err)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// apiPostWithKey dispara um POST com um Idempotency-Key header.
func apiPostWithKey(t *testing.T, srv *httptest.Server, path string, body map[string]any, key string) *http.Response {
	t.Helper()
	b, err := json.Marshal(body)
	require.NoError(t, err)
	req, err := http.NewRequest(http.MethodPost, srv.URL+path, bytes.NewReader(b))
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", key)
	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	return resp
}

// decodeBody faz o decode do JSON da response num map e fecha o body.
func decodeBody(t *testing.T, resp *http.Response) map[string]any {
	t.Helper()
	defer resp.Body.Close()
	var m map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&m))
	return m
}

// validCustomerPayload devolve um payload mínimo válido para POST /customers.
// Sobrescreva os campos necessários no teste.
func validCustomerPayload() map[string]any {
	return map[string]any{
		"firstName":   "Ana",
		"lastName":    "Ferreira",
		"email":       "ana@example.com",
		"taxId":       "52998224725",
		"countryCode": "BR",
	}
}
