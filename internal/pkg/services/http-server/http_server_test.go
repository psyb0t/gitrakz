package httpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestService_BootSmoke boots the real Service (real SQLite in a temp dir,
// no real gh/LLM calls — GET /api/v1/owners and GET / never touch either) on
// an OS-assigned port, and checks the two cheapest end-to-end signals: the
// generated API answers under /api/v1/, and the embedded SPA answers at /.
func TestService_BootSmoke(t *testing.T) {
	t.Setenv("GITRAKZ_GH_USER", "octocat")
	t.Setenv("GITRAKZ_DB_PATH", filepath.Join(t.TempDir(), "gitrakz.db"))
	t.Setenv("GITRAKZ_HTTP_ADDR", "127.0.0.1:0")

	svc, err := New()
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())

	runErrCh := make(chan error, 1)

	go func() {
		runErrCh <- svc.Run(ctx)
	}()

	require.Eventually(t, func() bool {
		return svc.srv.GetHTTPListenerAddr() != nil
	}, 2*time.Second, 5*time.Millisecond, "http listener never came up")

	baseURL := "http://" + svc.srv.GetHTTPListenerAddr().String()

	t.Run("GET /api/v1/owners on an empty db", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, baseURL+"/api/v1/owners", nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		t.Cleanup(func() { _ = resp.Body.Close() })

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)

		var owners []string
		require.NoError(t, json.Unmarshal(body, &owners))
		assert.Empty(t, owners)
	})

	t.Run("GET / serves the embedded SPA", func(t *testing.T) {
		req, err := http.NewRequestWithContext(
			t.Context(), http.MethodGet, baseURL+"/", nil,
		)
		require.NoError(t, err)

		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)

		t.Cleanup(func() { _ = resp.Body.Close() })

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "gitrakz")
	})

	cancel()

	select {
	case runErr := <-runErrCh:
		assert.NoError(t, runErr)
	case <-time.After(5 * time.Second):
		t.Fatal("service did not stop within the timeout")
	}

	require.NoError(t, svc.Stop(context.Background()))
}
