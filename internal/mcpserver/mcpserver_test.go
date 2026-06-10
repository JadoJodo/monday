package mcpserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/task"
)

type fakeTask struct {
	name    string
	enabled bool
	err     error
}

func (f fakeTask) Name() string               { return f.name }
func (f fakeTask) Description() string        { return "desc " + f.name }
func (f fakeTask) Enabled(config.Config) bool { return f.enabled }
func (f fakeTask) Run(context.Context, config.Config, task.Options) task.Result {
	return task.Result{Name: f.name, Summary: "ran", Err: f.err}
}

func regWith(tasks ...task.Task) *registry.Registry {
	r := registry.New()
	for _, t := range tasks {
		r.Register(t)
	}
	return r
}

func tempConfig(t *testing.T) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "monday.yaml")
	if err := os.WriteFile(path, config.Sample(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestNewConstructs(t *testing.T) {
	if New("1.0.0", registry.Default(), "") == nil {
		t.Fatal("New returned nil")
	}
}

func TestFormatSummary(t *testing.T) {
	out := formatSummary([]task.Result{
		{Name: "a", Summary: "done", Details: []string{"line"}},
		{Name: "b", Skipped: true, Summary: "skipped"},
		{Name: "c", Err: errors.New("boom"), Summary: "failed"},
	})
	if !strings.Contains(out, "a: ok — done") {
		t.Errorf("missing ok line: %s", out)
	}
	if !strings.Contains(out, "b: skipped") {
		t.Errorf("missing skipped line: %s", out)
	}
	if !strings.Contains(out, "failed: boom") {
		t.Errorf("missing failed line: %s", out)
	}
	if !strings.Contains(out, "    line") {
		t.Errorf("missing detail line: %s", out)
	}
}

func TestFormatSummaryEmpty(t *testing.T) {
	if formatSummary(nil) != "no tasks ran" {
		t.Error("empty summary wrong")
	}
}

func TestListTasks(t *testing.T) {
	reg := regWith(fakeTask{name: "x", enabled: true}, fakeTask{name: "y", enabled: false})
	res := listTasks(reg, tempConfig(t))
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "x\tenabled") || !strings.Contains(text, "y\tdisabled") {
		t.Errorf("list output wrong: %s", text)
	}
}

func TestRunSelectedSuccess(t *testing.T) {
	reg := regWith(fakeTask{name: "x", enabled: true})
	res := runSelected(context.Background(), reg, tempConfig(t), []string{"x"}, true)
	if res.IsError {
		t.Errorf("expected success, got error result")
	}
}

func TestRunSelectedFailure(t *testing.T) {
	reg := regWith(fakeTask{name: "x", enabled: true, err: errors.New("nope")})
	res := runSelected(context.Background(), reg, tempConfig(t), []string{"x"}, false)
	if !res.IsError {
		t.Error("expected error result when task fails")
	}
}

// TestRunAllRunsEnabledTaskNotInAnyProfile covers the run_all path
// (runSelected with no task list): an enabled task that no profile references
// must still run, since run_all means "every enabled task".
func TestRunAllRunsEnabledTaskNotInAnyProfile(t *testing.T) {
	// Config whose only profile lists a different task, so "x" is omitted.
	path := filepath.Join(t.TempDir(), "monday.yaml")
	cfgYAML := "profiles:\n  weekly:\n    days: [monday]\n    tasks: [other]\n"
	if err := os.WriteFile(path, []byte(cfgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	reg := regWith(fakeTask{name: "x", enabled: true})
	res := runSelected(context.Background(), reg, path, nil, false)
	if res.IsError {
		t.Errorf("run_all should succeed: %s", res.Content[0].(*mcp.TextContent).Text)
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "x: ok") {
		t.Errorf("enabled task x not referenced by any profile should still run via run_all: %s", text)
	}
}

func TestRunSelectedUnknownTask(t *testing.T) {
	reg := regWith(fakeTask{name: "x", enabled: true})
	res := runSelected(context.Background(), reg, tempConfig(t), []string{"nope"}, false)
	if !res.IsError {
		t.Error("unknown task should yield error result")
	}
}
