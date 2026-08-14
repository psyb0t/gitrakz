// Package templates provides gitrakz's built-in template library: a small
// set of deterministic, Builtin=true templates seeded into the templates
// table on every service boot, so the template runner and templates
// manager have something to run before any user or LLM-authored template
// exists.
package templates

import (
	"encoding/json"

	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/common/template"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/aggregate"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/excludeofftime"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/groupby"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/rate"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/sessionize"
	"github.com/psyb0t/gitrakz/internal/pkg/transform/splitbyactivedays"
)

// Built-in template IDs. Stable across boots so re-seeding always upserts
// the same rows (see db.Store.SaveTemplate) instead of accumulating
// duplicates.
const (
	IDActivitySummary       = "builtin-activity-summary"
	IDCommitsPerRepo        = "builtin-commits-per-repo"
	IDWorkSessionsTimesheet = "builtin-work-sessions-timesheet"
)

// Layout block Type values a template.LayoutBlock may carry. Owned by
// engine/layout.go, which keeps its own copy unexported — these are the
// wire-format contract, not an import of that package's internals.
const (
	layoutHeading = "heading"
	layoutTable   = "table"
	layoutMetric  = "metric"
)

// Export formats every built-in offers.
const (
	exportCSV  = "csv"
	exportJSON = "json"
	exportPDF  = "pdf"
)

// groupByParams mirrors the unexported JSON shape transform/groupby.New
// decodes its params from.
type groupByParams struct {
	By groupby.By `json:"by"`
}

// aggregateParams mirrors the unexported JSON shape transform/aggregate.New
// decodes its params from.
type aggregateParams struct {
	Op    aggregate.Op `json:"op"`
	Field string       `json:"field"`
	Key   string       `json:"key"`
}

// headingParams mirrors engine/layout.go's headingParams — the JSON shape
// a "heading" layout block's Params take.
type headingParams struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// metricParams mirrors engine/layout.go's metricParams — the JSON shape a
// "metric" layout block's Params take.
type metricParams struct {
	Label string `json:"label"`
	Unit  string `json:"unit"`
}

// Builtins returns gitrakz's built-in template library. Every call builds
// a fresh slice (and fresh nested slices/maps), so a caller mutating the
// result cannot affect a later call.
func Builtins() []template.Template {
	return []template.Template{
		activitySummary(),
		commitsPerRepo(),
		workSessionsTimesheet(),
	}
}

// defaultExports returns a fresh copy of the export formats every
// built-in offers.
func defaultExports() []string {
	return []string{exportCSV, exportJSON, exportPDF}
}

// mustMarshal encodes v into json.RawMessage for a Step or LayoutBlock
// Params field. Every caller in this file passes one of the plain param
// structs above, so json.Marshal cannot fail for them — a panic here means
// a params type was extended with something non-marshalable, a
// programming error rather than a runtime condition to handle.
func mustMarshal(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		panic(ctxerrors.Wrap(err, "marshal builtin template params"))
	}

	return data
}

// activitySummary groups every recorded event by owner, appends an
// overall total row, and renders both as a table under a heading.
func activitySummary() template.Template {
	return template.Template{
		ID:          IDActivitySummary,
		Name:        "Activity summary",
		Description: "Per-owner activity breakdown with an overall total.",
		Transform: []template.Step{
			{
				Name:   groupby.Name,
				Params: mustMarshal(groupByParams{By: groupby.ByOwner}),
			},
			{
				Name: aggregate.Name,
				Params: mustMarshal(aggregateParams{
					Op:    aggregate.OpSum,
					Field: "count",
					Key:   "total",
				}),
			},
		},
		Layout: []template.LayoutBlock{
			{
				Type: layoutHeading,
				Params: mustMarshal(headingParams{
					Level: 1,
					Text:  "Activity summary",
				}),
			},
			{Type: layoutTable, Source: "rows"},
		},
		Exports: defaultExports(),
		Builtin: true,
	}
}

// commitsPerRepo groups every recorded event by repository, sums code
// churn per repo, and renders the breakdown as a table plus a
// total-events metric.
func commitsPerRepo() template.Template {
	return template.Template{
		ID:          IDCommitsPerRepo,
		Name:        "Commits per repo",
		Description: "Per-repo event counts with total code churn.",
		Transform: []template.Step{
			{
				Name:   groupby.Name,
				Params: mustMarshal(groupByParams{By: groupby.ByRepo}),
			},
			{
				Name: aggregate.Name,
				Params: mustMarshal(aggregateParams{
					Op:    aggregate.OpSum,
					Field: "additions",
					Key:   "total-additions",
				}),
			},
		},
		Layout: []template.LayoutBlock{
			{
				Type: layoutHeading,
				Params: mustMarshal(headingParams{
					Level: 1,
					Text:  "Commits per repo",
				}),
			},
			{Type: layoutTable, Source: "rows"},
			{
				Type:   layoutMetric,
				Source: "count",
				Params: mustMarshal(metricParams{
					Label: "Total events",
					Unit:  "events",
				}),
			},
		},
		Exports: defaultExports(),
		Builtin: true,
	}
}

// workSessionsTimesheet clusters raw events into per-owner work sessions,
// excludes any declared off-hours, buckets the remaining time by active
// day, and estimates hours and cost at an hourly rate collected from the
// run's form.
func workSessionsTimesheet() template.Template {
	return template.Template{
		ID:          IDWorkSessionsTimesheet,
		Name:        "Work sessions timesheet",
		Description: "Per-day hours and cost from clustered work sessions.",
		Form: []template.FormField{
			{
				Key:      "rate",
				Label:    "Hourly rate",
				Type:     template.FieldTypeNumber,
				Required: false,
				Default:  0,
			},
		},
		Transform: []template.Step{
			{Name: sessionize.Name},
			{Name: excludeofftime.Name},
			{Name: splitbyactivedays.Name},
			{Name: rate.Name},
		},
		Layout: []template.LayoutBlock{
			{
				Type: layoutHeading,
				Params: mustMarshal(headingParams{
					Level: 1,
					Text:  "Work sessions timesheet",
				}),
			},
			{Type: layoutTable, Source: "rows"},
			{
				Type:   layoutMetric,
				Source: "dollars",
				Params: mustMarshal(metricParams{
					Label: "Total dollars",
					Unit:  "$",
				}),
			},
		},
		Exports: defaultExports(),
		Builtin: true,
	}
}
