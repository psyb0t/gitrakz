package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
)

// defaultPage / defaultPerPage / minPerPage / maxPerPage mirror
// internal/pkg/http/server's identically-named constants (in turn mirroring
// the PageQueryParam / PerPageQueryParam defaults+bounds in api/api.yml).
const (
	defaultPage    = 1
	defaultPerPage = 50
	minPerPage     = 1
	maxPerPage     = 200
)

type queryTimelineInput struct {
	Owner string `json:"owner,omitempty" jsonschema:"GitHub owner"`
	Repo  string `json:"repo,omitempty"  jsonschema:"repo name"`
	// Type is one of: commit, pr, review, issue, release.
	Type string `json:"type,omitempty" jsonschema:"event type"`
	From int64  `json:"from,omitempty" jsonschema:"unix seconds, from"`
	To   int64  `json:"to,omitempty"   jsonschema:"unix seconds, to"`
	// Page is the 1-indexed page number, default 1.
	Page int `json:"page,omitempty" jsonschema:"page number"`
	// PerPage is the page size, 1-200, default 50.
	PerPage int `json:"perPage,omitempty" jsonschema:"page size"`
}

type queryTimelineOutput struct {
	Events  []eventDTO `json:"events"`
	HasMore bool       `json:"hasMore"`
}

// registerTimelineTool adds gitrakz_query_timeline.
func (t *toolset) registerTimelineTool(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameQueryTimeline,
		Description: "Return one page of the filtered, newest-first event " +
			"timeline.",
	}, t.queryTimeline)
}

// resolvePage converts the 1-indexed, optional tool-input page number into
// the 0-indexed page db.TimelineFilter expects, defaulting/clamping to 1.
func resolvePage(page int) int {
	if page <= 0 {
		page = defaultPage
	}

	return page - 1
}

// resolvePerPage defaults/clamps the optional tool-input perPage into
// [minPerPage, maxPerPage].
func resolvePerPage(perPage int) int {
	if perPage == 0 {
		perPage = defaultPerPage
	}

	if perPage < minPerPage {
		perPage = minPerPage
	}

	if perPage > maxPerPage {
		perPage = maxPerPage
	}

	return perPage
}

func (t *toolset) queryTimeline(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in queryTimelineInput,
) (*mcpsdk.CallToolResult, queryTimelineOutput, error) {
	filter := db.TimelineFilter{
		Owner:   in.Owner,
		Repo:    in.Repo,
		Type:    in.Type,
		From:    in.From,
		To:      in.To,
		Page:    resolvePage(in.Page),
		PerPage: resolvePerPage(in.PerPage),
	}

	events, hasMore, err := t.deps.Store.QueryTimeline(ctx, filter)
	if err != nil {
		return nil, queryTimelineOutput{}, ctxerrors.Wrap(
			err, "query timeline",
		)
	}

	return nil, queryTimelineOutput{
		Events:  eventsToDTO(events),
		HasMore: hasMore,
	}, nil
}
