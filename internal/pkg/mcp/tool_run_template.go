package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
	"github.com/psyb0t/gitrakz/internal/pkg/common/types"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
)

// timelineFilterInput is the tool-facing shape of db.TimelineFilter's
// non-pagination fields — an omitted/zero field disables that filter
// dimension, matching db.TimelineFilter's own zero-value contract.
type timelineFilterInput struct {
	Owner string `json:"owner,omitempty" jsonschema:"GitHub owner"`
	Repo  string `json:"repo,omitempty"  jsonschema:"repo name"`
	// Type is one of: commit, pr, review, issue, release.
	Type string `json:"type,omitempty" jsonschema:"event type"`
	From int64  `json:"from,omitempty" jsonschema:"unix seconds, from"`
	To   int64  `json:"to,omitempty"   jsonschema:"unix seconds, to"`
}

func (f timelineFilterInput) toDB() db.TimelineFilter {
	return db.TimelineFilter{
		Owner: f.Owner,
		Repo:  f.Repo,
		Type:  f.Type,
		From:  f.From,
		To:    f.To,
	}
}

type runTemplateInput struct {
	// TemplateID is the saved template id to run.
	TemplateID string `json:"templateId" jsonschema:"template id"`
	// Filter optionally narrows the timeline the template runs over.
	Filter *timelineFilterInput `json:"filter,omitempty" jsonschema:"filter"`
	// FormValues holds values keyed by the template's form field names.
	FormValues map[string]any `json:"formValues,omitempty"`
}

type runTemplateOutput struct {
	Document []blockDTO `json:"document"`
}

// registerRunTemplateTool adds gitrakz_run_template.
func (t *toolset) registerRunTemplateTool(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameRunTemplate,
		Description: "Run a saved template's transform pipeline over the " +
			"(optionally filtered) timeline and return the rendered document.",
	}, t.runTemplate)
}

func (t *toolset) runTemplate(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in runTemplateInput,
) (*mcpsdk.CallToolResult, runTemplateOutput, error) {
	if in.TemplateID == "" {
		return nil, runTemplateOutput{}, ctxerrors.Wrap(
			commerr.ErrValidationFailed, "templateId is required",
		)
	}

	tmpl, err := t.deps.Store.GetTemplate(ctx, in.TemplateID)
	if err != nil {
		return nil, runTemplateOutput{}, ctxerrors.Wrap(err, "get template")
	}

	filter := db.TimelineFilter{}
	if in.Filter != nil {
		filter = in.Filter.toDB()
	}

	timeline, err := t.queryFullTimeline(ctx, filter)
	if err != nil {
		return nil, runTemplateOutput{}, err
	}

	form := types.FormValues{}
	if in.FormValues != nil {
		form = in.FormValues
	}

	doc, err := t.deps.Engine.Run(ctx, tmpl, timeline, form)
	if err != nil {
		return nil, runTemplateOutput{}, ctxerrors.Wrap(err, "run template")
	}

	dtoDoc, err := documentToDTO(doc)
	if err != nil {
		return nil, runTemplateOutput{}, err
	}

	return nil, runTemplateOutput{Document: dtoDoc}, nil
}
