package acceptance_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAcceptance_Metrics(t *testing.T) {
	db := setupDB(t)
	srv := startServer(t, db)

	t.Run("GET /metrics é público e responde 200", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})

	t.Run("GET /metrics expõe métricas no formato Prometheus", func(t *testing.T) {
		resp, err := http.Get(srv.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		text := string(body)

		// go_goroutines is part of the default Prometheus client registry,
		// always present regardless of any app-specific metric having been
		// recorded yet — a good baseline check that promhttp.Handler() is
		// actually wired up and serving the real registry, not an empty one.
		assert.True(t, strings.Contains(text, "go_goroutines"),
			"esperado que /metrics inclua métricas padrão do runtime Go")
	})

	t.Run("GET /metrics reflete uma request HTTP anterior", func(t *testing.T) {
		// Any prior request in this test run (including the ones above)
		// should already have incremented http_requests_total for the
		// GET /metrics route itself.
		resp, err := http.Get(srv.URL + "/metrics")
		require.NoError(t, err)
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		text := string(body)

		assert.True(t, strings.Contains(text, "http_requests_total"),
			"esperado que /metrics inclua a métrica http_requests_total")
	})
}
