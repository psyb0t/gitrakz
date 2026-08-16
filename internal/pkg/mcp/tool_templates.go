package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

type listTemplatesInput struct{}

type listTemplatesOutput struct {
	Templates []templateDTO `json:"templates"`
}

type getTemplateInput struct {
	ID string `json:"id" jsonschema:"the template id"`
}

type getTemplateOutput struct {
	Template templateDTO `json:"template"`
}

// registerTemplateTools adds gitrakz_list_templates and
// gitrakz_get_template.
func (t *toolset) registerTemplateTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameListTemplates,
		Description: "List every saved template, built-in and custom, " +
			"gitrakz can run.",
	}, t.listTemplates)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolNameGetTemplate,
		Description: "Get one saved template by id.",
	}, t.getTemplate)
}

func (t *toolset) listTemplates(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	_ listTemplatesInput,
) (*mcpsdk.CallToolResult, listTemplatesOutput, error) {
	tmpls, err := t.deps.Store.ListTemplates(ctx)
	if err != nil {
		return nil, listTemplatesOutput{}, ctxerrors.Wrap(
			err, "list templates",
		)
	}

	dtos, err := templatesToDTO(tmpls)
	if err != nil {
		return nil, listTemplatesOutput{}, err
	}

	return nil, listTemplatesOutput{Templates: dtos}, nil
}

func (t *toolset) getTemplate(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in getTemplateInput,
) (*mcpsdk.CallToolResult, getTemplateOutput, error) {
	if in.ID == "" {
		return nil, getTemplateOutput{}, ctxerrors.Wrap(
			commerr.ErrValidationFailed, "id is required",
		)
	}

	tmpl, err := t.deps.Store.GetTemplate(ctx, in.ID)
	if err != nil {
		return nil, getTemplateOutput{}, ctxerrors.Wrap(err, "get template")
	}

	dto, err := templateToDTO(tmpl)
	if err != nil {
		return nil, getTemplateOutput{}, err
	}

	return nil, getTemplateOutput{Template: dto}, nil
}
