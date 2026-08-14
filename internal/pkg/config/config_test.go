package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_Defaults(t *testing.T) {
	t.Setenv("GITRAKZ_GH_USER", "octocat")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Empty(t, cfg.AuthToken)
	assert.Equal(t, "octocat", cfg.GHUser)
	assert.Equal(t, "/data/gitrakz.db", cfg.DBPath)
	assert.Equal(t, "2025-01-01", cfg.SyncSince)
	assert.Equal(t, 30*time.Minute, cfg.SyncInterval)
	assert.Equal(t, 30*time.Minute, cfg.SessionGap)
	assert.Equal(t, 25*time.Minute, cfg.SessionLeadIn)
	assert.Empty(t, cfg.ElelemType)
	assert.Empty(t, cfg.ElelemBaseURL)
	assert.Empty(t, cfg.ElelemModel)
	assert.Empty(t, cfg.ElelemAPIKey)
}

func TestLoad_Overrides(t *testing.T) {
	t.Setenv("GITRAKZ_AUTH_TOKEN", "secret-token")
	t.Setenv("GITRAKZ_GH_USER", "psyb0t")
	t.Setenv("GITRAKZ_DB_PATH", "/tmp/gitrakz-test.db")
	t.Setenv("GITRAKZ_SYNC_SINCE", "2024-06-01")
	t.Setenv("GITRAKZ_SYNC_INTERVAL", "1h")
	t.Setenv("GITRAKZ_SESSION_GAP", "15m")
	t.Setenv("GITRAKZ_SESSION_LEADIN", "10m")
	t.Setenv("GITRAKZ_ELELEM_TYPE", "anthropic")
	t.Setenv("GITRAKZ_ELELEM_BASE_URL", "https://llm.example.com")
	t.Setenv("GITRAKZ_ELELEM_MODEL", "gpt-4")
	t.Setenv("GITRAKZ_ELELEM_API_KEY", "llm-key")

	cfg, err := Load()
	require.NoError(t, err)

	assert.Equal(t, "secret-token", cfg.AuthToken)
	assert.Equal(t, "psyb0t", cfg.GHUser)
	assert.Equal(t, "/tmp/gitrakz-test.db", cfg.DBPath)
	assert.Equal(t, "2024-06-01", cfg.SyncSince)
	assert.Equal(t, time.Hour, cfg.SyncInterval)
	assert.Equal(t, 15*time.Minute, cfg.SessionGap)
	assert.Equal(t, 10*time.Minute, cfg.SessionLeadIn)
	assert.Equal(t, "anthropic", cfg.ElelemType)
	assert.Equal(t, "https://llm.example.com", cfg.ElelemBaseURL)
	assert.Equal(t, "gpt-4", cfg.ElelemModel)
	assert.Equal(t, "llm-key", cfg.ElelemAPIKey)
}

func TestLoad_EmptyGHUserAllowed(t *testing.T) {
	t.Setenv("GITRAKZ_GH_USER", "")

	cfg, err := Load()
	require.NoError(t, err)
	assert.Empty(t, cfg.GHUser)
}

func TestLoad_ParseError(t *testing.T) {
	t.Setenv("GITRAKZ_GH_USER", "octocat")
	t.Setenv("GITRAKZ_SYNC_INTERVAL", "not-a-duration")

	cfg, err := Load()
	require.Error(t, err)
	assert.Empty(t, cfg)
}
