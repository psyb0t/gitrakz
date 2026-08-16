package mcp

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolset_ListOwners(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listOwnersFn: func(context.Context) ([]string, error) {
					return []string{"octocat", "psyb0t"}, nil
				},
			},
		}}

		_, out, err := ts.listOwners(t.Context(), nil, listOwnersInput{})
		require.NoError(t, err)
		assert.Equal(
			t, listOwnersOutput{Owners: []string{"octocat", "psyb0t"}}, out,
		)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listOwnersFn: func(context.Context) ([]string, error) {
					return nil, assert.AnError
				},
			},
		}}

		_, _, err := ts.listOwners(t.Context(), nil, listOwnersInput{})
		require.ErrorIs(t, err, assert.AnError)
	})
}

func TestToolset_ListRepos(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listReposFn: func(
					_ context.Context, owner string,
				) ([]string, error) {
					assert.Equal(t, "octocat", owner)

					return []string{"hello-world"}, nil
				},
			},
		}}

		_, out, err := ts.listRepos(
			t.Context(), nil, listReposInput{Owner: "octocat"},
		)
		require.NoError(t, err)
		assert.Equal(t, listReposOutput{Repos: []string{"hello-world"}}, out)
	})

	t.Run("missing owner is rejected before the store", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{}}

		_, _, err := ts.listRepos(t.Context(), nil, listReposInput{})
		require.ErrorIs(t, err, commerr.ErrValidationFailed)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listReposFn: func(context.Context, string) ([]string, error) {
					return nil, assert.AnError
				},
			},
		}}

		_, _, err := ts.listRepos(
			t.Context(), nil, listReposInput{Owner: "octocat"},
		)
		require.ErrorIs(t, err, assert.AnError)
	})
}
