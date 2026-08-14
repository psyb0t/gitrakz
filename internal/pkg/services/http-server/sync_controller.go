package httpserver

import (
	"context"
	"sync"
	"time"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// PerOwner summary keys. ghsync.Syncer.Sync only returns an aggregate
// SyncResult (repos scanned, events upserted, per-repo error strings) —
// nothing broken down by owner — so PerOwner carries that aggregate under
// these descriptive keys rather than a genuine per-owner breakdown, which
// would need a change to ghsync.Syncer's return shape (out of this
// service's scope; ghsync is DONE per the wiring brief).
const (
	syncSummaryKeyReposScanned   = "reposScanned"
	syncSummaryKeyEventsUpserted = "eventsUpserted"
	syncSummaryKeyErrors         = "errors"
)

// syncController implements server.SyncController: it wraps a
// *ghsync.Syncer, guarding the last SyncResult + last-run time with a
// mutex so Trigger and Status are both safe to call from concurrent HTTP
// requests and the background sync ticker.
type syncController struct {
	syncer *ghsync.Syncer

	mu           sync.Mutex
	inProgress   bool
	lastResult   ghsync.SyncResult
	lastSyncedTS int64
}

func newSyncController(syncer *ghsync.Syncer) *syncController {
	return &syncController{syncer: syncer}
}

// Trigger runs one incremental sync, recording its result and completion
// time for Status. Returns commerr.ErrConflict if a sync is already
// running.
func (c *syncController) Trigger(
	ctx context.Context,
) (ghsync.SyncResult, error) {
	c.mu.Lock()

	if c.inProgress {
		c.mu.Unlock()

		return ghsync.SyncResult{}, ctxerrors.Wrap(
			commerr.ErrConflict, "sync already in progress",
		)
	}

	c.inProgress = true
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.inProgress = false
		c.mu.Unlock()
	}()

	result, err := c.syncer.Sync(ctx)
	if err != nil {
		return ghsync.SyncResult{}, ctxerrors.Wrap(err, "sync")
	}

	c.mu.Lock()
	c.lastResult = result
	c.lastSyncedTS = time.Now().Unix()
	c.mu.Unlock()

	return result, nil
}

// Status returns the recorded sync state mapped onto the api.SyncStatus
// shape GetSyncStatus responds with.
func (c *syncController) Status(_ context.Context) (api.SyncStatus, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	return api.SyncStatus{
		InProgress:   c.inProgress,
		LastSyncedTs: c.lastSyncedTS,
		PerOwner: map[string]any{
			syncSummaryKeyReposScanned:   c.lastResult.ReposScanned,
			syncSummaryKeyEventsUpserted: c.lastResult.EventsUpserted,
			syncSummaryKeyErrors:         c.lastResult.Errors,
		},
	}, nil
}
