package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolset_ListTemplates(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listTemplatesFn: func(
					context.Context,
				) ([]template.Template, error) {
					return []template.Template{
						{ID: "t1", Name: "Weekly Report", Builtin: true},
					}, nil
				},
			},
		}}

		_, out, err := ts.listTemplates(t.Context(), nil, listTemplatesInput{})
		require.NoError(t, err)
		require.Len(t, out.Templates, 1)
		assert.Equal(t, "t1", out.Templates[0].ID)
		assert.True(t, out.Templates[0].Builtin)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listTemplatesFn: func(
					context.Context,
				) ([]template.Template, error) {
					return nil, assert.AnError
				},
			},
		}}

		_, _, err := ts.listTemplates(t.Context(), nil, listTemplatesInput{})
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("undecodable step params surface as an error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				listTemplatesFn: func(
					context.Context,
				) ([]template.Template, error) {
					return []template.Template{{
						ID: "t1",
						Transform: []template.Step{
							{
								Name:   "sessionize",
								Params: json.RawMessage(`not-json`),
							},
						},
					}}, nil
				},
			},
		}}

		_, _, err := ts.listTemplates(t.Context(), nil, listTemplatesInput{})
		require.Error(t, err)
	})
}

func TestToolset_GetTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					_ context.Context, id string,
				) (template.Template, error) {
					assert.Equal(t, "t1", id)

					return template.Template{
						ID: "t1", Name: "Weekly Report",
					}, nil
				},
			},
		}}

		_, out, err := ts.getTemplate(
			t.Context(), nil, getTemplateInput{ID: "t1"},
		)
		require.NoError(t, err)
		assert.Equal(t, "t1", out.Template.ID)
		assert.Equal(t, "Weekly Report", out.Template.Name)
	})

	t.Run("missing id is rejected before the store", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{}}

		_, _, err := ts.getTemplate(t.Context(), nil, getTemplateInput{})
		require.ErrorIs(t, err, commerr.ErrValidationFailed)
	})

	t.Run("store error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					context.Context, string,
				) (template.Template, error) {
					return template.Template{}, assert.AnError
				},
			},
		}}

		_, _, err := ts.getTemplate(
			t.Context(), nil, getTemplateInput{ID: "t1"},
		)
		require.ErrorIs(t, err, assert.AnError)
	})
}
