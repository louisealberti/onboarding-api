package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcceptance_CustomerLifecycle valida o fluxo completo de um customer
// de ponta a ponta: handler → service → repository → banco real.
//
// Os subtestes compartilham o mesmo container e servidor — o estado acumula
// propositalmente. A ordem importa: cada subtest depende do anterior.
func TestAcceptance_CustomerLifecycle(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	var customerID string

	t.Run("POST /customers cria customer com status pending", func(t *testing.T) {
		resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		require.Contains(t, body, "id", "response deve conter o id do customer criado")
		customerID = body["id"].(string)
		assert.Equal(t, "pending", body["status"])
		assert.Equal(t, float64(1), body["version"])
	})

	t.Run("GET /customers/:id retorna o customer criado", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/"+customerID)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, customerID, body["id"])
		assert.Equal(t, "ana@example.com", body["email"])
		assert.Equal(t, "pending", body["status"])
	})

	t.Run("PATCH status pending → approved", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "approved",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("PATCH status approved → active", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "active",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("PATCH status active → suspended", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "suspended",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("PATCH status suspended → active (reativação)", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "active",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("PATCH status active → blocked", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "blocked",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("DELETE customer bloqueado retorna 403", func(t *testing.T) {
		resp := apiDelete(t, srv, "/v1/customers/"+customerID)
		defer resp.Body.Close()

		// customer bloqueado não pode ser deletado (ErrCustomerIsBlocked)
		assert.Equal(t, http.StatusForbidden, resp.StatusCode)
	})

	t.Run("PATCH status blocked → terminated (encerramento)", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+customerID+"/status", map[string]any{
			"status": "terminated",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Contains(t, body["message"], "updated")
	})

	t.Run("GET /customers/:id após terminated ainda retorna o customer", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/"+customerID)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "terminated", body["status"])
	})
}

// TestAcceptance_TaxIDSearch valida a busca por taxId.
// Container próprio — isolado do lifecycle acima.
func TestAcceptance_TaxIDSearch(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	// Cria o customer primeiro
	resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	resp.Body.Close()

	t.Run("GET /customers?taxId= retorna o customer", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers?taxId=52998224725")
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "52998224725", body["taxId"])
	})

	t.Run("GET /customers?taxId= inexistente retorna 404", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers?taxId=00000000000")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}

// TestAcceptance_Delete valida soft delete num customer não bloqueado.
func TestAcceptance_Delete(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
	body := decodeBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	id := body["id"].(string)

	t.Run("DELETE customer pending retorna 204", func(t *testing.T) {
		resp := apiDelete(t, srv, "/v1/customers/"+id)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNoContent, resp.StatusCode)
	})

	t.Run("GET /customers/:id após delete retorna 404", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/"+id)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})
}
