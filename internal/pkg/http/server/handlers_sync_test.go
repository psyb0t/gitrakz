package server

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/ghsync"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_TriggerSync(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{SyncController: &fakeSyncController{
			triggerFn: func(context.Context) (ghsync.SyncResult, error) {
				return ghsync.SyncResult{
					ReposScanned:   3,
					EventsUpserted: 42,
				}, nil
			},
		}})

		resp, err := srv.TriggerSync(
			context.Background(), api.TriggerSyncRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.TriggerSync202JSONResponse)
		require.True(t, ok)
		assert.NotEmpty(t, got.JobId)
	})

	t.Run("controller error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{SyncController: &fakeSyncController{
			triggerFn: func(context.Context) (ghsync.SyncResult, error) {
				return ghsync.SyncResult{}, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.TriggerSync(
			context.Background(), api.TriggerSyncRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.TriggerSyncdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_GetSyncStatus(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{SyncController: &fakeSyncController{
			statusFn: func(context.Context) (api.SyncStatus, error) {
				return api.SyncStatus{
					InProgress:   true,
					LastSyncedTs: 123,
					PerOwner:     map[string]any{"alice": 5},
				}, nil
			},
		}})

		resp, err := srv.GetSyncStatus(
			context.Background(), api.GetSyncStatusRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GetSyncStatus200JSONResponse)
		require.True(t, ok)
		assert.True(t, got.InProgress)
		assert.Equal(t, int64(123), got.LastSyncedTs)
	})

	t.Run("controller error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{SyncController: &fakeSyncController{
			statusFn: func(context.Context) (api.SyncStatus, error) {
				return api.SyncStatus{}, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.GetSyncStatus(
			context.Background(), api.GetSyncStatusRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GetSyncStatusdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}
