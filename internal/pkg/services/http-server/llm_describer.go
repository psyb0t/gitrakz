package httpserver

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/elelem"
)

// llmDescriber implements describework.LLMClient over an elelem client — a
// single, non-streaming completion per prompt, using the stored LLM settings.
type llmDescriber struct {
	runtime *llmRuntime
}

func newLLMDescriber(runtime *llmRuntime) *llmDescriber {
	return &llmDescriber{runtime: runtime}
}

// Describe sends prompt as the sole user turn and returns the model's text.
func (d *llmDescriber) Describe(
	ctx context.Context,
	prompt string,
) (string, error) {
	req, err := d.runtime.configure(ctx, elelem.NewRequest(d.runtime.client))
	if err != nil {
		return "", err
	}

	response, err := req.
		WithPrompt(elelem.NewPrompt().UserText(prompt)).
		WithStreaming(false).
		Run(ctx)
	if err != nil {
		return "", ctxerrors.Wrap(err, "elelem describe request")
	}

	return response.Text, nil
}
