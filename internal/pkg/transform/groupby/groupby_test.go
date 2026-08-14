package groupby

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("defaults by to owner", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, ByOwner, p.by)
	})

	t.Run("accepts every known by value", func(t *testing.T) {
		t.Parallel()

		for _, by := range []By{ByOwner, ByRepo, ByType, ByBranch, ByDay} {
			rawParams := []byte(`{"by":"` + string(by) + `"}`)

			prim, err := New(rawParams)
			require.NoError(t, err)

			p, ok := prim.(primitive)
			require.True(t, ok)
			assert.Equal(t, by, p.by)
		}
	})

	t.Run("unknown by errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{"by":"nope"}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownBy)
		assert.Contains(t, err.Error(), "nope")
	})

	t.Run("malformed params errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{`))
		require.Error(t, err)
	})
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	timeline := types.Timeline{
		{
			Owner:     "alice",
			Repo:      "gitrakz",
			Type:      types.EventTypeCommit,
			TS:        1700000000,
			Branch:    "main",
			Additions: 10,
			Deletions: 2,
		},
		{
			Owner:     "alice",
			Repo:      "gitrakz",
			Type:      types.EventTypePR,
			TS:        1700003600,
			Branch:    "feature",
			Additions: 5,
			Deletions: 1,
		},
		{
			Owner:     "bob",
			Repo:      "gitrakz",
			Type:      types.EventTypeCommit,
			TS:        1700007200,
			Branch:    "main",
			Additions: 3,
			Deletions: 0,
		},
	}

	t.Run("groups by owner", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 2)

		assert.Equal(t, "alice", s.Rows[0].Key)
		assert.Equal(t, map[string]float64{
			valueKeyCount:     2,
			valueKeyAdditions: 15,
			valueKeyDeletions: 3,
		}, s.Rows[0].Values)
		assert.Empty(t, s.Rows[0].Labels)

		assert.Equal(t, "bob", s.Rows[1].Key)
		assert.Equal(t, map[string]float64{
			valueKeyCount:     1,
			valueKeyAdditions: 3,
			valueKeyDeletions: 0,
		}, s.Rows[1].Values)
	})

	t.Run("groups by type", func(t *testing.T) {
		t.Parallel()

		prim, err := New([]byte(`{"by":"type"}`))
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 2)
		assert.Equal(t, "commit", s.Rows[0].Key)
		assert.Equal(t, float64(2), s.Rows[0].Values[valueKeyCount])
		assert.Equal(t, "pr", s.Rows[1].Key)
		assert.Equal(t, float64(1), s.Rows[1].Values[valueKeyCount])
	})

	t.Run("groups by day in UTC", func(t *testing.T) {
		t.Parallel()

		const (
			day1Start = 1699920000 // 2023-11-14T00:00:00Z
			day1End   = 1700006399 // 2023-11-14T23:59:59Z
			day2Start = 1700006400 // 2023-11-15T00:00:00Z
		)

		dayTimeline := types.Timeline{
			{TS: day1Start},
			{TS: day1End},
			{TS: day2Start},
		}

		prim, err := New([]byte(`{"by":"day"}`))
		require.NoError(t, err)

		s := &transform.State{Timeline: dayTimeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 2)
		assert.Equal(t, "2023-11-14", s.Rows[0].Key)
		assert.Equal(t, float64(2), s.Rows[0].Values[valueKeyCount])
		assert.Equal(t, "2023-11-15", s.Rows[1].Key)
		assert.Equal(t, float64(1), s.Rows[1].Values[valueKeyCount])
	})

	t.Run("empty timeline produces no rows", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{}
		require.NoError(t, prim.Apply(context.Background(), s))
		assert.Empty(t, s.Rows)
	})
}
