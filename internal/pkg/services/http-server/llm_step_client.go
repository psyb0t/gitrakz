package httpserver

import (
	"context"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/elelem"
)

// llmStepSchemaName is the JSON-schema response name sent to elelem for a
// schema-mode "llm" step. It has no meaning beyond a stable identifier —
// gitrakz never inspects it.
const llmStepSchemaName = "llm_step_output"

// llmStepStrictSchema requires the model's structured output to match the
// step's schema exactly.
const llmStepStrictSchema = true

// llmStepClient implements llmstep.LLMClient over an elelem client: a
// single, non-streaming completion per instruction, using the stored LLM
// settings. A non-empty schema requests JSON-schema-constrained output when
// the selected model supports it; otherwise the request falls back to plain
// text.
type llmStepClient struct {
	runtime *llmRuntime
}

func newLLMStepClient(runtime *llmRuntime) *llmStepClient {
	return &llmStepClient{runtime: runtime}
}

// Complete sends instruction as the system message and data as the sole
// user turn, and returns the model's text (raw JSON text in schema mode).
func (c *llmStepClient) Complete(
	ctx context.Context,
	instruction, data string,
	schema []byte,
) (string, error) {
	req, err := c.runtime.configure(ctx, elelem.NewRequest(c.runtime.client))
	if err != nil {
		return "", err
	}

	req = req.WithPrompt(elelem.NewPrompt().
		WithSystem(instruction).
		UserText(data))

	if len(schema) > 0 {
		supported, err := c.supportsJSONSchema(ctx)
		if err != nil {
			return "", err
		}

		if supported {
			req = req.WithJSONSchema(
				llmStepSchemaName, schema, llmStepStrictSchema,
			)
		}
	}

	response, err := req.WithStreaming(false).Run(ctx)
	if err != nil {
		return "", ctxerrors.Wrap(err, "elelem llm step request")
	}

	return response.Text, nil
}

// supportsJSONSchema reports whether the currently configured model
// supports JSON-schema-constrained structured output.
func (c *llmStepClient) supportsJSONSchema(ctx context.Context) (bool, error) {
	settings, err := c.runtime.settings.GetLLMSettings(ctx)
	if err != nil {
		return false, ctxerrors.Wrap(err, "read llm settings")
	}

	model := c.runtime.lookupModel(settings.Model)
	caps := c.runtime.client.Capabilities(model)

	return caps.SupportsResponseFormatJSONSchema, nil
}
