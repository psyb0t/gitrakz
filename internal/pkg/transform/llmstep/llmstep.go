// Package llmstep implements the "llm" transform primitive: a
// user-authored LLM step that runs a caller-supplied instruction over the
// pipeline's data and writes the model's response into State.Rows. Unlike
// describe-work (which groups commits and always returns text), llm takes
// its input as-is — State.Rows when an earlier primitive already produced
// one, otherwise the raw timeline — and its instruction and (optional)
// output JSON schema come entirely from the step's params.
//
// LLMClient (interfaces.go) is a small in-package interface the engine
// wires to elelem; llm never talks to a model provider directly.
package llmstep

import (
	"context"
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
)

// Name is the primitive's registry key.
const Name = "llm"

// defaultOutputKey is the Row.Key used when the step params omit "name".
const defaultOutputKey = "llm"

// labelKeyOutput is the Row.Labels key the model's response is stored under.
const labelKeyOutput = "output"

// stepParams is the JSON shape an "llm" pipeline step configures itself
// from. Instruction is required; Schema and Name are optional.
type stepParams struct {
	Instruction string          `json:"instruction"`
	Schema      json.RawMessage `json:"schema"`
	Name        string          `json:"name"`
}

// primitive reads State.Rows (or, when empty, State.Timeline) and writes a
// single Row to State.Rows carrying the LLM's response.
type primitive struct {
	llm LLMClient

	instruction string
	schema      json.RawMessage
	outputKey   string
}

// New builds an llm primitive from llm — the engine's wired LLMClient — and
// its JSON params, shaped {"instruction": string, "schema": object,
// "name": string}. instruction is required; schema defaults to nil (plain
// text output); name defaults to "llm". Returns ErrMissingDependency when
// llm is nil, or ErrMissingInstruction when instruction is empty.
func New(llm LLMClient, rawParams []byte) (transform.Primitive, error) {
	if llm == nil {
		return nil, ctxerrors.Wrap(ErrMissingDependency, Name)
	}

	sp := stepParams{Name: defaultOutputKey}

	if len(rawParams) > 0 {
		if err := json.Unmarshal(rawParams, &sp); err != nil {
			return nil, ctxerrors.Wrap(err, "unmarshal llm params")
		}
	}

	if sp.Instruction == "" {
		return nil, ErrMissingInstruction
	}

	if sp.Name == "" {
		sp.Name = defaultOutputKey
	}

	return primitive{
		llm: llm,

		instruction: sp.Instruction,
		schema:      sp.Schema,
		outputKey:   sp.Name,
	}, nil
}

// Name returns the primitive's registry key.
func (p primitive) Name() string {
	return Name
}

// Apply serializes s.Rows (or s.Timeline when s.Rows is empty) to compact
// JSON, runs p.llm.Complete with p.instruction and p.schema over it, and
// appends one Row carrying the response to s.Rows.
func (p primitive) Apply(ctx context.Context, s *transform.State) error {
	data, err := inputJSON(s)
	if err != nil {
		return ctxerrors.Wrap(err, "marshal llm step input")
	}

	output, err := p.llm.Complete(ctx, p.instruction, data, p.schema)
	if err != nil {
		return ctxerrors.Wrap(err, "llm complete")
	}

	s.Rows = append(s.Rows, transform.Row{
		Key: p.outputKey,
		Labels: map[string]string{
			labelKeyOutput: output,
		},
	})

	return nil
}

// inputJSON marshals s.Rows when non-empty (an earlier primitive already
// shaped the data), or s.Timeline otherwise, to a compact JSON string.
func inputJSON(s *transform.State) (string, error) {
	if len(s.Rows) > 0 {
		b, err := json.Marshal(s.Rows)

		return string(b), err
	}

	b, err := json.Marshal(s.Timeline)

	return string(b), err
}
