package server

import (
	"encoding/json"
	"testing"

	"github.com/psyb0t/gitrakz/internal/pkg/common/blocks"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/http/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// unmarshalableChan has no JSON encoding — json.Marshal always errors on
// it, so tests use it to force the *ToAPI marshal-error branches that
// can't be reached with any ordinarily-constructed domain value.
type unmarshalableChan = chan int

func TestStringPtrOrNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, stringPtrOrNil(""))
	require.NotNil(t, stringPtrOrNil("x"))
	assert.Equal(t, "x", *stringPtrOrNil("x"))
}

func TestIntPtrOrNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, intPtrOrNil(0))
	require.NotNil(t, intPtrOrNil(5))
	assert.Equal(t, 5, *intPtrOrNil(5))
}

func TestEventToAPI(t *testing.T) {
	t.Parallel()

	ev := types.Event{
		ID:        "commit:a/b:1",
		TS:        100,
		Type:      types.EventTypeCommit,
		Owner:     "a",
		Repo:      "b",
		SHA:       "deadbeef",
		Number:    7,
		Title:     "fix",
		URL:       "https://example.com",
		Additions: 3,
		Deletions: 1,
		Branch:    "main",
	}

	got := eventToAPI(ev)
	assert.Equal(t, "commit:a/b:1", got.Id)
	assert.Equal(t, int64(100), got.Ts)
	assert.Equal(t, api.Commit, got.Type)
	require.NotNil(t, got.Sha)
	assert.Equal(t, "deadbeef", *got.Sha)
	require.NotNil(t, got.Number)
	assert.Equal(t, 7, *got.Number)

	zeroGot := eventToAPI(types.Event{})
	assert.Nil(t, zeroGot.Sha)
	assert.Nil(t, zeroGot.Number)
	assert.Nil(t, zeroGot.Title)
	assert.Nil(t, zeroGot.Url)
	assert.Nil(t, zeroGot.Additions)
	assert.Nil(t, zeroGot.Deletions)
	assert.Nil(t, zeroGot.Branch)
}

func TestSessionToAPI(t *testing.T) {
	t.Parallel()

	got := sessionToAPI(types.Session{
		Owner:           "alice",
		Start:           0,
		End:             7200,
		DurationSeconds: 7200,
	})
	assert.Equal(t, "alice", got.Owner)
	assert.InDelta(t, 2.0, got.DurationHours, 0.0001)
	assert.Equal(t, []api.Event{}, got.Events)
}

func TestFilterToDB(t *testing.T) {
	t.Parallel()

	assert.Zero(t, filterToDB(nil))

	owner, repo := "o", "r"
	evType := api.Pr
	from, to := int64(1), int64(2)

	got := filterToDB(&api.Filter{
		Owner: &owner,
		Repo:  &repo,
		Type:  &evType,
		From:  &from,
		To:    &to,
	})
	assert.Equal(t, "o", got.Owner)
	assert.Equal(t, "r", got.Repo)
	assert.Equal(t, "pr", got.Type)
	assert.Equal(t, int64(1), got.From)
	assert.Equal(t, int64(2), got.To)
}

func TestBlockRoundTrip(t *testing.T) {
	t.Parallel()

	block := blocks.NewMetric("Hours", "40", "h")

	apiBlock, err := blockToAPI(block)
	require.NoError(t, err)
	assert.Equal(t, "metric", apiBlock.Type)
	assert.Equal(t, "Hours", apiBlock.Data["label"])

	back, err := blockFromAPI(apiBlock)
	require.NoError(t, err)
	assert.Equal(t, blocks.BlockTypeMetric, back.Type)

	metric, err := back.AsMetric()
	require.NoError(t, err)
	assert.Equal(t, "Hours", metric.Label)
}

func TestDocumentRoundTrip(t *testing.T) {
	t.Parallel()

	doc := blocks.Document{
		blocks.NewHeading(1, "Title"),
		blocks.NewText("body"),
	}

	apiDoc, err := documentToAPI(doc)
	require.NoError(t, err)
	require.Len(t, apiDoc, 2)

	back, err := documentFromAPI(apiDoc)
	require.NoError(t, err)
	require.Len(t, back, 2)
	assert.Equal(t, blocks.BlockTypeHeading, back[0].Type)
}

