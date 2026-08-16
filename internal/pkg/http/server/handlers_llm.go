package server

import (
	"context"

	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
)

// ListLLMModels returns the provider's available models with their capability
// flags so the UI can gate the reasoning-effort and temperature controls.
func (s *Server) ListLLMModels(
	ctx context.Context,
	_ api.ListLLMModelsRequestObject,
) (api.ListLLMModelsResponseObject, error) {
	models, err := s.deps.LLMModelLister.ListModels(ctx)
	if err != nil {
		status, body := respondError(ctx, "list llm models", err)

		return api.ListLLMModelsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.ListLLMModels200JSONResponse(models), nil
}

// GetLLMSettings returns the stored LLM settings.
func (s *Server) GetLLMSettings(
	ctx context.Context,
	_ api.GetLLMSettingsRequestObject,
) (api.GetLLMSettingsResponseObject, error) {
	settings, err := s.deps.Store.GetLLMSettings(ctx)
	if err != nil {
		status, body := respondError(ctx, "get llm settings", err)

		return api.GetLLMSettingsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.GetLLMSettings200JSONResponse(llmSettingsToAPI(settings)), nil
}

// UpdateLLMSettings persists the submitted LLM settings.
func (s *Server) UpdateLLMSettings(
	ctx context.Context,
	request api.UpdateLLMSettingsRequestObject,
) (api.UpdateLLMSettingsResponseObject, error) {
	if request.Body == nil || request.Body.Model == "" {
		status, body := respondError(
			ctx, "update llm settings", validationError("model is required"),
		)

		return api.UpdateLLMSettingsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	settings := llmSettingsFromAPIInput(*request.Body)

	if err := s.deps.Store.SaveLLMSettings(ctx, settings); err != nil {
		status, body := respondError(ctx, "update llm settings", err)

		return api.UpdateLLMSettingsdefaultJSONResponse{
			Body:       body,
			StatusCode: status,
		}, nil
	}

	return api.UpdateLLMSettings200JSONResponse(llmSettingsToAPI(settings)), nil
}

func llmSettingsToAPI(s db.LLMSettings) api.LLMSettings {
	out := api.LLMSettings{Model: s.Model}

	if s.ReasoningEffort != "" {
		effort := s.ReasoningEffort
		out.ReasoningEffort = &effort
	}

	temperature := float32(s.Temperature)
	out.Temperature = &temperature

	return out
}

func llmSettingsFromAPIInput(in api.LLMSettingsInput) db.LLMSettings {
	out := db.LLMSettings{Model: in.Model}

	if in.ReasoningEffort != nil {
		out.ReasoningEffort = *in.ReasoningEffort
	}

	if in.Temperature != nil {
		out.Temperature = float64(*in.Temperature)
	}

	return out
}
