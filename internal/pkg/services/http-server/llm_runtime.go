package httpserver

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/elelem"
	"github.com/psyb0t/elelem/drivers/anthropic"
	"github.com/psyb0t/elelem/drivers/openai"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

const defaultLLMContextSize = 128_000

// llmSettingsReader reads the persisted LLM settings.
type llmSettingsReader interface {
	GetLLMSettings(ctx context.Context) (db.LLMSettings, error)
}

// llmRuntime resolves the runtime model and parameters from stored settings and
// lists the provider's available models. It backs both the summarization
// describer and the template composer, and the /v1/llm/models endpoint.
type llmRuntime struct {
	client   *elelem.Client
	kind     string
	settings llmSettingsReader
}

func newLLMRuntime(
	client *elelem.Client, kind string, settings llmSettingsReader,
) *llmRuntime {
	return &llmRuntime{client: client, kind: kind, settings: settings}
}

// configure applies the stored model, reasoning effort, and temperature to req,
// gating reasoning and temperature on what the selected model supports.
func (r *llmRuntime) configure(
	ctx context.Context, req *elelem.Request,
) (*elelem.Request, error) {
	settings, err := r.settings.GetLLMSettings(ctx)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "read llm settings")
	}

	model := r.lookupModel(settings.Model)
	req = req.WithModel(model)

	caps := r.client.Capabilities(model)
	if settings.ReasoningEffort != "" && caps.SupportsReasoningEffort {
		req = req.WithReasoningEffort(settings.ReasoningEffort)
	}

	if caps.SupportsSamplingParams {
		req = req.WithTemperature(settings.Temperature)
	}

	return req, nil
}

// ListModels returns the provider's available models with capability flags.
func (r *llmRuntime) ListModels(ctx context.Context) ([]api.LLMModel, error) {
	ids, err := r.client.Driver().ListModels(ctx)
	if err != nil {
		return nil, ctxerrors.Wrap(err, "list provider models")
	}

	out := make([]api.LLMModel, 0, len(ids))

	for _, id := range ids {
		model := r.lookupModel(id)
		caps := r.client.Capabilities(model)
		out = append(out, api.LLMModel{
			Id:                      id,
			ContextSize:             model.ContextSize,
			SupportsReasoningEffort: caps.SupportsReasoningEffort,
			MaxReasoningEffort:      caps.MaxReasoningEffort,
			SupportsSamplingParams:  caps.SupportsSamplingParams,
		})
	}

	return out, nil
}

// lookupModel resolves a bare model ID into a full model (context size,
// reasoning levels) from the active driver's model table.
func (r *llmRuntime) lookupModel(id string) elelem.Model {
	var model elelem.Model
	if r.kind == anthropic.Name {
		model = anthropic.LookupModel(id)
	} else {
		model = openai.LookupModel(id)
	}

	if model.ContextSize == 0 {
		model.ContextSize = defaultLLMContextSize
	}

	return model
}
