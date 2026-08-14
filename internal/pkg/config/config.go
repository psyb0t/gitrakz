// Package config loads gitrakz's runtime configuration from environment
// variables prefixed with GITRAKZ_.
package config

import (
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gonfiguration"
)

// Config holds gitrakz's runtime configuration.
type Config struct {
	HTTPAddr  string `default:":8080"            env:"GITRAKZ_HTTP_ADDR"`
	AuthToken string `env:"GITRAKZ_AUTH_TOKEN"`
	GHUser    string `env:"GITRAKZ_GH_USER"`
	DBPath    string `default:"/data/gitrakz.db" env:"GITRAKZ_DB_PATH"`
	SyncSince string `default:"2025-01-01"       env:"GITRAKZ_SYNC_SINCE"`

	SyncInterval  time.Duration `default:"30m" env:"GITRAKZ_SYNC_INTERVAL"`
	SessionGap    time.Duration `default:"30m" env:"GITRAKZ_SESSION_GAP"`
	SessionLeadIn time.Duration `default:"25m" env:"GITRAKZ_SESSION_LEADIN"`

	ElelemType    string `env:"GITRAKZ_ELELEM_TYPE"`
	ElelemBaseURL string `env:"GITRAKZ_ELELEM_BASE_URL"`
	ElelemModel   string `env:"GITRAKZ_ELELEM_MODEL"`
	ElelemAPIKey  string `env:"GITRAKZ_ELELEM_API_KEY"`
}

// Load parses Config from the environment and fails fast if GHUser
// is empty — gitrakz has no target GitHub user to track without it.
func Load() (Config, error) {
	cfg := Config{}

	if err := gonfiguration.Parse(&cfg); err != nil {
		return Config{}, ctxerrors.Wrap(err, "parse config")
	}

	if cfg.GHUser == "" {
		return Config{}, ctxerrors.Wrap(ErrGHUserRequired, "load config")
	}

	return cfg, nil
}
