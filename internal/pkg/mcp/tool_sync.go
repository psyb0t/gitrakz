package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

type triggerSyncInput struct{}

type getSyncStatusInput struct{}

// registerSyncTools adds gitrakz_trigger_sync and gitrakz_get_sync_status.
func (t *toolset) registerSyncTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameTriggerSync,
		Description: "Trigger one incremental `gh` sync and return its " +
			"outcome (repos scanned, events upserted, per-repo errors).",
	}, t.triggerSync)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolNameGetSyncStatus,
		Description: "Get the current background sync status.",
	}, t.getSyncStatus)
}

func (t *toolset) triggerSync(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	_ triggerSyncInput,
) (*mcpsdk.CallToolResult, ghsync.SyncResult, error) {
	result, err := t.deps.SyncController.Trigger(ctx)
	if err != nil {
		return nil, ghsync.SyncResult{}, ctxerrors.Wrap(err, "trigger sync")
	}

	return nil, result, nil
}

func (t *toolset) getSyncStatus(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	_ getSyncStatusInput,
) (*mcpsdk.CallToolResult, api.SyncStatus, error) {
	status, err := t.deps.SyncController.Status(ctx)
	if err != nil {
		return nil, api.SyncStatus{}, ctxerrors.Wrap(
			err, "get sync status",
		)
	}

	return nil, status, nil
}
