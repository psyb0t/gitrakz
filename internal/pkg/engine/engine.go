// Package engine runs a template's transform pipeline over a selected
// timeline and renders the result into a display Document via the
// template's layout — the glue between the transform and template
// packages.
package engine

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
)

// Engine builds and runs a template's transform pipeline, then renders its
// layout into a display Document. It looks up pipeline steps in registry.
type Engine struct {
	registry *transform.Registry
}

// NewEngine builds an Engine backed by registry for resolving a template's
// transform steps by name.
func NewEngine(registry *transform.Registry) *Engine {
	return &Engine{registry: registry}
}

// Run builds tmpl's transform pipeline, runs it over timeline and form, and
// renders tmpl's layout into a display Document. When tmpl.Layout is empty
// and the pipeline already appended blocks (e.g. via passthrough), those
// blocks are returned directly instead of an empty layout.
func (e *Engine) Run(
	ctx context.Context,
	tmpl template.Template,
	timeline types.Timeline,
	form types.FormValues,
) (blocks.Document, error) {
	pipeline, err := e.buildPipeline(tmpl.Transform)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "build transform pipeline")
	}

	state := &transform.State{
		Timeline: timeline,
		Form:     form,
	}

	if err := pipeline.Run(ctx, state); err != nil {
		return nil, ctxerrors.Wrap(err, "run transform pipeline")
	}

	if len(tmpl.Layout) == 0 && len(state.Blocks) > 0 {
		return state.Blocks, nil
	}

	doc, err := applyLayout(tmpl.Layout, state)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "apply layout")
	}

	return doc, nil
}

// buildPipeline resolves every step in steps against e.registry, wrapping
// the offending step's name into the first lookup failure.
func (e *Engine) buildPipeline(
	steps []template.Step,
) (transform.Pipeline, error) {
	primitives := make([]transform.Primitive, 0, len(steps))

	for _, step := range steps {
		primitive, err := e.registry.Build(step.Name, step.Params)
		if err != nil {
			return transform.Pipeline{}, ctxerrors.Wrapf(
				err, "step %q", step.Name,
			)
		}

		primitives = append(primitives, primitive)
	}

	return transform.Pipeline{Steps: primitives}, nil
}
