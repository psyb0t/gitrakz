package server

import (
	"context"

	"github.com/google/uuid"
	"github.com/psyb0t/ctxscope"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// TriggerSync runs one incremental `gh` sync and reports its outcome as
// an accepted job. ghsync.SyncResult carries no id of its own, so a
// fresh one is minted per call purely to satisfy the SyncJob response
// shape.
func (s *Server) TriggerSync(
	ctx context.Context,
	_ api.TriggerSyncRequestObject,
) (api.TriggerSyncResponseObject, error) {
	logger := ctxscope.GetLogger(ctx)

	result, err := s.deps.SyncController.Trigger(ctx)
	if err != nil {
		status, body := respondError(ctx, "trigger sync", err)

		return api.TriggerSyncdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	logger.Info("sync triggered",
		"repos_scanned", result.ReposScanned,
		"events_upserted", result.EventsUpserted,
		"errors", len(result.Errors),
	)

	return api.TriggerSync202JSONResponse{JobId: uuid.NewString()}, nil
}

// GetSyncStatus returns the current sync status.
func (s *Server) GetSyncStatus(
	ctx context.Context,
	_ api.GetSyncStatusRequestObject,
) (api.GetSyncStatusResponseObject, error) {
	syncStatus, err := s.deps.SyncController.Status(ctx)
	if err != nil {
		status, body := respondError(ctx, "get sync status", err)

		return api.GetSyncStatusdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.GetSyncStatus200JSONResponse(syncStatus), nil
}
