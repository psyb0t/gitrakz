package aggregate

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const hoursField = "hours"

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("valid params default key", func(t *testing.T) {
		t.Parallel()

		prim, err := New([]byte(`{"op":"sum","field":"hours"}`))
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())
	})

	t.Run("valid params custom key", func(t *testing.T) {
		t.Parallel()

		prim, err := New(
			[]byte(`{"op":"sum","field":"hours","key":"grand_total"}`),
		)
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())
	})

	t.Run("unknown op errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{"op":"median","field":"hours"}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidParams)
	})

	t.Run("empty field errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{"op":"sum","field":""}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrInvalidParams)
	})

	t.Run("invalid json errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`not json`))
		require.Error(t, err)
	})
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	baseRows := []transform.Row{
		{Key: "a", Values: map[string]float64{hoursField: 2}},
		{Key: "b", Values: map[string]float64{hoursField: 3}},
	}

	testCases := []struct {
		name    string
		params  string
		rows    []transform.Row
		wantRow transform.Row
	}{
		{
			name:   "sum",
			params: `{"op":"sum","field":"hours"}`,
			rows:   baseRows,
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 5},
			},
		},
		{
			name:   "avg",
			params: `{"op":"avg","field":"hours"}`,
			rows:   baseRows,
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 2.5},
			},
		},
		{
			name:   "min",
			params: `{"op":"min","field":"hours"}`,
			rows:   baseRows,
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 2},
			},
		},
		{
			name:   "max",
			params: `{"op":"max","field":"hours"}`,
			rows:   baseRows,
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 3},
			},
		},
		{
			name:   "custom key",
			params: `{"op":"sum","field":"hours","key":"grand_total"}`,
			rows:   baseRows,
			wantRow: transform.Row{
				Key:    "grand_total",
				Values: map[string]float64{hoursField: 5},
			},
		},
		{
			name:   "rows lacking field are skipped",
			params: `{"op":"sum","field":"hours"}`,
			rows: []transform.Row{
				{Key: "a", Values: map[string]float64{hoursField: 2}},
				{Key: "b", Values: map[string]float64{"dollars": 100}},
			},
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 2},
			},
		},
		{
			name:   "avg over zero contributing rows is zero",
			params: `{"op":"avg","field":"hours"}`,
			rows: []transform.Row{
				{Key: "a", Values: map[string]float64{"dollars": 100}},
			},
			wantRow: transform.Row{
				Key:    defaultKey,
				Values: map[string]float64{hoursField: 0},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			prim, err := New([]byte(tc.params))
			require.NoError(t, err)

			s := &transform.State{Rows: append([]transform.Row{}, tc.rows...)}

			err = prim.Apply(context.Background(), s)
			require.NoError(t, err)
			require.Len(t, s.Rows, len(tc.rows)+1)
			assert.Equal(t, tc.wantRow, s.Rows[len(s.Rows)-1])
		})
	}
}
