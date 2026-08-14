package httpserver

import (
	"github.com/psyb0t/elelem"
	"github.com/psyb0t/elelem/drivers/anthropic"
	"github.com/psyb0t/elelem/drivers/openai"
	"github.com/psyb0t/gitrakz/internal/pkg/config"
)

const defaultLLMContextSize = 128_000

func newLLMClient(cfg config.Config) (*elelem.Client, elelem.Model) {
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

	model := elelem.Model{
		ID:          cfg.ElelemModel,
		ContextSize: defaultLLMContextSize,
	}

	client := elelem.New(driver, elelem.WithDefaultModel(model))

	return client, model
}
