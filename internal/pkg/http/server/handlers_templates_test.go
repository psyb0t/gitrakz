package server

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServer_ListTemplates(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listTemplatesFn: func(
				context.Context,
			) ([]template.Template, error) {
				return []template.Template{
					{ID: "t1", Name: "Timesheet", Builtin: true},
				}, nil
			},
		}})

		resp, err := srv.ListTemplates(
			context.Background(), api.ListTemplatesRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListTemplates200JSONResponse)
		require.True(t, ok)
		require.Len(t, got, 1)
		assert.Equal(t, "t1", got[0].Id)
		assert.Equal(t, "Timesheet", got[0].Name)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			listTemplatesFn: func(
				context.Context,
			) ([]template.Template, error) {
				return nil, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.ListTemplates(
			context.Background(), api.ListTemplatesRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ListTemplatesdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_CreateTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var saved template.Template

		srv := New(Deps{Store: &fakeStore{
			saveTemplateFn: func(
				_ context.Context,
				tmpl template.Template,
			) error {
				saved = tmpl

				return nil
			},
		}})

		resp, err := srv.CreateTemplate(
			context.Background(),
			api.CreateTemplateRequestObject{
				Body: &api.CreateTemplateJSONRequestBody{
					Name:      "My Report",
					Form:      []api.FormField{},
					Transform: []api.TransformStep{},
					Layout:    []api.Block{},
					Exports:   []api.ExportFormat{},
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.CreateTemplate200JSONResponse)
		require.True(t, ok)
		assert.Equal(t, "My Report", got.Name)
		assert.NotEmpty(t, got.Id)
		assert.False(t, saved.Builtin)
		assert.Equal(t, got.Id, saved.ID)
	})

	t.Run("missing name", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{}})

		resp, err := srv.CreateTemplate(
			context.Background(),
			api.CreateTemplateRequestObject{
				Body: &api.CreateTemplateJSONRequestBody{},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.CreateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("nil body", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{}})

		resp, err := srv.CreateTemplate(
			context.Background(), api.CreateTemplateRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.CreateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("save error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			saveTemplateFn: func(
				context.Context,
				template.Template,
			) error {
				return ctxerrors.New("boom")
			},
		}})

		resp, err := srv.CreateTemplate(
			context.Background(),
			api.CreateTemplateRequestObject{
				Body: &api.CreateTemplateJSONRequestBody{Name: "x"},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.CreateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_GenerateTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{LLMComposer: &fakeLLMComposer{
			generateTemplateFn: func(
				_ context.Context,
				description string,
			) (template.Template, error) {
				assert.Equal(t, "weekly standup", description)

				return template.Template{ID: "gen1", Name: "Standup"}, nil
			},
		}})

		resp, err := srv.GenerateTemplate(
			context.Background(),
			api.GenerateTemplateRequestObject{
				Body: &api.GenerateTemplateJSONRequestBody{
					Prompt: "weekly standup",
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GenerateTemplate200JSONResponse)
		require.True(t, ok)
		assert.Equal(t, "Standup", got.Name)
	})

	t.Run("missing prompt", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{LLMComposer: &fakeLLMComposer{}})

		resp, err := srv.GenerateTemplate(
			context.Background(),
			api.GenerateTemplateRequestObject{
				Body: &api.GenerateTemplateJSONRequestBody{},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GenerateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("composer error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{LLMComposer: &fakeLLMComposer{
			generateTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{}, ctxerrors.New("boom")
			},
		}})

		resp, err := srv.GenerateTemplate(
			context.Background(),
			api.GenerateTemplateRequestObject{
				Body: &api.GenerateTemplateJSONRequestBody{Prompt: "x"},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.GenerateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_DeleteTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		deleted := false

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "t1"}, nil
			},
			deleteTemplateFn: func(_ context.Context, id string) error {
				assert.Equal(t, "t1", id)

				deleted = true

				return nil
			},
		}})

		resp, err := srv.DeleteTemplate(
			context.Background(),
			api.DeleteTemplateRequestObject{Id: "t1"},
		)
		require.NoError(t, err)
		assert.True(t, deleted)
		assert.Equal(t, api.DeleteTemplate204Response{}, resp)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{}, ctxerrors.Wrap(
					commerr.ErrNotFound, "get template",
				)
			},
		}})

		resp, err := srv.DeleteTemplate(
			context.Background(),
			api.DeleteTemplateRequestObject{Id: "missing"},
		)
		require.NoError(t, err)

		got, ok := resp.(api.DeleteTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 404, got.StatusCode)
	})

	t.Run("builtin rejected", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "b1", Builtin: true}, nil
			},
		}})

		resp, err := srv.DeleteTemplate(
			context.Background(),
			api.DeleteTemplateRequestObject{Id: "b1"},
		)
		require.NoError(t, err)

		got, ok := resp.(api.DeleteTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 403, got.StatusCode)
	})

	t.Run("delete error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "t1"}, nil
			},
			deleteTemplateFn: func(context.Context, string) error {
				return ctxerrors.New("boom")
			},
		}})

		resp, err := srv.DeleteTemplate(
			context.Background(),
			api.DeleteTemplateRequestObject{Id: "t1"},
		)
		require.NoError(t, err)

		got, ok := resp.(api.DeleteTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_UpdateTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var saved template.Template

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "t1", Name: "old"}, nil
			},
			saveTemplateFn: func(
				_ context.Context,
				tmpl template.Template,
			) error {
				saved = tmpl

				return nil
			},
		}})

		resp, err := srv.UpdateTemplate(
			context.Background(),
			api.UpdateTemplateRequestObject{
				Id: "t1",
				Body: &api.UpdateTemplateJSONRequestBody{
					Name: "new",
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateTemplate200JSONResponse)
		require.True(t, ok)
		assert.Equal(t, "new", got.Name)
		assert.Equal(t, "t1", got.Id)
		assert.Equal(t, "t1", saved.ID)
		assert.Equal(t, "new", saved.Name)
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{}, ctxerrors.Wrap(
					commerr.ErrNotFound, "get template",
				)
			},
		}})

		resp, err := srv.UpdateTemplate(
			context.Background(),
			api.UpdateTemplateRequestObject{Id: "missing"},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 404, got.StatusCode)
	})

	t.Run("builtin rejected", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "b1", Builtin: true}, nil
			},
		}})

		resp, err := srv.UpdateTemplate(
			context.Background(),
			api.UpdateTemplateRequestObject{
				Id:   "b1",
				Body: &api.UpdateTemplateJSONRequestBody{Name: "new"},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 403, got.StatusCode)
	})

	t.Run("missing name", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "t1"}, nil
			},
		}})

		resp, err := srv.UpdateTemplate(
			context.Background(),
			api.UpdateTemplateRequestObject{
				Id:   "t1",
				Body: &api.UpdateTemplateJSONRequestBody{},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("save error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{
			getTemplateFn: func(
				context.Context,
				string,
			) (template.Template, error) {
				return template.Template{ID: "t1"}, nil
			},
			saveTemplateFn: func(
				context.Context,
				template.Template,
			) error {
				return ctxerrors.New("boom")
			},
		}})

		resp, err := srv.UpdateTemplate(
			context.Background(),
			api.UpdateTemplateRequestObject{
				Id:   "t1",
				Body: &api.UpdateTemplateJSONRequestBody{Name: "new"},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.UpdateTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}
