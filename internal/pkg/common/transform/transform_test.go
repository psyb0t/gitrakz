package transform

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errBoom = ctxerrors.New("boom")

// recPrimitive records its name into a shared slice when applied, and tags the
// State with a Row so the pipeline's mutations are observable.
type recPrimitive struct {
	name   string
	record *[]string
}

func (r recPrimitive) Name() string {
	return r.name
}

func (r recPrimitive) Apply(_ context.Context, s *State) error {
	*r.record = append(*r.record, r.name)
	s.Rows = append(s.Rows, Row{Key: r.name})

	return nil
}

// errPrimitive always fails with errBoom.
type errPrimitive struct {
	name string
}

func (e errPrimitive) Name() string {
	return e.name
}

func (e errPrimitive) Apply(_ context.Context, _ *State) error {
	return errBoom
}

func TestPipeline_Run(t *testing.T) {
	t.Run("applies steps in order", func(t *testing.T) {
		var order []string

		p := Pipeline{Steps: []Primitive{
			recPrimitive{name: "a", record: &order},
			recPrimitive{name: "b", record: &order},
		}}

		s := &State{}
		require.NoError(t, p.Run(context.Background(), s))
		assert.Equal(t, []string{"a", "b"}, order)
		assert.Len(t, s.Rows, 2)
	})

	t.Run("wraps the failing primitive name", func(t *testing.T) {
		p := Pipeline{Steps: []Primitive{errPrimitive{name: "broken"}}}

		err := p.Run(context.Background(), &State{})
		require.Error(t, err)
		assert.ErrorIs(t, err, errBoom)
		assert.Contains(t, err.Error(), "broken")
	})
}

func TestRegistry_Build(t *testing.T) {
	t.Run("builds a registered primitive", func(t *testing.T) {
		var order []string

		r := NewRegistry()
		r.Register("fake", func(_ []byte) (Primitive, error) {
			return recPrimitive{name: "fake", record: &order}, nil
		})

		p, err := r.Build("fake", nil)
		require.NoError(t, err)
		assert.Equal(t, "fake", p.Name())
	})

	t.Run("unknown name errors", func(t *testing.T) {
		r := NewRegistry()

		_, err := r.Build("nope", nil)
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrUnknownPrimitive)
		assert.Contains(t, err.Error(), "nope")
	})
}
