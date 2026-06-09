package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestAcceptance_CreateCustomer_Errors valida as respostas de erro do POST /customers.
// Cada subtest é independente — usa o mesmo container, mas cria dados próprios.
func TestAcceptance_CreateCustomer_Errors(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	t.Run("email duplicado retorna 409", func(t *testing.T) {
		// Primeiro POST: sucesso
		resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
		require.Equal(t, http.StatusCreated, resp.StatusCode)
		resp.Body.Close()

		// Segundo POST com o mesmo email
		resp = apiPost(t, srv, "/v1/customers", validCustomerPayload())
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusConflict, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("email ausente retorna 400", func(t *testing.T) {
		payload := validCustomerPayload()
		delete(payload, "email")

		resp := apiPost(t, srv, "/v1/customers", payload)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("email inválido retorna 400", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "nao-e-um-email"
		payload["taxId"] = "11144477735" // CPF diferente para não conflitar

		resp := apiPost(t, srv, "/v1/customers", payload)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("taxId ausente retorna 400", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "outro@example.com"
		delete(payload, "taxId")

		resp := apiPost(t, srv, "/v1/customers", payload)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("taxId inválido para o país retorna 400", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "invalido@example.com"
		payload["taxId"] = "00000000000" // CPF inválido

		resp := apiPost(t, srv, "/v1/customers", payload)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("countryCode ausente retorna 400", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "semcountry@example.com"
		payload["taxId"] = "11144477735"
		delete(payload, "countryCode")

		resp := apiPost(t, srv, "/v1/customers", payload)
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("body inválido (não é JSON) retorna 400", func(t *testing.T) {
		// apiPost serializa map, então usamos http diretamente aqui
		resp, err := http.Post(srv.URL+"/v1/customers", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestAcceptance_GetCustomer_Errors valida os casos de erro do GET /customers/:id.
func TestAcceptance_GetCustomer_Errors(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	t.Run("id inexistente retorna 404", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/00000000-0000-0000-0000-000000000000")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	t.Run("id com formato inválido retorna 400", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/nao-e-um-uuid")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

// TestAcceptance_UpdateStatus_Errors valida os casos de erro do PATCH status.
func TestAcceptance_UpdateStatus_Errors(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	// Cria e aprova um customer para ter estado conhecido
	resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
	body := decodeBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	id := body["id"].(string)

	t.Run("transição inválida retorna 422", func(t *testing.T) {
		// pending → active não é uma transição permitida
		resp := apiPatch(t, srv, "/v1/customers/"+id+"/status", map[string]any{
			"status": "active",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusUnprocessableEntity, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("customer inexistente retorna 404", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/00000000-0000-0000-0000-000000000000/status", map[string]any{
			"status": "approved",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("body sem status retorna 400", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/"+id+"/status", map[string]any{})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})

	t.Run("id inválido retorna 400", func(t *testing.T) {
		resp := apiPatch(t, srv, "/v1/customers/nao-e-uuid/status", map[string]any{
			"status": "approved",
		})
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
		assert.Contains(t, body, "error")
	})
}
