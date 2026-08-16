package mcp

import (
	"context"
	"encoding/json"
	"testing"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDeps returns a Deps wired with fakes that fail the test if called —
// individual test cases override only the fields they exercise.
func testDeps(t *testing.T) Deps {
	t.Helper()

	return Deps{
		Store: &fakeStore{
			listOwnersFn: func(context.Context) ([]string, error) {
				t.Fatal("unexpected ListOwners call")

				return nil, nil
			},
		},
		Engine:         &fakeEngine{},
		Sessionizer:    &fakeSessionizer{},
		SyncController: &fakeSyncController{},
	}
}

func TestNewServer(t *testing.T) {
	t.Parallel()

	srv := NewServer(testDeps(t))
	require.NotNil(t, srv)
}

// TestNewServer_ToolsRegistered connects a client to the server over an
// in-memory transport and verifies every tool the package advertises is
// discoverable and callable end to end (registration, schema generation,
// arg decode, and result encode all exercised together).
func TestNewServer_ToolsRegistered(t *testing.T) {
	t.Parallel()

	deps := Deps{
		Store: &fakeStore{
			listOwnersFn: func(context.Context) ([]string, error) {
				return []string{"octocat"}, nil
			},
		},
		Engine:         &fakeEngine{},
		Sessionizer:    &fakeSessionizer{},
		SyncController: &fakeSyncController{},
	}
	srv := NewServer(deps)

	clientTransport, serverTransport := mcpsdk.NewInMemoryTransports()

	ctx := t.Context()

	serverSession, err := srv.Connect(ctx, serverTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = serverSession.Close()
	})

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "test-client"}, nil)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = clientSession.Close()
	})

	toolsRes, err := clientSession.ListTools(ctx, nil)
	require.NoError(t, err)

	gotNames := make([]string, 0, len(toolsRes.Tools))
	for _, tool := range toolsRes.Tools {
		gotNames = append(gotNames, tool.Name)
	}

	assert.ElementsMatch(t, []string{
		toolNameListOwners,
		toolNameListRepos,
		toolNameListTemplates,
		toolNameGetTemplate,
		toolNameRunTemplate,
		toolNameTriggerSync,
		toolNameGetSyncStatus,
		toolNameListSessions,
		toolNameQueryTimeline,
	}, gotNames)

	callRes, err := clientSession.CallTool(ctx, &mcpsdk.CallToolParams{
		Name: toolNameListOwners,
	})
	require.NoError(t, err)
	require.False(t, callRes.IsError)

	gotContent, err := json.Marshal(callRes.StructuredContent)
	require.NoError(t, err)
	assert.JSONEq(t, `{"owners":["octocat"]}`, string(gotContent))
}

func TestToolset_QueryFullTimeline(t *testing.T) {
	t.Parallel()

	t.Run("paginates until hasMore is false", func(t *testing.T) {
		t.Parallel()

		var gotPages []int

		store := &fakeStore{
			queryTimelineFn: func(
				_ context.Context,
				filter db.TimelineFilter,
			) ([]types.Event, bool, error) {
				gotPages = append(gotPages, filter.Page)

				if filter.Page == 0 {
					return []types.Event{{ID: "e1"}}, true, nil
				}

				return []types.Event{{ID: "e2"}}, false, nil
			},
		}
		ts := &toolset{deps: Deps{Store: store}}

		timeline, err := ts.queryFullTimeline(t.Context(), db.TimelineFilter{})
		require.NoError(t, err)
		assert.Equal(t, types.Timeline{{ID: "e1"}, {ID: "e2"}}, timeline)
		assert.Equal(t, []int{0, 1}, gotPages)
	})

	t.Run("propagates a store error", func(t *testing.T) {
		t.Parallel()

		store := &fakeStore{
			queryTimelineFn: func(
				context.Context,
				db.TimelineFilter,
			) ([]types.Event, bool, error) {
				return nil, false, assert.AnError
			},
		}
		ts := &toolset{deps: Deps{Store: store}}

		_, err := ts.queryFullTimeline(t.Context(), db.TimelineFilter{})
		require.ErrorIs(t, err, assert.AnError)
	})
}
