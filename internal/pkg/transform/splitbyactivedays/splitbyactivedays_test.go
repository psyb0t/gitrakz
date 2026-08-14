package splitbyactivedays

import (
	"context"
	"testing"
	"time"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testDelta = 0

const testYear = 2024

func unixAt(
	t *testing.T,
	month, day, hour int,
) int64 {
	t.Helper()

	return time.Date(
		testYear, time.Month(month), day,
		hour, 0, 0, 0, time.UTC,
	).Unix()
}

func TestNew(t *testing.T) {
	p, err := New(nil)
	require.NoError(t, err)
	assert.Equal(t, Name, p.Name())
}

func TestPrimitive_Apply(t *testing.T) {
	t.Run("two distinct UTC days produce two rows", func(t *testing.T) {
		p, err := New(nil)
		require.NoError(t, err)

		day1 := unixAt(t, 1, 15, 9)
		day2 := unixAt(t, 1, 16, 10)

		s := &transform.State{
			Sessions: []types.Session{
				{Start: day1, DurationSeconds: 3600},
				{Start: day2, DurationSeconds: 7200},
			},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		require.Len(t, s.Rows, 2)

		assert.Equal(t, "2024-01-15", s.Rows[0].Key)
		assert.InDelta(t, 3600.0, s.Rows[0].Values[valueKeySeconds], testDelta)
		assert.InDelta(t, 1.0, s.Rows[0].Values[valueKeyHours], testDelta)
		assert.Empty(t, s.Rows[0].Labels)

		assert.Equal(t, "2024-01-16", s.Rows[1].Key)
		assert.InDelta(t, 7200.0, s.Rows[1].Values[valueKeySeconds], testDelta)
		assert.InDelta(t, 2.0, s.Rows[1].Values[valueKeyHours], testDelta)
		assert.Empty(t, s.Rows[1].Labels)
	})

	t.Run("same UTC day sessions merge into one row", func(t *testing.T) {
		p, err := New(nil)
		require.NoError(t, err)

		morning := unixAt(t, 3, 5, 8)
		afternoon := unixAt(t, 3, 5, 14)

		s := &transform.State{
			Sessions: []types.Session{
				{Start: morning, DurationSeconds: 1800},
				{Start: afternoon, DurationSeconds: 900},
			},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		require.Len(t, s.Rows, 1)

		assert.Equal(t, "2024-03-05", s.Rows[0].Key)
		assert.InDelta(t, 2700.0, s.Rows[0].Values[valueKeySeconds], testDelta)
		assert.InDelta(t, 0.75, s.Rows[0].Values[valueKeyHours], testDelta)
		assert.Empty(t, s.Rows[0].Labels)
	})

	t.Run("no sessions produces no rows", func(t *testing.T) {
		p, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{}

		require.NoError(t, p.Apply(context.Background(), s))
		assert.Empty(t, s.Rows)
	})
}
