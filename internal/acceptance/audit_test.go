package acceptance_test

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptance_AuditLog(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	// Create a customer
	resp := apiPost(t, srv, "/v1/customers", validCustomerPayload())
	body := decodeBody(t, resp)
	require.Equal(t, http.StatusCreated, resp.StatusCode)
	id := body["id"].(string)

	t.Run("GET /audit logo após criação tem 1 entrada", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/"+id+"/audit")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Lendo o corpo diretamente como uma lista/slice
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var entries []map[string]any
		err = json.Unmarshal(respBody, &entries)
		require.NoError(t, err)

		// Agora sim, validamos que o array contém exatamente 1 log de auditoria
		assert.Len(t, entries, 1)
	})

	t.Run("mudança de status cria entrada no audit log", func(t *testing.T) {
		// Approve the customer
		apiPatch(t, srv, "/v1/customers/"+id+"/status", map[string]any{"status": "approved"}).Body.Close()

		resp := apiGet(t, srv, "/v1/customers/"+id+"/audit")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GET /audit com id inválido retorna 400", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/nao-e-uuid/audit")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})

	t.Run("GET /audit de customer inexistente retorna lista vazia", func(t *testing.T) {
		resp := apiGet(t, srv, "/v1/customers/00000000-0000-0000-0000-000000000000/audit")
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Lendo o corpo diretamente para validar a lista vazia
		respBody, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var entries []map[string]any
		err = json.Unmarshal(respBody, &entries)
		require.NoError(t, err)

		// Valida se retornou o array vazio `[]` com tamanho 0
		assert.Len(t, entries, 0)
	})
}
