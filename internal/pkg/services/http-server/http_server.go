// Package httpserver is gitrakz's only servicepack service: it boots the
// whole app — SQLite storage + migrations, the gh sync engine, the
// transform/template engine, the LLM adapters, and an aichteeteapee/serbewr
// HTTP server mounting the generated API under /api/v1/ plus the embedded
// Svelte SPA — and runs a background sync ticker for the lifetime of the
// process.
package httpserver

import (
	"context"
	"errors"
	"net/http"
	"runtime/debug"
	"sync"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/aichteeteapee"
	"github.com/psyb0t/aichteeteapee/serbewr"
	"github.com/psyb0t/commander"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gitrakz/internal/pkg/common/templates"
	ctransform "github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/config"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/engine"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	httpapi "github.com/psyb0t/gitrakz/internal/pkg/http/server"
	mcpserver "github.com/psyb0t/gitrakz/internal/pkg/mcp"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/describework"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/llmstep"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/registry"
)

// ServiceName is this service's registry key, matching the folder name
// convention (http-server -> package httpserver) per wiring-servicepack.md.
const ServiceName = "http-server"

// httpListenAddr is the fixed in-container bind address, not configurable.
// gitrakz ships as a Docker image, so the address that matters is the host
// port published with `-p`.
const httpListenAddr = ":8080"

// Service implements servicemanager.Service. New wires every dependency
// eagerly and fails fast; Run starts the HTTP server and the background
// sync ticker and blocks until ctx is done; Stop tears both down.
type Service struct {
	cfg     config.Config
	store   *db.Store
	syncCtl httpapi.SyncController
	srv     *serbewr.Server
	router  *serbewr.Router
}

// New builds every dependency the service needs (config, storage,
// migrations, the gh sync engine, the transform/template engine, the LLM
// adapters, and the HTTP server) and fails fast on the first error — no
// partially-wired Service is ever returned.
func New() (*Service, error) {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "load config")
	}

	store, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "open db")
	}

	if err := SeedBuiltinTemplates(ctx, store); err != nil {
		if closeErr := store.Close(); closeErr != nil {
			ctxscope.GetLogger(ctx).Error(
				"close db store after seed failure",
				"service", ServiceName,
				"err", closeErr,
			)
		}

		return nil, err
	}

	svc, err := wire(cfg, store)
	if err != nil {
		if closeErr := store.Close(); closeErr != nil {
			ctxscope.GetLogger(ctx).Error(
				"close db store after wiring failure",
				"service", ServiceName,
				"err", closeErr,
			)
		}

		return nil, err
	}

	return svc, nil
}

// SeedBuiltinTemplates upserts every templates.Builtins() entry into store.
// SaveTemplate is an idempotent upsert (clause.OnConflict on id,
// UpdateAll), so calling this on every boot is safe and keeps a running
// service's built-ins in sync with whatever version of the templates
// package it ships. A seed failure aborts boot, per New's fail-fast
// contract. Exported so the `gitrakz mcp` stdio command (see cmd/) seeds
// the same built-ins the HTTP service would.
func SeedBuiltinTemplates(ctx context.Context, store *db.Store) error {
	logger := ctxscope.GetLogger(ctx)
	builtins := templates.Builtins()

	for _, tmpl := range builtins {
		if err := store.SaveTemplate(ctx, tmpl); err != nil {
			return ctxerrors.Wrapf(err, "seed built-in template %q", tmpl.ID)
		}

		logger.Debug(
			"seeded built-in template",
			"service", ServiceName,
			"template_id", tmpl.ID,
		)
	}

	logger.Info(
		"seeded built-in templates",
		"service", ServiceName,
		"count", len(builtins),
	)

	return nil
}

