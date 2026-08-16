package server

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ListLLMModels(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{LLMModelLister: &fakeLLMModelLister{
			listModelsFn: func(context.Context) ([]api.LLMModel, error) {
				return []api.LLMModel{{
					Id:                      "gpt-5",
					ContextSize:             400000,
					SupportsReasoningEffort: true,
					MaxReasoningEffort:      "high",
					SupportsSamplingParams:  true,
				}}, nil
			},
		}})

		resp, err := srv.ListLLMModels(
			context.Background(), api.ListLLMModelsRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListLLMModels200JSONResponse)
		require.True(t, ok)
		require.Len(t, got, 1)
		assert.Equal(t, "gpt-5", got[0].Id)
		assert.True(t, got[0].SupportsReasoningEffort)
	})

	t.Run("lister error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{LLMModelLister: &fakeLLMModelLister{
			listModelsFn: func(context.Context) ([]api.LLMModel, error) {
				return nil, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListLLMModels(
			context.Background(), api.ListLLMModelsRequestObject{},
		)
		require.NoError(t, err)

		_, ok := resp.(api.ListLLMModelsdefaultJSONResponse)
		require.True(t, ok)
	})
}

func TestServer_GetLLMSettings(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getLLMSettingsFn: func(context.Context) (db.LLMSettings, error) {
				return db.LLMSettings{
					Model:           "gpt-5",
					ReasoningEffort: "high",
					Temperature:     0.4,
				}, nil
			},
		}})

		resp, err := srv.GetLLMSettings(
			context.Background(), api.GetLLMSettingsRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GetLLMSettings200JSONResponse)
		require.True(t, ok)
		assert.Equal(t, "gpt-5", got.Model)
		require.NotNil(t, got.ReasoningEffort)
		assert.Equal(t, "high", *got.ReasoningEffort)
		require.NotNil(t, got.Temperature)
		assert.InDelta(t, 0.4, float64(*got.Temperature), 0.001)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getLLMSettingsFn: func(context.Context) (db.LLMSettings, error) {
				return db.LLMSettings{}, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.GetLLMSettings(
			context.Background(), api.GetLLMSettingsRequestObject{},
		)
		require.NoError(t, err)

		_, ok := resp.(api.GetLLMSettingsdefaultJSONResponse)
		require.True(t, ok)
	})
}

func TestServer_UpdateLLMSettings(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var saved db.LLMSettings

		srv := New(Deps{Store: &fakeStore{
			saveLLMSettingsFn: func(_ context.Context, s db.LLMSettings) error {
				saved = s

				return nil
			},
		}})

		effort := "medium"
		temp := float32(0.7)

		resp, err := srv.UpdateLLMSettings(
			context.Background(),
			api.UpdateLLMSettingsRequestObject{Body: &api.LLMSettingsInput{
				Model:           "claude-x",
				ReasoningEffort: &effort,
				Temperature:     &temp,
			}},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateLLMSettings200JSONResponse)
		require.True(t, ok)
		assert.Equal(t, "claude-x", got.Model)
		assert.Equal(t, "claude-x", saved.Model)
		assert.Equal(t, "medium", saved.ReasoningEffort)
		assert.InDelta(t, 0.7, saved.Temperature, 0.001)
	})

	t.Run("missing model is rejected", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{}})

		resp, err := srv.UpdateLLMSettings(
			context.Background(),
			api.UpdateLLMSettingsRequestObject{
				Body: &api.LLMSettingsInput{Model: ""},
			},
		)
		require.NoError(t, err)

		_, ok := resp.(api.UpdateLLMSettingsdefaultJSONResponse)
		require.True(t, ok)
	})

	t.Run("save error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			saveLLMSettingsFn: func(context.Context, db.LLMSettings) error {
				return ctxerrors.New("boom")
			},
		}})

		resp, err := srv.UpdateLLMSettings(
			context.Background(),
			api.UpdateLLMSettingsRequestObject{
				Body: &api.LLMSettingsInput{Model: "gpt-5"},
			},
		)
		require.NoError(t, err)

		_, ok := resp.(api.UpdateLLMSettingsdefaultJSONResponse)
		require.True(t, ok)
	})
}
