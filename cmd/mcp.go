package cmd

import (
	"context"
	"errors"
	"io"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/mcpserver"
	"github.com/JadoJodo/rundown/internal/registry"
)

func newMCPCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:   "mcp",
		Short: "Start the MCP server (stdio) for AI agents",
		Long: "Starts a Model Context Protocol server over stdio, exposing each " +
			"maintenance task as a tool. Configure it in your MCP client as the " +
			"command `rundown mcp`.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			srv := mcpserver.New(version, registry.Default(), gf.configPath)
			err := srv.Run(cmd.Context(), &mcp.StdioTransport{})
			// A closed stdin (EOF) or cancelled context is a normal shutdown
			// for a stdio MCP server, not an error.
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			return err
		},
	}
}
