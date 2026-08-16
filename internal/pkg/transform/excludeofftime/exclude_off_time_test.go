package excludeofftime

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// daytimeWindowParams is a 10:00-11:00 non-wrapping off-window.
const daytimeWindowParams = `{
	"offHours": [{"startMinute": 600, "endMinute": 660}]
}`

// wrapWindowParams is a 23:00-06:00 window that wraps past midnight.
// gosec flags this JSON blob based on its digit mix, not an actual
// credential.
//
//nolint:gosec // JSON test fixture, gosec's entropy check false-positives
const wrapWindowParams = `{
	"offHours": [{"startMinute": 1380, "endMinute": 360}]
}`

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("nil params falls back to no off-hours", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())

		p, ok := prim.(*primitive)
		require.True(t, ok)
		assert.Empty(t, p.fallback)
	})

	t.Run("params JSON populates the fallback", func(t *testing.T) {
		t.Parallel()

		raw := []byte(`{"offHours":[{"startMinute":600,"endMinute":660}]}`)

		prim, err := New(raw)
		require.NoError(t, err)

		p, ok := prim.(*primitive)
		require.True(t, ok)
		assert.Equal(t, types.OffHours{
			{StartMinute: 600, EndMinute: 660},
		}, p.fallback)
	})

	t.Run("invalid params JSON errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{not json`))
		require.Error(t, err)
		assert.Contains(
			t, err.Error(), "unmarshal exclude-off-time params",
		)
	})
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name         string
		params       []byte
		form         map[string]any
		sessions     []types.Session
		wantSessions []types.Session
	}{
		{
			name:   "no off-hours configured leaves sessions untouched",
			params: nil,
			sessions: []types.Session{
				{Start: 0, End: 1000, DurationSeconds: 100},
			},
			wantSessions: []types.Session{
				{Start: 0, End: 1000, DurationSeconds: 100},
			},
		},
		{
			name:   "session overlapping a daytime window is reduced",
			params: []byte(daytimeWindowParams),
			sessions: []types.Session{
				// 09:43:20 -> 11:10:00 overlaps the 10:00-11:00 window
				// for exactly 3600s (36000-39600).
				{Start: 35000, End: 40200, DurationSeconds: 10000},
			},
			wantSessions: []types.Session{
				{Start: 35000, End: 40200, DurationSeconds: 6400},
			},
		},
		{
			name:   "session fully outside the window is unchanged",
			params: []byte(daytimeWindowParams),
			sessions: []types.Session{
				{Start: 0, End: 1000, DurationSeconds: 500},
			},
			wantSessions: []types.Session{
				{Start: 0, End: 1000, DurationSeconds: 500},
			},
		},
		{
			name: "midnight-wrapping window computes overlap " +
				"across the boundary",
			params: []byte(wrapWindowParams),
			sessions: []types.Session{
				// 23:00 day0 -> 06:00 day1 wraps; session
				// 22:13:20 day0 -> 02:16:40 day1 overlaps it for
				// exactly 12200s (82800-95000).
				{Start: 80000, End: 95000, DurationSeconds: 20000},
			},
			wantSessions: []types.Session{
				{Start: 80000, End: 95000, DurationSeconds: 7800},
			},
		},
		{
			name: "session inside a wrapped window's early-morning " +
				"tail is fully covered",
			params: []byte(wrapWindowParams),
			sessions: []types.Session{
				// 23:00-06:00 wraps; session 02:00 -> 05:00 day1
				// (93600-104400) lies entirely in the tail the
				// 23:00 day0 occurrence casts into day1, so the
				// full 10800s duration is off-hours.
				{Start: 93600, End: 104400, DurationSeconds: 10800},
			},
			wantSessions: []types.Session{
				{Start: 93600, End: 104400, DurationSeconds: 0},
			},
		},
		{
			name:   "overlap cannot drive duration below zero",
			params: []byte(daytimeWindowParams),
			sessions: []types.Session{
				// overlap is 1000s (36500-37500), well over the
				// session's own 500s duration.
				{Start: 36500, End: 37500, DurationSeconds: 500},
			},
			wantSessions: []types.Session{
				{Start: 36500, End: 37500, DurationSeconds: 0},
			},
		},
		{
			name:   "form off-hours overrides the params fallback",
			params: nil,
			form: map[string]any{
				formOffHoursKey: []any{
					map[string]any{
						"startMinute": 600.0,
						"endMinute":   660.0,
					},
				},
			},
			sessions: []types.Session{
				{Start: 35000, End: 40200, DurationSeconds: 10000},
			},
			wantSessions: []types.Session{
				{Start: 35000, End: 40200, DurationSeconds: 6400},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prim, err := New(tc.params)
			require.NoError(t, err)

			s := &transform.State{Sessions: tc.sessions, Form: tc.form}

			require.NoError(t, prim.Apply(context.Background(), s))
			assert.Equal(t, tc.wantSessions, s.Sessions)
		})
	}

	t.Run("EventCount is left untouched", func(t *testing.T) {
		t.Parallel()

		prim, err := New([]byte(daytimeWindowParams))
		require.NoError(t, err)

		s := &transform.State{
			Sessions: []types.Session{
				{
					Start:           35000,
					End:             40200,
					DurationSeconds: 10000,
					EventCount:      7,
				},
			},
		}

		require.NoError(t, prim.Apply(context.Background(), s))
		assert.Equal(t, 7, s.Sessions[0].EventCount)
	})

	t.Run("unmarshalable form off-hours errors", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Form: map[string]any{formOffHoursKey: make(chan int)},
		}

		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "marshal form off-hours")
	})

	t.Run("malformed form off-hours shape errors", func(t *testing.T) {
		t.Parallel()

		prim, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Form: map[string]any{formOffHoursKey: "not-an-array"},
		}

		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unmarshal form off-hours")
	})
}

func TestWindowOverlap(t *testing.T) {
	t.Parallel()

	t.Run("non-wrapping window", func(t *testing.T) {
		t.Parallel()

		w := types.OffWindow{StartMinute: 600, EndMinute: 660}
		got := windowOverlap(0, w, 35000, 40200)
		assert.Equal(t, int64(3600), got)
	})

	t.Run("wrapping window", func(t *testing.T) {
		t.Parallel()

		// windowOverlap computes only day 0's own head piece here
		// (3600s, 82800-86400); the wrapped tail into day 1 (8600s)
		// is a separate call totalOverlap makes for day 1 — see
		// TestPrimitive_Apply's midnight-wrapping case for the
		// combined 12200s total across both days.
		w := types.OffWindow{StartMinute: 1380, EndMinute: 360}
		got := windowOverlap(0, w, 80000, 95000)
		assert.Equal(t, int64(3600), got)
	})

	t.Run("wrapping window early-morning tail", func(t *testing.T) {
		t.Parallel()

		// 02:00-05:00 on day 0 is fully inside the [0, 21600) tail a
		// 23:00-06:00 window casts into every day's early morning.
		w := types.OffWindow{StartMinute: 1380, EndMinute: 360}
		got := windowOverlap(0, w, 7200, 18000)
		assert.Equal(t, int64(10800), got)
	})

	t.Run("no overlap returns zero", func(t *testing.T) {
		t.Parallel()

		w := types.OffWindow{StartMinute: 600, EndMinute: 660}
		got := windowOverlap(0, w, 0, 1000)
		assert.Equal(t, int64(0), got)
	})
}
