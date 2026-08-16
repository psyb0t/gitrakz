// Package mcp exposes gitrakz's capabilities as MCP (Model Context
// Protocol) tools, so external MCP clients (Claude Code and friends) can
// drive gitrakz directly — list owners/repos, inspect and run templates,
// trigger/inspect syncs, and read the timeline/sessions.
//
// Every tool wraps the same small Store/Engine/Sessionizer/SyncController
// surfaces internal/pkg/http/server's handlers are built against — it never
// reimplements domain logic, only decodes tool arguments and encodes tool
// results. NewServer is transport-agnostic: the caller mounts the returned
// *mcp.Server on the streamable HTTP transport (see
// internal/pkg/services/http-server) or runs it over stdio (see the
// `gitrakz mcp` command in cmd/).
package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// Store is the persistence surface the MCP tools need. Mirrors a strict
// subset of internal/pkg/http/server.Store's method set, field-for-field,
// so *db.Store satisfies it with no adaptation.
type Store interface {
	ListOwners(ctx context.Context) ([]string, error)
	ListRepos(ctx context.Context, owner string) ([]string, error)
	QueryTimeline(
		ctx context.Context,
		filter db.TimelineFilter,
	) ([]types.Event, bool, error)
	ListTemplates(ctx context.Context) ([]template.Template, error)
	GetTemplate(ctx context.Context, id string) (template.Template, error)
}

// Engine runs a template's transform pipeline over a selected timeline and
// renders it into a display Document. Mirrors
// internal/pkg/http/server.Engine.
type Engine interface {
	Run(
		ctx context.Context,
		tmpl template.Template,
		timeline types.Timeline,
		form types.FormValues,
	) (blocks.Document, error)
}

// Sessionizer derives per-owner work sessions from a raw timeline. Mirrors
// internal/pkg/http/server.Sessionizer.
type Sessionizer interface {
	Sessions(
		ctx context.Context,
		timeline types.Timeline,
	) ([]types.Session, error)
}

// SyncController drives, and reports on, the background `gh` sync. Mirrors
// internal/pkg/http/server.SyncController.
type SyncController interface {
	Trigger(ctx context.Context) (ghsync.SyncResult, error)
	Status(ctx context.Context) (api.SyncStatus, error)
}

// Deps collects every dependency the MCP tools need.
type Deps struct {
	Store          Store
	Engine         Engine
	Sessionizer    Sessionizer
	SyncController SyncController
}

// toolset holds Deps behind the tool handler methods NewServer registers.
type toolset struct {
	deps Deps
}

// NewServer builds an MCP server exposing gitrakz's capabilities as tools,
// backed by deps. The returned *mcp.Server has no transport of its own —
// the caller connects it over stdio (mcp.StdioTransport) or mounts it on
// mcp.NewStreamableHTTPHandler for HTTP.
func NewServer(deps Deps) *mcpsdk.Server {
	srv := mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: serverVersion},
		nil,
	)

	ts := &toolset{deps: deps}
	ts.registerOwnerTools(srv)
	ts.registerTemplateTools(srv)
	ts.registerRunTemplateTool(srv)
	ts.registerSyncTools(srv)
	ts.registerSessionTools(srv)
	ts.registerTimelineTool(srv)

	return srv
}

// queryFullTimeline pulls every event matching filter, across as many
// Store.QueryTimeline pages as it takes, for tools that need the whole
// filtered range rather than one page (RunTemplate, ListSessions). Mirrors
// internal/pkg/http/server's identically-named helper.
func (t *toolset) queryFullTimeline(
	ctx context.Context,
	filter db.TimelineFilter,
) (types.Timeline, error) {
	filter.Page = 0
	filter.PerPage = runQueryPerPage

	var all types.Timeline

	for {
		events, hasMore, err := t.deps.Store.QueryTimeline(ctx, filter)
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
