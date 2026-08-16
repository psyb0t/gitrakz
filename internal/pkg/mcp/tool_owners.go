package mcp

import (
	"context"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/ctxerrors/commerr"
)

type listOwnersInput struct{}

type listOwnersOutput struct {
	Owners []string `json:"owners"`
}

type listReposInput struct {
	Owner string `json:"owner" jsonschema:"GitHub owner (user or org)"`
}

type listReposOutput struct {
	Repos []string `json:"repos"`
}

// registerOwnerTools adds gitrakz_list_owners and gitrakz_list_repos.
func (t *toolset) registerOwnerTools(srv *mcpsdk.Server) {
	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name: toolNameListOwners,
		Description: "List every GitHub owner (user or org) with activity " +
			"ingested into gitrakz.",
	}, t.listOwners)

	mcpsdk.AddTool(srv, &mcpsdk.Tool{
		Name:        toolNameListRepos,
		Description: "List every repo ingested for a given GitHub owner.",
	}, t.listRepos)
}

func (t *toolset) listOwners(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	_ listOwnersInput,
) (*mcpsdk.CallToolResult, listOwnersOutput, error) {
	owners, err := t.deps.Store.ListOwners(ctx)
	if err != nil {
		return nil, listOwnersOutput{}, ctxerrors.Wrap(err, "list owners")
	}

	return nil, listOwnersOutput{Owners: owners}, nil
}

func (t *toolset) listRepos(
	ctx context.Context,
	_ *mcpsdk.CallToolRequest,
	in listReposInput,
) (*mcpsdk.CallToolResult, listReposOutput, error) {
	if in.Owner == "" {
		return nil, listReposOutput{}, ctxerrors.Wrap(
			commerr.ErrValidationFailed, "owner is required",
		)
	}

	repos, err := t.deps.Store.ListRepos(ctx, in.Owner)
	if err != nil {
		return nil, listReposOutput{}, ctxerrors.Wrap(err, "list repos")
	}

	return nil, listReposOutput{Repos: repos}, nil
}
