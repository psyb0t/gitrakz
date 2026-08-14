package httpserver

import (
	"context"

	"github.com/google/uuid"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/elelem"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
)

// generateTemplateSystemPrompt describes the SHAPE of a gitrakz template —
// the building blocks the model may compose from — never a worked example
// with real data (per the "never bake real data into LLM prompts" rule).
const generateTemplateSystemPrompt = `You compose a gitrakz template: a JSON
object describing "name", "description", "form", "transform", "layout",
"exports", and optionally "model". Never author raw HTML or free-form code —
every field is one of the building blocks below.

"form" is a list of input fields collected before a run. Each entry has
"key", "label", "type" (one of "string", "number", "bool", "offHours"),
"required", and an optional "default".

"transform" is an ordered pipeline of steps. Each entry has "name" (one of
the primitives below) and an optional "params" object matching that
primitive's own configuration shape:
  sessionize, exclude-off-time, split-by-active-days, group-by, aggregate,
  rate, passthrough, describe-work

"layout" is an ordered list of display blocks rendering the transform's
output. Each entry has "type" (one of the block types below), an optional
"source" naming which part of the transform output feeds it, and an
optional "params" object:
  heading, text, list, table, keyvalue, metric, code, chart

"exports" is a subset of: csv, pdf, json.

Compose a template matching the user's description using only these
building blocks. Respond with only the JSON object — no prose, no markdown
fences.`

// llmComposer implements server.LLMComposer over an elelem client:
// structured-decode the model's reply directly into template.Template, per
// elelem's RunInto (schema derived from the struct's own json tags).
type llmComposer struct {
	client *elelem.Client
	model  elelem.Model
}

func newLLMComposer(client *elelem.Client, model elelem.Model) *llmComposer {
	return &llmComposer{client: client, model: model}
}

// GenerateTemplate drafts a template.Template from description. Nothing is
// persisted here — the caller reviews/edits/saves separately. id/Builtin
// are always overwritten after decode: a draft is never a saved builtin,
// and any id the model invented is discarded in favor of a fresh one, same
// as the CreateTemplate handler assigns for a client-submitted draft.
func (c *llmComposer) GenerateTemplate(
	ctx context.Context,
	description string,
) (template.Template, error) {
	var tmpl template.Template

	_, err := elelem.NewRequest(c.client).
		WithModel(c.model).
		WithPrompt(elelem.NewPrompt().
			WithSystem(generateTemplateSystemPrompt).
			UserText(description)).
		WithStreaming(false).
		RunInto(ctx, &tmpl)
	if err != nil {
		return template.Template{}, ctxerrors.Wrap(err, "generate template")
	}

	tmpl.ID = uuid.NewString()
	tmpl.Builtin = false

	return tmpl, nil
}
