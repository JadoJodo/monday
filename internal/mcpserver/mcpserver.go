// Package mcpserver exposes monday's maintenance tasks to AI agents over the
// Model Context Protocol. Tools are generated from the same task registry the
// CLI uses, so there is a single source of truth.
package mcpserver

import (
	"context"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/runner"
	"github.com/JadoJodo/monday/internal/task"
)

// RunArgs is the input schema shared by the run tools.
type RunArgs struct {
	DryRun bool `json:"dry_run" jsonschema:"preview actions without making changes"`
}

// New builds an MCP server exposing one run tool per task, a run_all tool, and
// a list_tasks tool. cfgPath is the config file to load on each call ("" uses
// the default ~/.monday.yaml). Tasks invoked via MCP always run regardless of
// the weekday schedule (the agent's call is the trigger).
func New(version string, reg *registry.Registry, cfgPath string) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{Name: "monday", Version: version}, nil)

	for _, t := range reg.All() {
		name := t.Name()
		mcp.AddTool(s, &mcp.Tool{
			Name:        "run_" + name,
			Description: t.Description(),
		}, func(ctx context.Context, _ *mcp.CallToolRequest, in RunArgs) (*mcp.CallToolResult, any, error) {
			return runSelected(ctx, reg, cfgPath, []string{name}, in.DryRun), nil, nil
		})
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "run_all",
		Description: "Run all enabled maintenance tasks",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in RunArgs) (*mcp.CallToolResult, any, error) {
		return runSelected(ctx, reg, cfgPath, nil, in.DryRun), nil, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_tasks",
		Description: "List maintenance tasks and whether each is enabled",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return listTasks(reg, cfgPath), nil, nil
	})

	return s
}

func load(cfgPath string) (config.Config, error) {
	if cfgPath == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return config.Config{}, err
		}
		cfgPath = p
	}
	return config.Load(cfgPath)
}

func runSelected(ctx context.Context, reg *registry.Registry, cfgPath string, only []string, dryRun bool) *mcp.CallToolResult {
	cfg, err := load(cfgPath)
	if err != nil {
		return errorResult(err)
	}
	sum, err := runner.Run(ctx, reg, cfg, runner.Options{
		DryRun:    dryRun,
		Only:      only,
		Force:     true, // an explicit agent call always runs
		Commander: exec.System{},
	})
	if err != nil {
		return errorResult(err)
	}
	return &mcp.CallToolResult{
		IsError: sum.Failed(),
		Content: []mcp.Content{&mcp.TextContent{Text: formatSummary(sum.Results)}},
	}
}

func listTasks(reg *registry.Registry, cfgPath string) *mcp.CallToolResult {
	cfg, err := load(cfgPath)
	if err != nil {
		return errorResult(err)
	}
	var b strings.Builder
	for _, t := range reg.All() {
		state := "disabled"
		if t.Enabled(cfg) {
			state = "enabled"
		}
		fmt.Fprintf(&b, "%s\t%s\t%s\n", t.Name(), state, t.Description())
	}
	return textResult(strings.TrimRight(b.String(), "\n"))
}

func formatSummary(results []task.Result) string {
	if len(results) == 0 {
		return "no tasks ran"
	}
	var b strings.Builder
	for _, r := range results {
		status := "ok"
		switch {
		case r.Err != nil:
			status = "failed: " + r.Err.Error()
		case r.Skipped:
			status = "skipped"
		}
		fmt.Fprintf(&b, "%s: %s — %s\n", r.Name, status, r.Summary)
		for _, d := range r.Details {
			fmt.Fprintf(&b, "    %s\n", d)
		}
	}
	return strings.TrimRight(b.String(), "\n")
}

func textResult(text string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func errorResult(err error) *mcp.CallToolResult {
	return &mcp.CallToolResult{
		IsError: true,
		Content: []mcp.Content{&mcp.TextContent{Text: err.Error()}},
	}
}
