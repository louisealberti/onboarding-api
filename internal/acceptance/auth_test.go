package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptance_Auth(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	t.Run("POST /auth/token gera token válido", func(t *testing.T) {
		resp := apiPostAs(t, srv, "/auth/token", map[string]any{
			"sub":  "user@fintech.com",
			"role": "admin",
		}, "") // rota pública, sem token
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body, "token")
	})

	t.Run("POST /auth/token com role inválida retorna 400", func(t *testing.T) {
		resp := apiPostAs(t, srv, "/auth/token", map[string]any{
			"sub":  "user@fintech.com",
			"role": "superuser",
		}, "")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("request sem token retorna 401", func(t *testing.T) {
		req, _ := newRequestNoAuth("GET", srv.URL+"/v1/customers", nil)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)
	})

	t.Run("token com role operator não pode criar customer", func(t *testing.T) {
		token := GenerateTestToken(t, "op@fintech.com", "operator")
		resp := apiPostAs(t, srv, "/v1/customers", validCustomerPayload(), token)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("token com role operator pode listar customers", func(t *testing.T) {
		token := GenerateTestToken(t, "op@fintech.com", "operator")
		req, _ := newRequestWithToken("GET", srv.URL+"/v1/customers", nil, token)
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("token com role admin pode criar customer", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "jwt-admin@example.com"
		payload["taxId"] = "11144477735"

		resp := apiPost(t, srv, "/v1/customers", payload)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
	})
}
