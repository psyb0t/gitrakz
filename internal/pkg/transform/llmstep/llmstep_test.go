package llmstep

import (
	"context"
	"errors"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// errMockFailure is a shared sentinel the mockLLM returns to verify Apply
// wraps and propagates its error.
var errMockFailure = errors.New("mock failure")

// mockLLM is a stub LLMClient that always returns output (or err), recording
// the arguments of its last call.
type mockLLM struct {
	calls int
	err   error

	output string

	lastInstruction string
	lastData        string
	lastSchema      []byte
}

func (m *mockLLM) Complete(
	_ context.Context, instruction, data string, schema []byte,
) (string, error) {
	m.calls++
	m.lastInstruction = instruction
	m.lastData = data
	m.lastSchema = schema

	if m.err != nil {
		return "", m.err
	}

	return m.output, nil
}

func TestNew(t *testing.T) {
	t.Parallel()

	t.Run("nil llm errors", func(t *testing.T) {
		t.Parallel()

		_, err := New(nil, []byte(`{"instruction":"summarize this"}`))
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrMissingDependency)
	})

	t.Run("missing instruction errors", func(t *testing.T) {
		t.Parallel()

		testCases := []struct {
			name   string
			params []byte
		}{
			{"nil params", nil},
			{"empty params", []byte(`{}`)},
			{"blank instruction", []byte(`{"instruction":""}`)},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				_, err := New(&mockLLM{}, tc.params)
				require.Error(t, err)
				assert.ErrorIs(t, err, ErrMissingInstruction)
			})
		}
	})

	t.Run("malformed params errors", func(t *testing.T) {
		t.Parallel()

		_, err := New(&mockLLM{}, []byte(`{`))
		require.Error(t, err)
	})

	t.Run("default output key", func(t *testing.T) {
		t.Parallel()

		prim, err := New(&mockLLM{}, []byte(`{"instruction":"summarize"}`))
		require.NoError(t, err)
		assert.Equal(t, Name, prim.Name())

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, defaultOutputKey, p.outputKey)
		assert.Equal(t, "summarize", p.instruction)
		assert.Nil(t, p.schema)
	})

	t.Run("custom output key and schema", func(t *testing.T) {
		t.Parallel()

		rawParams := []byte(
			`{"instruction":"classify this","name":"classification",` +
				`"schema":{"type":"object"}}`,
		)

		prim, err := New(&mockLLM{}, rawParams)
		require.NoError(t, err)

		p, ok := prim.(primitive)
		require.True(t, ok)
		assert.Equal(t, "classification", p.outputKey)
		assert.Equal(t, "classify this", p.instruction)
		assert.JSONEq(t, `{"type":"object"}`, string(p.schema))
	})
}

func TestPrimitive_Name(t *testing.T) {
	t.Parallel()

	prim, err := New(&mockLLM{}, []byte(`{"instruction":"summarize"}`))
	require.NoError(t, err)
	assert.Equal(t, "llm", prim.Name())
}

func TestPrimitive_Apply(t *testing.T) {
	t.Parallel()

	timeline := types.Timeline{
		{
			Type:  types.EventTypeCommit,
			Owner: "acme",
			Repo:  "widgets",
			SHA:   "sha1",
			Title: "add pagination to the timeline endpoint",
			TS:    1700000000,
		},
	}

	t.Run("text mode passes nil schema and writes output row", func(
		t *testing.T,
	) {
		t.Parallel()

		llm := &mockLLM{output: "a one-line summary"}

		prim, err := New(llm, []byte(`{"instruction":"summarize this"}`))
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, 1, llm.calls)
		assert.Equal(t, "summarize this", llm.lastInstruction)
		assert.Nil(t, llm.lastSchema)
		assert.Contains(t, llm.lastData, "widgets")

		require.Len(t, s.Rows, 1)
		assert.Equal(t, defaultOutputKey, s.Rows[0].Key)
		assert.Equal(
			t, "a one-line summary", s.Rows[0].Labels[labelKeyOutput],
		)
	})

	t.Run("schema mode passes the schema through unchanged", func(
		t *testing.T,
	) {
		t.Parallel()

		llm := &mockLLM{output: `{"result":"ok"}`}

		rawParams := []byte(
			`{"instruction":"classify this",` +
				`"schema":{"type":"object",` +
				`"properties":{"result":{"type":"string"}}}}`,
		)

		prim, err := New(llm, rawParams)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.NotNil(t, llm.lastSchema)
		assert.JSONEq(
			t,
			`{"type":"object","properties":{"result":{"type":"string"}}}`,
			string(llm.lastSchema),
		)
		require.Len(t, s.Rows, 1)
		assert.Equal(
			t, `{"result":"ok"}`, s.Rows[0].Labels[labelKeyOutput],
		)
	})

	t.Run("custom output key names the row", func(t *testing.T) {
		t.Parallel()

		llm := &mockLLM{output: "classified"}

		rawParams := []byte(
			`{"instruction":"classify this","name":"classification"}`,
		)

		prim, err := New(llm, rawParams)
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		require.NoError(t, prim.Apply(context.Background(), s))

		require.Len(t, s.Rows, 1)
		assert.Equal(t, "classification", s.Rows[0].Key)
	})

	t.Run("existing rows are used as input over the raw timeline", func(
		t *testing.T,
	) {
		t.Parallel()

		llm := &mockLLM{output: "summary of rows"}

		prim, err := New(llm, []byte(`{"instruction":"summarize this"}`))
		require.NoError(t, err)

		s := &transform.State{
			Timeline: timeline,
			Rows: []transform.Row{
				{Key: "alpha", Values: map[string]float64{"count": 3}},
			},
		}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Contains(t, llm.lastData, "alpha")
		assert.NotContains(t, llm.lastData, "widgets")
	})

	t.Run("llm error is wrapped and returned", func(t *testing.T) {
		t.Parallel()

		llm := &mockLLM{err: errMockFailure}

		prim, err := New(llm, []byte(`{"instruction":"summarize this"}`))
		require.NoError(t, err)

		s := &transform.State{Timeline: timeline}
		err = prim.Apply(context.Background(), s)
		require.Error(t, err)
		assert.ErrorIs(t, err, errMockFailure)
	})

	t.Run("empty state produces one row from an empty timeline", func(
		t *testing.T,
	) {
		t.Parallel()

		llm := &mockLLM{output: "nothing to see"}

		prim, err := New(llm, []byte(`{"instruction":"summarize this"}`))
		require.NoError(t, err)

		s := &transform.State{}
		require.NoError(t, prim.Apply(context.Background(), s))

		assert.Equal(t, "null", llm.lastData)
		require.Len(t, s.Rows, 1)
	})
}
