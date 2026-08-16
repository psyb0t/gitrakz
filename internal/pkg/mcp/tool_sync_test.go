package mcp

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolset_TriggerSync(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		want := ghsync.SyncResult{ReposScanned: 3, EventsUpserted: 12}

		ts := &toolset{deps: Deps{
			SyncController: &fakeSyncController{
				triggerFn: func(context.Context) (ghsync.SyncResult, error) {
					return want, nil
				},
			},
		}}

		_, out, err := ts.triggerSync(t.Context(), nil, triggerSyncInput{})
		require.NoError(t, err)
		assert.Equal(t, want, out)
	})

	t.Run("sync controller error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			SyncController: &fakeSyncController{
				triggerFn: func(context.Context) (ghsync.SyncResult, error) {
					return ghsync.SyncResult{}, assert.AnError
				},
			},
		}}

		_, _, err := ts.triggerSync(t.Context(), nil, triggerSyncInput{})
		require.ErrorIs(t, err, assert.AnError)
	})
}

func TestToolset_GetSyncStatus(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			SyncController: &fakeSyncController{
				statusFn: func(context.Context) (api.SyncStatus, error) {
					return api.SyncStatus{
						InProgress: true, LastSyncedTs: 100,
					}, nil
				},
			},
		}}

		_, out, err := ts.getSyncStatus(t.Context(), nil, getSyncStatusInput{})
		require.NoError(t, err)
		assert.True(t, out.InProgress)
		assert.Equal(t, int64(100), out.LastSyncedTs)
	})

	t.Run("sync controller error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			SyncController: &fakeSyncController{
				statusFn: func(context.Context) (api.SyncStatus, error) {
					return api.SyncStatus{}, assert.AnError
				},
			},
		}}

		_, _, err := ts.getSyncStatus(t.Context(), nil, getSyncStatusInput{})
		require.ErrorIs(t, err, assert.AnError)
	})
}