func TestFormFieldRoundTrip(t *testing.T) {
	t.Parallel()

	f := template.FormField{
		Key:      "rate",
		Label:    "Hourly rate",
		Type:     template.FieldTypeNumber,
		Required: true,
		Default:  50,
	}

	apiField := formFieldToAPI(f)
	assert.Equal(t, "rate", apiField.Name)
	require.NotNil(t, apiField.Label)
	assert.Equal(t, "Hourly rate", *apiField.Label)
	require.NotNil(t, apiField.Required)
	assert.True(t, *apiField.Required)

	back := formFieldFromAPI(apiField)
	assert.Equal(t, f.Key, back.Key)
	assert.Equal(t, f.Label, back.Label)
	assert.Equal(t, f.Required, back.Required)

	minimal := formFieldFromAPI(api.FormField{Name: "x", Type: "string"})
	assert.Equal(t, "x", minimal.Key)
	assert.False(t, minimal.Required)
	assert.Empty(t, minimal.Label)
}

func TestStepRoundTrip(t *testing.T) {
	t.Parallel()

	step := template.Step{
		Name:   "group-by",
		Params: []byte(`{"by":"owner"}`),
	}

	apiStep, err := stepToAPI(step)
	require.NoError(t, err)
	assert.Equal(t, "group-by", apiStep.Primitive)
	require.NotNil(t, apiStep.Params)
	assert.Equal(t, "owner", (*apiStep.Params)["by"])

	back, err := stepFromAPI(apiStep)
	require.NoError(t, err)
	assert.Equal(t, "group-by", back.Name)
	assert.JSONEq(t, `{"by":"owner"}`, string(back.Params))

	noParams, err := stepToAPI(template.Step{Name: "passthrough"})
	require.NoError(t, err)
	assert.Nil(t, noParams.Params)

	backNoParams, err := stepFromAPI(api.TransformStep{Primitive: "x"})
	require.NoError(t, err)
	assert.Nil(t, backNoParams.Params)
}

func TestLayoutBlockRoundTrip(t *testing.T) {
	t.Parallel()

	lb := template.LayoutBlock{
		Type:   "table",
		Source: "rows",
		Params: []byte(`{"title":"Report"}`),
	}

	apiBlock, err := layoutBlockToAPI(lb)
	require.NoError(t, err)
	assert.Equal(t, "table", apiBlock.Type)
	assert.Equal(t, "rows", apiBlock.Data["source"])
	assert.Equal(t, "Report", apiBlock.Data["title"])

	back, err := layoutBlockFromAPI(apiBlock)
	require.NoError(t, err)
	assert.Equal(t, "table", back.Type)
	assert.Equal(t, "rows", back.Source)
	assert.JSONEq(t, `{"title":"Report"}`, string(back.Params))

	bare, err := layoutBlockToAPI(template.LayoutBlock{Type: "text"})
	require.NoError(t, err)
	assert.Empty(t, bare.Data)

	bareBack, err := layoutBlockFromAPI(
		api.Block{Type: "text", Data: map[string]any{}},
	)
	require.NoError(t, err)
	assert.Empty(t, bareBack.Source)
	assert.Nil(t, bareBack.Params)
}

func TestTemplateRoundTrip(t *testing.T) {
	t.Parallel()

	tmpl := template.Template{
		ID:          "t1",
		Name:        "Timesheet",
		Description: "weekly hours",
		Form: []template.FormField{
			{Key: "rate", Type: template.FieldTypeNumber},
		},
		Transform: []template.Step{{Name: "passthrough"}},
		Layout: []template.LayoutBlock{
			{Type: "table", Source: "rows"},
		},
		Exports: []string{"csv", "pdf"},
		Model:   "gpt",
		Builtin: true,
	}

	apiTmpl, err := templateToAPI(tmpl)
	require.NoError(t, err)
	assert.Equal(t, "t1", apiTmpl.Id)
	require.NotNil(t, apiTmpl.Description)
	assert.Equal(t, "weekly hours", *apiTmpl.Description)
	require.Len(t, apiTmpl.Exports, 2)

	list, err := templatesToAPI([]template.Template{tmpl})
	require.NoError(t, err)
	require.Len(t, list, 1)

	input := api.TemplateInput{
		Name:      apiTmpl.Name,
		Form:      apiTmpl.Form,
		Transform: apiTmpl.Transform,
		Layout:    apiTmpl.Layout,
		Exports:   apiTmpl.Exports,
		Model:     apiTmpl.Model,
	}

	back, err := templateFromInput("t2", input)
	require.NoError(t, err)
	assert.Equal(t, "t2", back.ID)
	assert.Equal(t, "Timesheet", back.Name)
	assert.False(t, back.Builtin)
	assert.Equal(t, "gpt", back.Model)
}

