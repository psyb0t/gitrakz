// Package server implements gitrakz's HTTP handlers as the generated
// api.StrictServerInterface. Every external dependency (storage, the
// transform engine, sessionization, the gh sync controller, and the LLM
// template composer) is expressed as a small interface owned by this
// package, so Server is fully unit-testable without a real DB, LLM, or
// gh CLI. Production wiring (a later chunk, the "service" layer) supplies
// the concrete implementations.
package server

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

const (
	// defaultPage / defaultPerPage / minPerPage / maxPerPage mirror the
	// PageQueryParam / PerPageQueryParam defaults+bounds in api/api.yml.
	defaultPage    = 1
	defaultPerPage = 50
	minPerPage     = 1
	maxPerPage     = 200

	// runQueryPerPage is the page size used internally whenever a handler
	// needs an entire filtered timeline instead of one paginated page
	// (RunTemplate, ExportDocument, ListSessions). queryFullTimeline
	// calls Store.QueryTimeline repeatedly, one page at a time, until
	// hasMore is false, so this only bounds how many round trips that
	// takes — it never truncates the result.
	runQueryPerPage = 500
)

// Store is the persistence surface the HTTP handlers need. *db.Store
// satisfies it in production; tests supply an in-memory fake.
//
//nolint:dupl // fakeStore mirrors this interface field-for-field
type Store interface {
	ListOwners(ctx context.Context) ([]string, error)
	ListRepos(ctx context.Context, owner string) ([]string, error)
	QueryTimeline(
		ctx context.Context,
		filter db.TimelineFilter,
	) ([]types.Event, bool, error)
	ListTemplates(ctx context.Context) ([]template.Template, error)
	GetTemplate(ctx context.Context, id string) (template.Template, error)
	SaveTemplate(ctx context.Context, tmpl template.Template) error
	DeleteTemplate(ctx context.Context, id string) error
	GetLLMSettings(ctx context.Context) (db.LLMSettings, error)
	SaveLLMSettings(ctx context.Context, settings db.LLMSettings) error
}

// Engine runs a template's transform pipeline over a selected timeline
// and renders it into a display Document. *engine.Engine satisfies it in
// production.
type Engine interface {
	Run(
		ctx context.Context,
		tmpl template.Template,
		timeline types.Timeline,
		form types.FormValues,
	) (blocks.Document, error)
}

// Sessionizer derives per-owner work sessions from a raw timeline. The
// service layer wires this to the sessionize transform primitive.
type Sessionizer interface {
	Sessions(
		ctx context.Context,
		timeline types.Timeline,
	) ([]types.Session, error)
}

// SyncController drives, and reports on, the background `gh` sync. The
// service layer wires this to the running syncer.
type SyncController interface {
	Trigger(ctx context.Context) (ghsync.SyncResult, error)
	Status(ctx context.Context) (api.SyncStatus, error)
}

// LLMComposer drafts a new template from a natural-language description.
// The service layer wires this to elelem.
type LLMComposer interface {
	GenerateTemplate(
		ctx context.Context,
		description string,
	) (template.Template, error)
}

// LLMModelLister lists the provider's available models with their capability
// flags. The service layer wires this to elelem.
type LLMModelLister interface {
	ListModels(ctx context.Context) ([]api.LLMModel, error)
}

// Deps collects every dependency Server needs to satisfy
// api.StrictServerInterface.
type Deps struct {
	Store          Store
	Engine         Engine
	Sessionizer    Sessionizer
	SyncController SyncController
	LLMComposer    LLMComposer
	LLMModelLister LLMModelLister
}

// Server implements api.StrictServerInterface entirely against the
// interfaces in Deps, so it never talks to a real DB/LLM/gh CLI directly.
type Server struct {
	deps Deps
}

// New builds a Server backed by deps.
func New(deps Deps) *Server {
	return &Server{deps: deps}
}

var _ api.StrictServerInterface = (*Server)(nil)

// queryFullTimeline pulls every event matching filter, across as many
// Store.QueryTimeline pages as it takes, for handlers that need the
// whole filtered range rather than one paginated page (RunTemplate,
// ExportDocument, ListSessions).
func (s *Server) queryFullTimeline(
	ctx context.Context,
	filter db.TimelineFilter,
) (types.Timeline, error) {
	filter.Page = 0
	filter.PerPage = runQueryPerPage

	var all types.Timeline

	for {
		events, hasMore, err := s.deps.Store.QueryTimeline(ctx, filter)
		if err != nil {
			return nil, ctxerrors.Wrap(err, "query timeline page")
		}

		all = append(all, events...)

		if !hasMore {
			break
		}

		filter.Page++
	}

	return all, nil
}
