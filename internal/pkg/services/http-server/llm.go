package httpserver

import (
	"github.com/psyb0t/elelem"
	"github.com/psyb0t/elelem/drivers/anthropic"
	"github.com/psyb0t/elelem/drivers/openai"
	"github.com/psyb0t/gitrakz/internal/pkg/config"
)

// newLLMClient builds the elelem client from the provider connection config
// (type, base URL, API key). The model and per-request parameters come from
// the stored LLM settings via llmRuntime.
func newLLMClient(cfg config.Config) *elelem.Client {
	var driver elelem.Driver

	switch cfg.ElelemType {
	case anthropic.Name:
		driver = anthropic.NewDriver(
			anthropic.WithAPIKey(cfg.ElelemAPIKey),
			anthropic.WithBaseURL(cfg.ElelemBaseURL),
		)
	default:
		driver = openai.NewDriver(
			openai.WithAPIKey(cfg.ElelemAPIKey),
			openai.WithBaseURL(cfg.ElelemBaseURL),
		)
	}

	return elelem.New(driver)
}
