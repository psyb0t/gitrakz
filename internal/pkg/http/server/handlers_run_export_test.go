package server

import (
	"context"
	"io"
	"testing"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testDocument includes both a prose block (heading) and a tabular block
// (metric) so it renders non-empty output in every export format —
// export.ToCSV skips prose-only blocks (heading/text/code) entirely.
func testDocument() blocks.Document {
	return blocks.Document{
		blocks.NewHeading(1, "Report"),
		blocks.NewMetric("Hours", "40", "h"),
	}
}

// exportOctetStreamResponse shortens the generated response type name so
// multi-line type assertions in this file's tests stay readable.
//
//nolint:lll // the generated octet-stream response type name is inherently long
type exportOctetStreamResponse = api.ExportDocument200ApplicationoctetStreamResponse

func TestServer_RunTemplate(t *testing.T) {
	t.Parallel()

	t.Run("happy path", func(t *testing.T) {
		t.Parallel()

		var gotFilter db.TimelineFilter

		srv := New(Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					_ context.Context,
					id string,
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
					require.Len(t, timeline, 1)
					assert.Equal(t, "bob", form["assignee"])

					return testDocument(), nil
				},
			},
		})

		owner := "psyb0t"
		formValues := map[string]any{"assignee": "bob"}

		resp, err := srv.RunTemplate(
			context.Background(),
			api.RunTemplateRequestObject{
				Body: &api.RunTemplateJSONRequestBody{
					TemplateId: "t1",
					Filter:     &api.Filter{Owner: &owner},
					FormValues: &formValues,
				},
			},
		)
		require.NoError(t, err)
		assert.Equal(t, "psyb0t", gotFilter.Owner)

		got, ok := resp.(api.RunTemplate200JSONResponse)
		require.True(t, ok)
		require.Len(t, got, 2)
		assert.Equal(t, "heading", got[0].Type)
		assert.Equal(t, "metric", got[1].Type)
	})

	t.Run("missing templateId", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{Store: &fakeStore{}})

		resp, err := srv.RunTemplate(
			context.Background(),
			api.RunTemplateRequestObject{
				Body: &api.RunTemplateJSONRequestBody{},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.RunTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("template not found", func(t *testing.T) {
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

		resp, err := srv.RunTemplate(
			context.Background(),
			api.RunTemplateRequestObject{
				Body: &api.RunTemplateJSONRequestBody{
					TemplateId: "missing",
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.RunTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 404, got.StatusCode)
	})

	t.Run("engine error", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					context.Context,
					string,
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
					return nil, ctxerrors.New("boom")
				},
			},
		})

		resp, err := srv.RunTemplate(
			context.Background(),
			api.RunTemplateRequestObject{
				Body: &api.RunTemplateJSONRequestBody{TemplateId: "t1"},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.RunTemplatedefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 500, got.StatusCode)
	})
}

func TestServer_ExportDocument(t *testing.T) {
	t.Parallel()

	t.Run("inline document as csv", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		apiDoc, err := documentToAPI(testDocument())
		require.NoError(t, err)

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					Document: &apiDoc,
					Format:   api.Csv,
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(exportOctetStreamResponse)
		require.True(t, ok)

		data, err := io.ReadAll(got.Body)
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.Equal(t, int64(len(data)), got.ContentLength)
	})

	t.Run("inline document as json", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		apiDoc, err := documentToAPI(testDocument())
		require.NoError(t, err)

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					Document: &apiDoc,
					Format:   api.Json,
				},
			},
		)
		require.NoError(t, err)

		_, ok := resp.(exportOctetStreamResponse)
		require.True(t, ok)
	})

	t.Run("inline document as pdf", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		apiDoc, err := documentToAPI(testDocument())
		require.NoError(t, err)

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					Document: &apiDoc,
					Format:   api.Pdf,
				},
			},
		)
		require.NoError(t, err)

		_, ok := resp.(exportOctetStreamResponse)
		require.True(t, ok)
	})

	t.Run("resolves via templateId", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{
			Store: &fakeStore{
				getTemplateFn: func(
					context.Context,
					string,
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
					return testDocument(), nil
				},
			},
		})

		templateID := "t1"

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					TemplateId: &templateID,
					Format:     api.Json,
				},
			},
		)
		require.NoError(t, err)

		_, ok := resp.(exportOctetStreamResponse)
		require.True(t, ok)
	})

	t.Run("invalid format", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					Format: api.ExportFormat("bogus"),
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ExportDocumentdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("nil body", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		resp, err := srv.ExportDocument(
			context.Background(), api.ExportDocumentRequestObject{},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ExportDocumentdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("neither document nor templateId", func(t *testing.T) {
		t.Parallel()

		srv := New(Deps{})

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					Format: api.Csv,
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ExportDocumentdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 400, got.StatusCode)
	})

	t.Run("template resolution error", func(t *testing.T) {
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

		templateID := "missing"

		resp, err := srv.ExportDocument(
			context.Background(),
			api.ExportDocumentRequestObject{
				Body: &api.ExportDocumentJSONRequestBody{
					TemplateId: &templateID,
					Format:     api.Csv,
				},
			},
		)
		require.NoError(t, err)

		got, ok := resp.(api.ExportDocumentdefaultJSONResponse)
		require.True(t, ok)
		assert.Equal(t, 404, got.StatusCode)
	})
}
