package main

import (
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/psyb0t/ctxerrors"
	"github.com/psyb0t/gitrakz/internal/pkg/config"
	"github.com/psyb0t/gitrakz/internal/pkg/db"
	mcpserver "github.com/psyb0t/gitrakz/internal/pkg/mcp"
	httpserver "github.com/psyb0t/gitrakz/internal/pkg/services/http-server"
	"github.com/spf13/cobra"
)

// This file is yours - it never gets replaced by framework
// updates. Return your custom CLI commands here.
func commands() []*cobra.Command {
	return []*cobra.Command{
		mcpCommand(),
	}
}

// mcpCommand runs gitrakz's MCP server over stdio, wired against the real
// SQLite store — the entry point for local MCP clients (Claude Code and
// friends) per the project's `.agents/skills/gitrakz` skill. It builds the
// exact same production dependency stack the http-server service mounts at
// /mcp (see internal/pkg/services/http-server.BuildDeps), so both
// transports expose identical tool behavior.
func mcpCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Run gitrakz's MCP server over stdio",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runMCPStdio(cmd)
		},
	}
}

func runMCPStdio(cmd *cobra.Command) error {
	ctx := cmd.Context()

	cfg, err := config.Load()
	if err != nil {
		return ctxerrors.Wrap(err, "load config")
	}

	store, err := db.Open(ctx, cfg.DBPath)
	if err != nil {
		return ctxerrors.Wrap(err, "open db")
	}

	defer func() {
		if closeErr := store.Close(); closeErr != nil {
			cmd.PrintErrln("close db store:", closeErr)
		}
	}()

	if err := httpserver.SeedBuiltinTemplates(ctx, store); err != nil {
		return ctxerrors.Wrap(err, "seed builtin templates")
	}

	deps, err := httpserver.BuildDeps(cfg, store)
	if err != nil {
		return ctxerrors.Wrap(err, "build mcp server dependencies")
	}

	srv := mcpserver.NewServer(mcpserver.Deps{
		Store:          deps.Store,
		Engine:         deps.Engine,
		Sessionizer:    deps.Sessionizer,
		SyncController: deps.SyncController,
	})

	if err := srv.Run(ctx, &mcpsdk.StdioTransport{}); err != nil {
		return ctxerrors.Wrap(err, "run mcp server over stdio")
	}

	return nil
}
