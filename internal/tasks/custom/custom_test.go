package custom

import (
	"context"
	"errors"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func cfgWithScripts(scripts ...config.Script) config.Config {
	c := config.Default()
	c.Tasks.Custom.Scripts = scripts
	return c
}

func TestNoScriptsSkips(t *testing.T) {
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: exec.NewFake()})
	if !res.Skipped {
		t.Error("no scripts should skip")
	}
}

func TestDryRunDoesNotExecute(t *testing.T) {
	fake := exec.NewFake()
	cfg := cfgWithScripts(config.Script{Name: "a", Run: "echo hi"})
	res := New().Run(context.Background(), cfg, task.Options{DryRun: true, Commander: fake})
	if len(fake.Calls) != 0 {
		t.Error("dry run must not execute scripts")
	}
	if res.Changed {
		t.Error("dry run should not be Changed")
	}
	if len(res.Details) != 1 {
		t.Errorf("expected 1 detail line, got %v", res.Details)
	}
}

func TestRunsScriptsInOrder(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("sh", exec.Output{Stdout: "one"}, nil)
	fake.AddResult("sh", exec.Output{Stdout: "two"}, nil)
	cfg := cfgWithScripts(
		config.Script{Name: "one", Run: "echo one"},
		config.Script{Name: "two", Run: "echo two"},
	)
	res := New().Run(context.Background(), cfg, task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("should report Changed")
	}
	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(fake.Calls))
	}
	if fake.Calls[0].Args[1] != "echo one" || fake.Calls[1].Args[1] != "echo two" {
		t.Errorf("scripts ran out of order: %+v", fake.Calls)
	}
}

func TestStopsAtFirstFailure(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("sh", exec.Output{Stderr: "fail", ExitCode: 1}, errors.New("exit 1"))
	cfg := cfgWithScripts(
		config.Script{Name: "bad", Run: "false"},
		config.Script{Name: "never", Run: "echo never"},
	)
	res := New().Run(context.Background(), cfg, task.Options{Commander: fake})
	if res.Err == nil {
		t.Error("expected failure error")
	}
	if len(fake.Calls) != 1 {
		t.Errorf("should stop after first failure, ran %d", len(fake.Calls))
	}
}

func TestSkipsEmptyRun(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("sh", exec.Output{}, nil)
	cfg := cfgWithScripts(
		config.Script{Name: "blank", Run: ""},
		config.Script{Name: "real", Run: "echo hi"},
	)
	res := New().Run(context.Background(), cfg, task.Options{Commander: fake})
	if len(fake.Calls) != 1 {
		t.Errorf("empty Run should be skipped, ran %d calls", len(fake.Calls))
	}
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
}
