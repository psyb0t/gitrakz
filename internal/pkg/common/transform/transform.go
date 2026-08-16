// Package transform defines the pipeline contract for gitrakz templates: an
// ordered chain of primitives that reshape a selected timeline into a typed
// display document. Each primitive reads and writes a shared State; the boss
// precomputes which State fields each primitive touches so primitives compose
// without knowing about each other.
package transform

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Row is the generic tabular intermediate primitives produce and consume:
// a grouping key plus named numeric and string values. Example keys in Values:
// "seconds", "hours", "count", "dollars"; in Labels: "description".
type Row struct {
	Key    string             `json:"key"`
	Values map[string]float64 `json:"values"`
	Labels map[string]string  `json:"labels"`
}

// State is the working state threaded through a pipeline. A primitive mutates
// the fields its contract names and leaves the rest untouched.
//   - Timeline: the selected events (pipeline input).
//   - Sessions: work sessions (sessionize writes; downstream reads).
//   - Rows: tabular intermediate (group/aggregate/split write; rate reads).
//   - Blocks: accumulated display output (terminal primitives append).
//   - Form: the run's form values (read-only).
type State struct {
	Timeline types.Timeline
	Sessions []types.Session
	Rows     []Row
	Blocks   blocks.Document
	Form     types.FormValues
}

// A Primitive is one deterministic (or, for LLM-backed steps, cached) step in a
// template's transform pipeline. Apply mutates s in place.
type Primitive interface {
	Name() string
	Apply(ctx context.Context, s *State) error
}

// Pipeline is an ordered list of primitives run left to right over one State.
type Pipeline struct {
	Steps []Primitive
}

// Run applies every step in order, wrapping the first error with the failing
// primitive's name so a broken pipeline is diagnosable.
func (p Pipeline) Run(ctx context.Context, s *State) error {
	for _, step := range p.Steps {
		if err := step.Apply(ctx, s); err != nil {
			return ctxerrors.Wrapf(err, "transform primitive %q", step.Name())
		}
	}

	return nil
}

// Factory builds a primitive from its JSON parameters (a step entry in a
// template's transform definition). Params is the raw step config.
type Factory func(params []byte) (Primitive, error)

// Registry maps a primitive name to its factory. The template engine looks each
// pipeline step up here, so an unknown step fails loudly instead of silently
// doing nothing.
type Registry struct {
	factories map[string]Factory
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{factories: map[string]Factory{}}
}

// Register adds a factory under name, overwriting any previous entry.
func (r *Registry) Register(name string, f Factory) {
	r.factories[name] = f
}

// Build constructs the named primitive, or returns ErrUnknownPrimitive wrapped
// with the offending name.
func (r *Registry) Build(name string, params []byte) (Primitive, error) {
	f, ok := r.factories[name]
	if !ok {
		return nil, ctxerrors.Wrapf(ErrUnknownPrimitive, "name %q", name)
	}

	return f(params)
}
