package mcp

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolvePage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero defaults to page 1 -> offset 0", in: 0, want: 0},
		{name: "negative defaults to page 1 -> offset 0", in: -5, want: 0},
		{name: "explicit page 3 -> offset 2", in: 3, want: 2},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, resolvePage(tc.in))
		})
	}
}

func TestResolvePerPage(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		in   int
		want int
	}{
		{name: "zero defaults", in: 0, want: defaultPerPage},
		{name: "below min clamps up", in: -1, want: minPerPage},
		{name: "above max clamps down", in: 9999, want: maxPerPage},
		{name: "within range passes through", in: 10, want: 10},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, tc.want, resolvePerPage(tc.in))
		})
	}
}

func TestToolset_QueryTimeline(t *testing.T) {
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

					return []types.Event{ev}, true, nil
				},
			},
		}}

		_, out, err := ts.queryTimeline(t.Context(), nil, queryTimelineInput{
			Owner:   "octocat",
			Page:    2,
			PerPage: 10,
		})
		require.NoError(t, err)
		require.Len(t, out.Events, 1)
		assert.Equal(t, "e1", out.Events[0].ID)
		assert.True(t, out.HasMore)
		assert.Equal(t, 1, gotFilter.Page)
		assert.Equal(t, 10, gotFilter.PerPage)
	})

	t.Run("store error", func(t *testing.T) {
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

		_, _, err := ts.queryTimeline(t.Context(), nil, queryTimelineInput{})
		require.ErrorIs(t, err, assert.AnError)
	})
}