// BuildDeps wires every dependency downstream of store: the gh sync
// engine, the transform registry (the 7 deterministic primitives plus the
// LLM-backed describe-work and llm-step primitives), the template engine,
// and the LLM adapters — returning the shared httpapi.Deps both the
// generated HTTP API (wire, below) and the MCP server (wire, below, and the
// `gitrakz mcp` stdio command in cmd/) are built from. Exported so cmd/ can
// wire the exact same production stack without duplicating this logic.
func BuildDeps(cfg config.Config, store *db.Store) (httpapi.Deps, error) {
	cmd := commander.New()
	ghClient := ghsync.NewCommanderGHClient(cmd)
	syncer := ghsync.NewSyncer(ghClient, store, cfg.GHUser)
	syncCtl := newSyncController(syncer)

	llmRT := newLLMRuntime(newLLMClient(cfg), cfg.ElelemType, store)

	cache := newDescribeWorkCacheStore(store)
	describer := newLLMDescriber(llmRT)
	differ := newCommanderGHDiffer(cmd)

	reg := buildTransformRegistry(llmRT, cache, describer, differ)
	eng := engine.NewEngine(reg)

	sess, err := newSessionizer(cfg.SessionGap, cfg.SessionLeadIn)
	if err != nil {
		return httpapi.Deps{}, ctxerrors.Wrap(err, "build sessionizer")
	}

	composer := newLLMComposer(llmRT)

	return httpapi.Deps{
		Store:          store,
		Engine:         eng,
		Sessionizer:    sess,
		SyncController: syncCtl,
		LLMComposer:    composer,
		LLMModelLister: llmRT,
	}, nil
}

// newMCPHandler builds gitrakz's MCP server over deps and wraps it as a
// streamable-HTTP http.Handler. The same *mcpsdk.Server instance is reused
// across every request/session — it has no per-request state of its own,
// only the shared Deps.
func newMCPHandler(deps httpapi.Deps) http.Handler {
	mcpSrv := mcpserver.NewServer(mcpserver.Deps{
		Store:          deps.Store,
		Engine:         deps.Engine,
		Sessionizer:    deps.Sessionizer,
		SyncController: deps.SyncController,
	})

	return mcpsdk.NewStreamableHTTPHandler(
		func(*http.Request) *mcpsdk.Server { return mcpSrv }, nil,
	)
}

// wire builds the generated API's Server and gitrakz's MCP server from a
// shared BuildDeps call, and the serbewr router mounting both alongside the
// embedded SPA.
func wire(cfg config.Config, store *db.Store) (*Service, error) {
	deps, err := BuildDeps(cfg, store)
	if err != nil {
		return nil, err
	}

	apiServer := httpapi.New(deps)

	strictHandler := api.NewStrictHandler(apiServer, nil)
	// The OpenAPI paths are version-less; BaseURL applies the /api/v1 prefix
	// so registered routes are "/api/v1/...", matching apiMountPattern.
	apiHandler := api.HandlerWithOptions(
		strictHandler, api.StdHTTPServerOptions{BaseURL: apiBaseURL},
	)

	mcpHandler := newMCPHandler(deps)

	spaHandler, err := newSPAHandler()
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build spa handler")
	}

	router := newRouter(cfg.AuthToken, apiHandler, mcpHandler, spaHandler)

	srv, err := serbewr.NewWithConfig(serverConfig())
	if err != nil {
		return nil, ctxerrors.Wrap(err, "new http server")
	}

	return &Service{
		cfg:     cfg,
		store:   store,
		syncCtl: deps.SyncController,
		srv:     srv,
		router:  router,
	}, nil
}

// buildTransformRegistry returns registry.Default() (the 7 deterministic
// primitives — sessionize, exclude-off-time, split-by-active-days,
// group-by, aggregate, rate, passthrough — under their Name consts) plus
// the two primitives with real dependencies (cache/LLM/gh), describe-work
// and llm, added on top via closures over this service's wired adapters.
func buildTransformRegistry(
	llmRT *llmRuntime,
	cache describework.CacheStore,
	describer describework.LLMClient,
	differ describework.GHDiffer,
) *ctransform.Registry {
	reg := registry.Default()
	reg.Register(describework.Name, func(
		params []byte,
	) (ctransform.Primitive, error) {
		return describework.New(cache, describer, differ, params)
	})

	stepClient := newLLMStepClient(llmRT)

	reg.Register(llmstep.Name, func(
		params []byte,
	) (ctransform.Primitive, error) {
		return llmstep.New(stepClient, params)
	})

	return reg
}

