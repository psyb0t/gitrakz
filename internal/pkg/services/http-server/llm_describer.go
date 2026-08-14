package httpserver

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/elelem"
)

// llmDescriber implements describework.LLMClient over an elelem client — a
// single, non-streaming completion per prompt.
type llmDescriber struct {
	client *elelem.Client
	model  elelem.Model
}

func newLLMDescriber(client *elelem.Client, model elelem.Model) *llmDescriber {
	return &llmDescriber{client: client, model: model}
}

// Describe sends prompt as the sole user turn and returns the model's text.
func (d *llmDescriber) Describe(
	ctx context.Context,
	prompt string,
) (string, error) {
	response, err := elelem.NewRequest(d.client).
		WithModel(d.model).
		WithPrompt(elelem.NewPrompt().UserText(prompt)).
		WithStreaming(false).
		Run(ctx)
	if err != nil {
		return "", ctxerrors.Wrap(err, "elelem describe request")
	}

	return response.Text, nil
}