func TestBlockToAPI_UnmarshalError(t *testing.T) {
	t.Parallel()

	_, err := blockToAPI(blocks.Block{
		Type: blocks.BlockTypeText,
		Data: json.RawMessage(`{not valid json`),
	})
	require.Error(t, err)
}

func TestBlockFromAPI_MarshalError(t *testing.T) {
	t.Parallel()

	_, err := blockFromAPI(api.Block{
		Type: "text",
		Data: map[string]any{"bad": make(unmarshalableChan)},
	})
	require.Error(t, err)
}

func TestDocumentToAPI_PropagatesBlockError(t *testing.T) {
	t.Parallel()

	_, err := documentToAPI(blocks.Document{{
		Type: blocks.BlockTypeText,
		Data: json.RawMessage(`{not valid json`),
	}})
	require.Error(t, err)
}

func TestDocumentFromAPI_PropagatesBlockError(t *testing.T) {
	t.Parallel()

	_, err := documentFromAPI(api.Document{{
		Type: "text",
		Data: map[string]any{"bad": make(unmarshalableChan)},
	}})
	require.Error(t, err)
}

func TestStepToAPI_UnmarshalError(t *testing.T) {
	t.Parallel()

	_, err := stepToAPI(template.Step{
		Name:   "group-by",
		Params: json.RawMessage(`{not valid json`),
	})
	require.Error(t, err)
}

func TestStepFromAPI_MarshalError(t *testing.T) {
	t.Parallel()

	bad := map[string]any{"bad": make(unmarshalableChan)}

	_, err := stepFromAPI(api.TransformStep{
		Primitive: "group-by",
		Params:    &bad,
	})
	require.Error(t, err)
}

func TestLayoutBlockToAPI_UnmarshalError(t *testing.T) {
	t.Parallel()

	_, err := layoutBlockToAPI(template.LayoutBlock{
		Type:   "table",
		Params: json.RawMessage(`{not valid json`),
	})
	require.Error(t, err)
}

func TestLayoutBlockFromAPI_MarshalError(t *testing.T) {
	t.Parallel()

	_, err := layoutBlockFromAPI(api.Block{
		Type: "table",
		Data: map[string]any{"bad": make(unmarshalableChan)},
	})
	require.Error(t, err)
}

func TestTemplateToAPI_PropagatesStepError(t *testing.T) {
	t.Parallel()

	_, err := templateToAPI(template.Template{
		Transform: []template.Step{
			{Name: "x", Params: json.RawMessage(`{not valid json`)},
		},
	})
	require.Error(t, err)
}

func TestTemplateToAPI_PropagatesLayoutError(t *testing.T) {
	t.Parallel()

	_, err := templateToAPI(template.Template{
		Layout: []template.LayoutBlock{
			{Type: "x", Params: json.RawMessage(`{not valid json`)},
		},
	})
	require.Error(t, err)
}

func TestTemplatesToAPI_PropagatesError(t *testing.T) {
	t.Parallel()

	_, err := templatesToAPI([]template.Template{{
		Transform: []template.Step{
			{Name: "x", Params: json.RawMessage(`{not valid json`)},
		},
	}})
	require.Error(t, err)
}

func TestTemplateFromInput_PropagatesStepError(t *testing.T) {
	t.Parallel()

	bad := map[string]any{"bad": make(unmarshalableChan)}

	_, err := templateFromInput("t1", api.TemplateInput{
		Name: "x",
		Transform: []api.TransformStep{
			{Primitive: "x", Params: &bad},
		},
	})
	require.Error(t, err)
}

func TestTemplateFromInput_PropagatesLayoutError(t *testing.T) {
	t.Parallel()

	_, err := templateFromInput("t1", api.TemplateInput{
		Name: "x",
		Layout: []api.Block{
			{Type: "x", Data: map[string]any{
				"bad": make(unmarshalableChan),
			}},
		},
	})
	require.Error(t, err)
}
