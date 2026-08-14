package rate

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("defaults when params are empty", func(t *testing.T) {
		t.Parallel()

		p, err := New(nil)
		require.NoError(t, err)
		assert.Equal(t, Name, p.Name())

		pr, ok := p.(primitive)
		require.True(t, ok)
		assert.Equal(t, defaultHoursField, pr.hoursField)
		assert.Equal(t, defaultDollarsField, pr.dollarsField)
		assert.Equal(t, defaultFormKey, pr.formKey)
		assert.InDelta(t, 0, pr.fallbackRate, 0)
	})

	t.Run("overrides from params JSON", func(t *testing.T) {
		t.Parallel()

		raw := []byte(`{
			"hoursField": "hrs",
			"dollarsField": "usd",
			"formKey": "hourlyRate",
			"rate": 50
		}`)

		p, err := New(raw)
		require.NoError(t, err)

		pr, ok := p.(primitive)
		require.True(t, ok)
		assert.Equal(t, "hrs", pr.hoursField)
		assert.Equal(t, "usd", pr.dollarsField)
		assert.Equal(t, "hourlyRate", pr.formKey)
		assert.InDelta(t, 50, pr.fallbackRate, 0)
	})

	t.Run("invalid params JSON errors", func(t *testing.T) {
		t.Parallel()

		_, err := New([]byte(`{`))
		require.Error(t, err)
	})
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	t.Run("computes dollars from the form rate", func(t *testing.T) {
		t.Parallel()

		p, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{
					Key: "a",
					Values: map[string]float64{
						defaultHoursField: 2,
					},
				},
				{
					Key: "b",
					Values: map[string]float64{
						defaultHoursField: 3,
					},
				},
			},
			Form: map[string]any{defaultFormKey: 100.0},
		}

		require.NoError(t, p.Apply(context.Background(), s))

		assert.InDelta(
			t, 200, s.Rows[0].Values[defaultDollarsField], 0,
		)
		assert.InDelta(
			t, 300, s.Rows[1].Values[defaultDollarsField], 0,
		)
	})

	t.Run("coerces an int form rate", func(t *testing.T) {
		t.Parallel()

		p, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{
					Key: "a",
					Values: map[string]float64{
						defaultHoursField: 2,
					},
				},
			},
			Form: map[string]any{defaultFormKey: 100},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		assert.InDelta(
			t, 200, s.Rows[0].Values[defaultDollarsField], 0,
		)
	})

	t.Run("coerces a json.Number form rate", func(t *testing.T) {
		t.Parallel()

		p, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{
					Key: "a",
					Values: map[string]float64{
						defaultHoursField: 2,
					},
				},
			},
			Form: map[string]any{
				defaultFormKey: json.Number("100"),
			},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		assert.InDelta(
			t, 200, s.Rows[0].Values[defaultDollarsField], 0,
		)
	})

	t.Run("missing hours field leaves the row untouched", func(t *testing.T) {
		t.Parallel()

		p, err := New(nil)
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{Key: "a", Values: map[string]float64{"count": 5}},
			},
			Form: map[string]any{defaultFormKey: 100.0},
		}

		require.NoError(t, p.Apply(context.Background(), s))

		_, ok := s.Rows[0].Values[defaultDollarsField]
		assert.False(t, ok)
	})

	t.Run("falls back to params rate, no form key", func(t *testing.T) {
		t.Parallel()

		p, err := New([]byte(`{"rate": 25}`))
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{
					Key: "a",
					Values: map[string]float64{
						defaultHoursField: 4,
					},
				},
			},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		assert.InDelta(
			t, 100, s.Rows[0].Values[defaultDollarsField], 0,
		)
	})

	t.Run("falls back to params rate, non-numeric form", func(t *testing.T) {
		t.Parallel()

		p, err := New([]byte(`{"rate": 25}`))
		require.NoError(t, err)

		s := &transform.State{
			Rows: []transform.Row{
				{
					Key: "a",
					Values: map[string]float64{
						defaultHoursField: 4,
					},
				},
			},
			Form: map[string]any{defaultFormKey: "not-a-number"},
		}

		require.NoError(t, p.Apply(context.Background(), s))
		assert.InDelta(
			t, 100, s.Rows[0].Values[defaultDollarsField], 0,
		)
	})
}
