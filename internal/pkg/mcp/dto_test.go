package mcp

import (
	"encoding/json"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRawToMap(t *testing.T) {
	t.Parallel()

	t.Run("empty raw returns nil", func(t *testing.T) {
		t.Parallel()

		m, err := rawToMap(nil)
		require.NoError(t, err)
		assert.Nil(t, m)
	})

	t.Run("decodes a JSON object", func(t *testing.T) {
		t.Parallel()

		m, err := rawToMap(json.RawMessage(`{"a":1,"b":"x"}`))
		require.NoError(t, err)
		assert.Equal(t, map[string]any{"a": float64(1), "b": "x"}, m)
	})

	t.Run("invalid JSON is an error", func(t *testing.T) {
		t.Parallel()

		_, err := rawToMap(json.RawMessage(`not-json`))
		require.Error(t, err)
	})
}

func TestTemplateToDTO(t *testing.T) {
	t.Parallel()

	t.Run("maps every field", func(t *testing.T) {
		t.Parallel()

		tmpl := template.Template{
			ID:          "t1",
			Name:        "Weekly Report",
			Description: "a report",
			Form: []template.FormField{
				{
					Key: "rate", Label: "Rate",
					Type: template.FieldTypeNumber, Required: true,
				},
			},
			Transform: []template.Step{
				{
					Name:   "sessionize",
					Params: json.RawMessage(`{"gapMinutes":30}`),
				},
			},
			Layout: []template.LayoutBlock{
				{
					Type: "metric", Source: "hours",
					Params: json.RawMessage(`{"unit":"h"}`),
				},
			},
			Exports: []string{"csv", "pdf"},
			Model:   "gpt-5",
			Builtin: true,
		}

		dto, err := templateToDTO(tmpl)
		require.NoError(t, err)

		assert.Equal(t, "t1", dto.ID)
		assert.Equal(t, "Weekly Report", dto.Name)
		require.Len(t, dto.Form, 1)
		assert.Equal(t, "rate", dto.Form[0].Key)
		require.Len(t, dto.Transform, 1)
		assert.Equal(t, "sessionize", dto.Transform[0].Name)

		wantParams := map[string]any{"gapMinutes": float64(30)}
		assert.Equal(t, wantParams, dto.Transform[0].Params)
		require.Len(t, dto.Layout, 1)
		assert.Equal(t, "metric", dto.Layout[0].Type)
		assert.Equal(t, "hours", dto.Layout[0].Source)
		assert.Equal(t, []string{"csv", "pdf"}, dto.Exports)
		assert.True(t, dto.Builtin)
	})

	t.Run("propagates an undecodable step params error", func(t *testing.T) {
		t.Parallel()

		_, err := templateToDTO(template.Template{
			Transform: []template.Step{{Params: json.RawMessage(`not-json`)}},
		})
		require.Error(t, err)
	})

	t.Run("propagates an undecodable layout params error", func(t *testing.T) {
		t.Parallel()

		_, err := templateToDTO(template.Template{
			Layout: []template.LayoutBlock{
				{Params: json.RawMessage(`not-json`)},
			},
		})
		require.Error(t, err)
	})
}

func TestTemplatesToDTO(t *testing.T) {
	t.Parallel()

	dtos, err := templatesToDTO([]template.Template{{ID: "t1"}, {ID: "t2"}})
	require.NoError(t, err)
	require.Len(t, dtos, 2)
	assert.Equal(t, "t1", dtos[0].ID)
	assert.Equal(t, "t2", dtos[1].ID)
}

func TestEventsToDTO(t *testing.T) {
	t.Parallel()

	dtos := eventsToDTO([]types.Event{
		{ID: "e1", Owner: "octocat", Type: types.EventTypeCommit},
	})
	require.Len(t, dtos, 1)
	assert.Equal(t, "e1", dtos[0].ID)
	assert.Equal(t, "octocat", dtos[0].Owner)
	assert.Equal(t, "commit", dtos[0].Type)
}

func TestDocumentToDTO(t *testing.T) {
	t.Parallel()

	t.Run("maps every block", func(t *testing.T) {
		t.Parallel()

		doc := blocks.Document{blocks.NewMetric("Hours", "40", "h")}

		dtos, err := documentToDTO(doc)
		require.NoError(t, err)
		require.Len(t, dtos, 1)
		assert.Equal(t, "metric", dtos[0].Type)
		assert.Equal(t, "40", dtos[0].Data["value"])
	})

	t.Run("propagates an undecodable block data error", func(t *testing.T) {
		t.Parallel()

		doc := blocks.Document{
			{Type: blocks.BlockTypeText, Data: json.RawMessage(`not-json`)},
		}

		_, err := documentToDTO(doc)
		require.Error(t, err)
	})
}
