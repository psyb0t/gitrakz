package mcp

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolset_ListSessions(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var gotFilter db.TimelineFilter

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				queryTimelineFn: func(
					_ context.Context,
					filter db.TimelineFilter,
				) ([]types.Event, bool, error) {
					gotFilter = filter

					ev := types.Event{ID: "e1", Owner: "octocat"}

					return []types.Event{ev}, false, nil
				},
			},
			Sessionizer: &fakeSessionizer{
				sessionsFn: func(
					_ context.Context,
					timeline types.Timeline,
				) ([]types.Session, error) {
					want := types.Timeline{{ID: "e1", Owner: "octocat"}}
					assert.Equal(t, want, timeline)

					sess := types.Session{Owner: "octocat", EventCount: 1}

					return []types.Session{sess}, nil
				},
			},
		}}

		_, out, err := ts.listSessions(t.Context(), nil, listSessionsInput{
			Owner: "octocat",
		})
		require.NoError(t, err)

		want := []types.Session{{Owner: "octocat", EventCount: 1}}
		assert.Equal(t, want, out.Sessions)
		assert.Equal(t, "octocat", gotFilter.Owner)
	})

	t.Run("timeline query error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				queryTimelineFn: func(
					context.Context,
					db.TimelineFilter,
				) ([]types.Event, bool, error) {
					return nil, false, assert.AnError
				},
			},
		}}

		_, _, err := ts.listSessions(t.Context(), nil, listSessionsInput{})
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("sessionizer error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				queryTimelineFn: func(
					context.Context,
					db.TimelineFilter,
				) ([]types.Event, bool, error) {
					return nil, false, nil
				},
			},
			Sessionizer: &fakeSessionizer{
				sessionsFn: func(
					context.Context,
					types.Timeline,
				) ([]types.Session, error) {
					return nil, assert.AnError
				},
			},
		}}

		_, _, err := ts.listSessions(t.Context(), nil, listSessionsInput{})
		require.ErrorIs(t, err, assert.AnError)
	})
}
