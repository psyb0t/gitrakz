package template

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplate_JSONRoundTrip(t *testing.T) {
	orig := Template{
		ID:          "timesheet",
		Name:        "Timesheet",
		Description: "hours per day",
		Form: []FormField{
			{
				Key:      "offHours",
				Label:    "Off hours",
				Type:     FieldTypeOffHours,
				Required: true,
			},
			{
				Key:     "rate",
				Label:   "Hourly rate",
				Type:    FieldTypeNumber,
				Default: float64(100),
			},
		},
		Transform: []Step{
			{Name: "sessionize", Params: json.RawMessage(`{"gap":1800}`)},
			{Name: "rate"},
		},
		Layout: []LayoutBlock{
			{Type: "metric", Source: "totalHours"},
			{Type: "table", Source: "rows"},
		},
		Exports: []string{"csv", "pdf"},
		Model:   "glm-4.6",
		Builtin: true,
	}

	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	var got Template
	require.NoError(t, json.Unmarshal(raw, &got))

	assert.Equal(t, orig, got)
	assert.Equal(t, FieldTypeOffHours, got.Form[0].Type)
	assert.Equal(t, "sessionize", got.Transform[0].Name)
}
