package server

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ListOwners(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listOwnersFn: func(context.Context) ([]string, error) {
				return []string{"alice", "bob"}, nil
			},
		}})

		resp, err := srv.ListOwners(
			context.Background(), api.ListOwnersRequestObject{},
		)
		require.NoError(t, err)
		require.IsType(t, api.ListOwners200JSONResponse{}, resp)
		assert.Equal(t, api.ListOwners200JSONResponse{"alice", "bob"}, resp)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listOwnersFn: func(context.Context) ([]string, error) {
				return nil, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListOwners(
			context.Background(), api.ListOwnersRequestObject{},
		)
		require.NoError(t, err)
		require.IsType(t, api.ListOwnersdefaultJSONResponse{}, resp)

		got, _ := resp.(api.ListOwnersdefaultJSONResponse)
		assert.Equal(t, 500, got.StatusCode)
		assert.Equal(t, errCodeInternal, got.Body.Code)
	})
}

func TestServer_ListRepos(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listReposFn: func(
				_ context.Context,
				owner string,
			) ([]string, error) {
				assert.Equal(t, "psyb0t", owner)

				return []string{"gitrakz"}, nil
			},
		}})

		resp, err := srv.ListRepos(
			context.Background(),
			api.ListReposRequestObject{
				Params: api.ListReposParams{Owner: "psyb0t"},
			})
		require.NoError(t, err)
		assert.Equal(t, api.ListRepos200JSONResponse{"gitrakz"}, resp)
	})

	t.Run("missing owner", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{}})

		resp, err := srv.ListRepos(
			context.Background(), api.ListReposRequestObject{},
		)
		require.NoError(t, err)
		require.IsType(t, api.ListReposdefaultJSONResponse{}, resp)

		got, _ := resp.(api.ListReposdefaultJSONResponse)
		assert.Equal(t, 400, got.StatusCode)
		assert.Equal(t, errCodeValidationFailed, got.Body.Code)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listReposFn: func(
				context.Context,
				string,
			) ([]string, error) {
				return nil, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListRepos(
			context.Background(),
			api.ListReposRequestObject{
				Params: api.ListReposParams{Owner: "psyb0t"},
			})
		require.NoError(t, err)

		got, ok := resp.(api.ListReposdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestResolvePage(t *testing.T) {
	t.Parallel()

	one, three := 1, 3

	testCases := []struct {
		name string
		in   *int
		want int
	}{
		{"nil defaults to first page", nil, 0},
		{"page 1 is index 0", &one, 0},
		{"page 3 is index 2", &three, 2},
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

	zero, low, high, mid := 0, -5, 9999, 75

	testCases := []struct {
		name string
		in   *int
		want int
	}{
		{"nil defaults", nil, defaultPerPage},
		{"zero clamps to min", &zero, minPerPage},
		{"negative clamps to min", &low, minPerPage},
		{"huge clamps to max", &high, maxPerPage},
		{"within range passes through", &mid, 75},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, resolvePerPage(tc.in))
		})
	}
}

func TestServer_ListTimeline(t *testing.T) {
	t.Parallel()

	t.Run("happy path with filters", func(t *testing.T) {
		t.Parallel()

		owner, repo := "psyb0t", "gitrakz"
		evType := api.Commit
		from, to := int64(100), int64(200)
		page, perPage := 2, 10

		var gotFilter db.TimelineFilter

		srv := New(Deps{Store: &fakeStore{
			queryTimelineFn: func(
				_ context.Context,
				filter db.TimelineFilter,
			) ([]types.Event, bool, error) {
				gotFilter = filter

				return []types.Event{
					{ID: "commit:a/b:1", Owner: owner, Repo: repo},
				}, true, nil
			},
		}})

		resp, err := srv.ListTimeline(
			context.Background(),
			api.ListTimelineRequestObject{
				Params: api.ListTimelineParams{
					Owner:   &owner,
					Repo:    &repo,
					Type:    &evType,
					From:    &from,
					To:      &to,
					Page:    &page,
					PerPage: &perPage,
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListTimeline200JSONResponse)
		require.True(t, ok)
		assert.True(t, got.HasMore)
		require.Len(t, got.Items, 1)

		assert.Equal(t, db.TimelineFilter{
			Owner: owner, Repo: repo, Type: "commit",
			From: from, To: to, Page: 1, PerPage: perPage,
		}, gotFilter)
	})

	t.Run("no params uses defaults", func(t *testing.T) {
		t.Parallel()

		var gotFilter db.TimelineFilter

		srv := New(Deps{Store: &fakeStore{
			queryTimelineFn: func(
				_ context.Context,
				filter db.TimelineFilter,
			) ([]types.Event, bool, error) {
				gotFilter = filter

				return nil, false, nil
			},
		}})

		resp, err := srv.ListTimeline(
			context.Background(), api.ListTimelineRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListTimeline200JSONResponse)
		require.True(t, ok)
		assert.False(t, got.HasMore)
		assert.Empty(t, got.Items)
		assert.Equal(t, 0, gotFilter.Page)
		assert.Equal(t, defaultPerPage, gotFilter.PerPage)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			queryTimelineFn: func(
				context.Context,
				db.TimelineFilter,
			) ([]types.Event, bool, error) {
				return nil, false, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListTimeline(
			context.Background(), api.ListTimelineRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListTimelinedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_ListSessions(t *testing.T) {
	t.Parallel()

	t.Run("happy path paginates the full timeline", func(t *testing.T) {
		t.Parallel()

		owner := "alice"
		calls := 0

		srv := New(Deps{
			Store: &fakeStore{
				queryTimelineFn: func(
					_ context.Context,
					filter db.TimelineFilter,
				) ([]types.Event, bool, error) {
					calls++

					if filter.Page == 0 {
						return []types.Event{{ID: "1"}}, true, nil
					}

					return []types.Event{{ID: "2"}}, false, nil
				},
			},
			Sessionizer: &fakeSessionizer{
				sessionsFn: func(
					_ context.Context,
					timeline types.Timeline,
				) ([]types.Session, error) {
					require.Len(t, timeline, 2)

					return []types.Session{
						{
							Owner:           owner,
							Start:           100,
							End:             200,
							DurationSeconds: 3600,
						},
					}, nil
				},
			},
		})

		resp, err := srv.ListSessions(
			context.Background(),
			api.ListSessionsRequestObject{
				Params: api.ListSessionsParams{Owner: &owner},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, 2, calls)

		got, ok := resp.(api.ListSessions200JSONResponse)
		require.True(t, ok)
		require.Len(t, got.Sessions, 1)
		assert.Equal(t, owner, got.Sessions[0].Owner)
		assert.InDelta(t, 1.0, got.Sessions[0].DurationHours, 0.0001)
		assert.Equal(t, []api.Event{}, got.Sessions[0].Events)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			queryTimelineFn: func(
				context.Context,
				db.TimelineFilter,
			) ([]types.Event, bool, error) {
				return nil, false, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListSessions(
			context.Background(), api.ListSessionsRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListSessionsdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})

	t.Run("sessionizer error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{
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
					return nil, ctxerrors.New("boom")
				},
			},
		})

		resp, err := srv.ListSessions(
			context.Background(), api.ListSessionsRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListSessionsdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}