// serverConfig builds serbewr's Config from the fixed httpListenAddr plus
// aichteeteapee's own package defaults for everything else. Built directly,
// bypassing serbewr's HTTP_SERVER_* env vars, so they can't override
// gitrakz's fixed bind address.
func serverConfig() serbewr.Config {
	return serbewr.Config{
		ListenAddress:       httpListenAddr,
		ReadTimeout:         aichteeteapee.DefaultHTTPServerReadTimeout,
		ReadHeaderTimeout:   aichteeteapee.DefaultHTTPServerReadHeaderTimeout,
		WriteTimeout:        aichteeteapee.DefaultHTTPServerWriteTimeout,
		IdleTimeout:         aichteeteapee.DefaultHTTPServerIdleTimeout,
		MaxHeaderBytes:      aichteeteapee.DefaultHTTPServerMaxHeaderBytes,
		ShutdownTimeout:     aichteeteapee.DefaultHTTPServerShutdownTimeout,
		ServiceName:         ServiceName,
		FileUploadMaxMemory: aichteeteapee.DefaultFileUploadMaxMemory,
	}
}

// Name returns this service's registry key.
func (s *Service) Name() string {
	return ServiceName
}

// Run starts the HTTP server and the background sync ticker, and blocks
// until ctx is done or the server fails. serbewr.Server.Start already
// performs its own graceful shutdown internally once ctx is done (or Stop
// is called), so Run does not repeat that step.
func (s *Service) Run(ctx context.Context) error {
	logger := ctxscope.GetLogger(ctx)
	logger.Info(
		"starting service",
		"service", ServiceName,
		"addr", httpListenAddr,
	)

	var wg sync.WaitGroup

	wg.Go(func() {
		s.runSyncTicker(ctx)
	})

	err := s.srv.Start(ctx, s.router)

	wg.Wait()

	if err != nil && !errors.Is(err, context.Canceled) {
		return ctxerrors.Wrap(err, "run http server")
	}

	logger.Info("service stopped", "service", ServiceName)

	return nil
}

// Stop shuts down the HTTP server (a no-op if Run's own shutdown already
// ran — serbewr.Server.Stop is idempotent) and closes the db store.
func (s *Service) Stop(ctx context.Context) error {
	logger := ctxscope.GetLogger(ctx)
	logger.Info("stopping service", "service", ServiceName)

	if err := s.srv.Stop(ctx); err != nil {
		logger.Error(
			"stop http server",
			"service", ServiceName,
			"err", err,
		)
	}

	if err := s.store.Close(); err != nil {
		return ctxerrors.Wrap(err, "close db store")
	}

	return nil
}

// runSyncTicker calls syncCtl.Trigger every cfg.SyncInterval until ctx is
// done. cfg.SyncInterval <= 0 disables the ticker — manual /api/v1/sync
// triggers still work. A panic here is recovered and logged, per the
// goroutine panic-recovery rule.
func (s *Service) runSyncTicker(ctx context.Context) {
	logger := ctxscope.GetLogger(ctx)

	defer func() {
		if r := recover(); r != nil {
			logger.Error(
				"sync ticker panicked",
				"service", ServiceName,
				"panic", r,
				"stack", string(debug.Stack()),
			)
		}
	}()

	if s.cfg.SyncInterval <= 0 {
		logger.Info("background sync disabled", "service", ServiceName)

		return
	}

	ticker := time.NewTicker(s.cfg.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("sync ticker stopping", "service", ServiceName)

			return
		case <-ticker.C:
			s.runScheduledSync(ctx)
		}
	}
}

// runScheduledSync runs one sync through syncCtl (so /api/v1/sync/status
// reflects scheduled runs too) and logs the outcome.
func (s *Service) runScheduledSync(ctx context.Context) {
	logger := ctxscope.GetLogger(ctx)
	logger.Debug("scheduled sync starting", "service", ServiceName)

	result, err := s.syncCtl.Trigger(ctx)
	if err != nil {
		logger.Error(
			"scheduled sync failed",
			"service", ServiceName,
			"err", err,
		)

		return
	}

	logger.Info(
		"scheduled sync completed",
		"service", ServiceName,
		"repos_scanned", result.ReposScanned,
		"events_upserted", result.EventsUpserted,
		"errors", len(result.Errors),
	)
}
