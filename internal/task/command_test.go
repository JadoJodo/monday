package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
)

func spec() CommandSpec {
	return CommandSpec{
		Name:        "demo",
		Description: "demo task",
		Bin:         "demo",
		DryArgs:     []string{"list"},
		ApplyArgs:   []string{"apply"},
		Enabled:     func(config.Config) bool { return true },
	}
}

func TestCommandMetadata(t *testing.T) {
	tk := NewCommand(spec())
	if tk.Name() != "demo" || tk.Description() != "demo task" {
		t.Errorf("metadata wrong: %s / %s", tk.Name(), tk.Description())
	}
	if !tk.Enabled(config.Default()) {
		t.Error("expected enabled")
	}
}

func TestCommandDryRunAndDetails(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "out line\n", Stderr: "err line\n"}, nil)

	res := NewCommand(spec()).Run(context.Background(), config.Default(),
		Options{DryRun: true, Commander: fake})

	if res.Changed {
		t.Error("dry run must not be Changed")
	}
	if fake.Calls[0].Args[0] != "list" {
		t.Errorf("dry-run should use DryArgs, got %v", fake.Calls[0].Args)
	}
	joined := strings.Join(res.Details, "|")
	if !strings.Contains(joined, "out line") || !strings.Contains(joined, "err line") {
		t.Errorf("details should include stdout and stderr: %v", res.Details)
	}
}

func TestCommandApplySetsChanged(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "ok"}, nil)
	res := NewCommand(spec()).Run(context.Background(), config.Default(),
		Options{Commander: fake})
	if !res.Changed {
		t.Error("apply should set Changed")
	}
	if fake.Calls[0].Args[0] != "apply" {
		t.Errorf("apply should use ApplyArgs, got %v", fake.Calls[0].Args)
	}
}

func TestCommandTolerate(t *testing.T) {
	s := spec()
	s.Tolerate = func(code int) bool { return code == 7 }
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{ExitCode: 7}, errors.New("exit 7"))
	res := NewCommand(s).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Err != nil {
		t.Errorf("exit 7 should be tolerated, got %v", res.Err)
	}
}

func TestCommandUntoleratedFailure(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{ExitCode: 2}, errors.New("exit 2"))
	res := NewCommand(spec()).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Err == nil {
		t.Error("untolerated non-zero exit should fail")
	}
	if !strings.Contains(res.Summary, "exit 2") {
		t.Errorf("summary should mention exit code: %q", res.Summary)
	}
}

func TestCommandMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["demo"] = true
	res := NewCommand(spec()).Run(context.Background(), config.Default(), Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing binary should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
}
