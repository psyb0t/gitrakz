package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
)

type listSessionsInput struct {
	Owner string `json:"owner,omitempty" jsonschema:"GitHub owner"`
	From  int64  `json:"from,omitempty"  jsonschema:"unix seconds, lower bound"`
	To    int64  `json:"to,omitempty"    jsonschema:"unix seconds, upper bound"`
}

type listSessionsOutput struct {
	Sessions []types.Session `json:"sessions"`
}

// registerSessionTools adds gitrakz_list_sessions.
func (t *toolset) registerSessionTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameListSessions,
		Description: "Derive per-owner work sessions (clusters of events " +
			"whose inter-event gaps are under the sessionization threshold) " +
			"over the filtered range.",
	}, t.listSessions)
}

func (t *toolset) listSessions(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in listSessionsInput,
) (*mcpsdk.CallToolResult, listSessionsOutput, error) {
	filter := db.TimelineFilter{Owner: in.Owner, From: in.From, To: in.To}

	timeline, err := t.queryFullTimeline(ctx, filter)
	if err != nil {
		return nil, listSessionsOutput{}, err
	}

	sessions, err := t.deps.Sessionizer.Sessions(ctx, timeline)
	if err != nil {
		return nil, listSessionsOutput{}, ctxerrors.Wrap(
			err, "list sessions",
		)
	}

	return nil, listSessionsOutput{Sessions: sessions}, nil
}
