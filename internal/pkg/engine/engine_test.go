package engine

import (
	"context"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	ctransform "github.com/psyb0t/gitrakz/internal/pkg/common/transform"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testTimeline() types.Timeline {
	return types.Timeline{
		{
			TS:    1700000000,
			Type:  types.EventTypeCommit,
			Owner: "alice",
			Repo:  "gitrakz",
			Title: "fix bug",
		},
		{
			TS:    1700003600,
			Type:  types.EventTypePR,
			Owner: "bob",
			Repo:  "gitrakz",
			Title: "add feature",
		},
	}
}

func TestEngine_Run_Passthrough(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "passthrough"},
		},
	}

	doc, err := e.Run(context.Background(), tmpl, testTimeline(), nil)
	require.NoError(t, err)
	require.Len(t, doc, 1)

	assert.Equal(t, blocks.BlockTypeTable, doc[0].Type)

	table, err := doc[0].AsTable()
	require.NoError(t, err)
	assert.Len(t, table.Rows, 2)
}

func TestEngine_Run_GroupByTable(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "group-by", Params: []byte(`{"by":"owner"}`)},
		},
		Layout: []template.LayoutBlock{
			{Type: "table", Source: "rows"},
		},
	}

	doc, err := e.Run(context.Background(), tmpl, testTimeline(), nil)
	require.NoError(t, err)
	require.Len(t, doc, 1)

	assert.Equal(t, blocks.BlockTypeTable, doc[0].Type)

	table, err := doc[0].AsTable()
	require.NoError(t, err)
	assert.Len(t, table.Rows, 2)
}

func TestEngine_Run_UnknownStepErrors(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "does-not-exist"},
		},
	}

	_, err := e.Run(context.Background(), tmpl, testTimeline(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ctransform.ErrUnknownPrimitive)
}

func TestEngine_Run_MetricHeadingTextLayout(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "group-by", Params: []byte(`{"by":"owner"}`)},
		},
		Layout: []template.LayoutBlock{
			{
				Type:   "metric",
				Source: "count",
				Params: []byte(`{"label":"Total Events","unit":"events"}`),
			},
			// No Params — label falls back to Source.
			{Type: "metric", Source: "count"},
			{Type: "heading", Params: []byte(`{"level":2,"text":"Report"}`)},
			{Type: "text", Params: []byte(`{"markdown":"body text"}`)},
		},
	}

	doc, err := e.Run(context.Background(), tmpl, testTimeline(), nil)
	require.NoError(t, err)
	require.Len(t, doc, 4)

	assert.Equal(t, blocks.BlockTypeMetric, doc[0].Type)

	metricWithLabel, err := doc[0].AsMetric()
	require.NoError(t, err)
	assert.Equal(t, blocks.Metric{
		Label: "Total Events",
		Value: "2",
		Unit:  "events",
	}, metricWithLabel)

	assert.Equal(t, blocks.BlockTypeMetric, doc[1].Type)

	metricNoLabel, err := doc[1].AsMetric()
	require.NoError(t, err)
	assert.Equal(t,
		blocks.Metric{Label: "count", Value: "2", Unit: ""}, metricNoLabel)

	assert.Equal(t, blocks.BlockTypeHeading, doc[2].Type)

	heading, err := doc[2].AsHeading()
	require.NoError(t, err)
	assert.Equal(t, blocks.Heading{Level: 2, Text: "Report"}, heading)

	assert.Equal(t, blocks.BlockTypeText, doc[3].Type)

	text, err := doc[3].AsText()
	require.NoError(t, err)
	assert.Equal(t, blocks.Text{Markdown: "body text"}, text)
}

func TestEngine_Run_UnknownLayoutTypeErrors(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "passthrough"},
		},
		Layout: []template.LayoutBlock{
			{Type: "bogus"},
		},
	}

	_, err := e.Run(context.Background(), tmpl, testTimeline(), nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrUnknownLayoutType)
}

func TestEngine_Run_PipelineStepRuntimeErrorAborts(t *testing.T) {
	t.Parallel()

	e := NewEngine(registry.Default())

	tmpl := template.Template{
		Transform: []template.Step{
			{Name: "exclude-off-time"},
		},
	}

	// offHours holds a value encoding/json cannot marshal, so
	// exclude-off-time's Apply fails at RUNTIME (not at pipeline-build
	// time, unlike TestEngine_Run_UnknownStepErrors above) — the "run
	// transform pipeline" branch of Engine.Run.
	form := types.FormValues{"offHours": make(chan int)}

	_, err := e.Run(context.Background(), tmpl, testTimeline(), form)
	require.Error(t, err)
}
