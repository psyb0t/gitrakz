package mcp

import (
	"context"
	"testing"

	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolset_RunTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var gotFilter db.TimelineFilter

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					_ context.Context, id string,
				) (template.Template, error) {
					assert.Equal(t, "t1", id)

					return template.Template{ID: "t1"}, nil
				},
				queryTimelineFn: func(
					_ context.Context,
					filter db.TimelineFilter,
				) ([]types.Event, bool, error) {
					gotFilter = filter

					return []types.Event{{ID: "e1"}}, false, nil
				},
			},
			Engine: &fakeEngine{
				runFn: func(
					_ context.Context,
					tmpl template.Template,
					timeline types.Timeline,
					form types.FormValues,
				) (blocks.Document, error) {
					assert.Equal(t, "t1", tmpl.ID)
					assert.Equal(t, types.Timeline{{ID: "e1"}}, timeline)
					assert.Equal(t, types.FormValues{"rate": float64(50)}, form)

					return blocks.Document{blocks.NewHeading(1, "Report")}, nil
				},
			},
		}}

		_, out, err := ts.runTemplate(t.Context(), nil, runTemplateInput{
			TemplateID: "t1",
			Filter:     &timelineFilterInput{Owner: "octocat"},
			FormValues: map[string]any{"rate": float64(50)},
		})
		require.NoError(t, err)
		require.Len(t, out.Document, 1)
		assert.Equal(t, "heading", out.Document[0].Type)
		assert.Equal(t, "octocat", gotFilter.Owner)
	})

	t.Run("missing templateId rejected before the store", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{}}

		_, _, err := ts.runTemplate(t.Context(), nil, runTemplateInput{})
		require.ErrorIs(t, err, commerr.ErrValidationFailed)
	})

	t.Run("get template error", func(t *testing.T) {
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

		_, _, err := ts.runTemplate(
			t.Context(), nil, runTemplateInput{TemplateID: "t1"},
		)
		require.ErrorIs(t, err, assert.AnError)
	})

	t.Run("engine error", func(t *testing.T) {
		t.Parallel()

		ts := &toolset{deps: Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					context.Context, string,
				) (template.Template, error) {
					return template.Template{ID: "t1"}, nil
				},
				queryTimelineFn: func(
					context.Context,
					db.TimelineFilter,
				) ([]types.Event, bool, error) {
					return nil, false, nil
				},
			},
			Engine: &fakeEngine{
				runFn: func(
					context.Context,
					template.Template,
					types.Timeline,
					types.FormValues,
				) (blocks.Document, error) {
					return nil, assert.AnError
				},
			},
		}}

		_, _, err := ts.runTemplate(
			t.Context(), nil, runTemplateInput{TemplateID: "t1"},
		)
		require.ErrorIs(t, err, assert.AnError)
	})
}
