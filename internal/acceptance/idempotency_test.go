package acceptance_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptance_Idempotency(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	t.Run("sem Idempotency-Key processa normalmente", func(t *testing.T) {
		resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
		body := decodeBody(t, resp)

		assert.Equal(t, http.StatusCreated, resp.StatusCode)
		assert.Contains(t, body, "id")
		assert.Empty(t, resp.Header.Get("X-Idempotency-Replayed"))
	})

	t.Run("segunda request com mesma key retorna response original", func(t *testing.T) {
		payload := validCustomerPayload()
		payload["email"] = "idem@example.com"
		payload["taxId"] = "11144477735"
		key := "idem-key-001"

		// Primeira request — cria o customer
		resp1 := apiPostWithKey(t, srv, "/v1/customers", payload, key)
		body1 := decodeBody(t, resp1)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)
		id1 := body1["id"].(string)

		// Segunda request com mesma key — deve replay
		resp2 := apiPostWithKey(t, srv, "/v1/customers", payload, key)
		body2 := decodeBody(t, resp2)

		assert.Equal(t, http.StatusCreated, resp2.StatusCode)
		assert.Equal(t, id1, body2["id"], "id deve ser o mesmo da primeira request")
		assert.Equal(t, "true", resp2.Header.Get("X-Idempotency-Replayed"))
	})

	t.Run("keys diferentes criam customers diferentes", func(t *testing.T) {
		p1 := validCustomerPayload()
		p1["email"] = "key1@example.com"
		p1["taxId"] = "29715304346"

		p2 := validCustomerPayload()
		p2["email"] = "key2@example.com"
		p2["taxId"] = "62322729434"

		resp1 := apiPostWithKey(t, srv, "/v1/customers", p1, "key-aaa")
		body1 := decodeBody(t, resp1)
		require.Equal(t, http.StatusCreated, resp1.StatusCode)

		resp2 := apiPostWithKey(t, srv, "/v1/customers", p2, "key-bbb")
		body2 := decodeBody(t, resp2)
		require.Equal(t, http.StatusCreated, resp2.StatusCode)

		assert.NotEqual(t, body1["id"], body2["id"])
		assert.Empty(t, resp2.Header.Get("X-Idempotency-Replayed"))
	})

	t.Run("request com erro não é armazenada", func(t *testing.T) {
		invalidPayload := validCustomerPayload()
		delete(invalidPayload, "email")
		key := "error-key-001"

		// Primeira request — erro de validação
		resp1 := apiPostWithKey(t, srv, "/v1/customers", invalidPayload, key)
		defer resp1.Body.Close()
		require.Equal(t, http.StatusBadRequest, resp1.StatusCode)

		// Segunda request com a mesma key mas payload válido — deve processar normalmente
		validPayload := validCustomerPayload()
		validPayload["email"] = "retry@example.com"
		validPayload["taxId"] = "66792954322"

		resp2 := apiPostWithKey(t, srv, "/v1/customers", validPayload, key)
		body2 := decodeBody(t, resp2)

		assert.Equal(t, http.StatusCreated, resp2.StatusCode)
		assert.Empty(t, resp2.Header.Get("X-Idempotency-Replayed"))
		assert.Contains(t, body2, "id")
	})
}
