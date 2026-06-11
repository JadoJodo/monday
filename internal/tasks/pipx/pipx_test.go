package pipx

import (
	"context"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestPipxMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "pipx" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestPipxEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("pipx should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Pipx.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("pipx should be disabled")
	}
}

func TestPipxDryRunListsShort(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("pipx", exec.Output{Stdout: "black 24.0.0\nruff 0.1.0"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("dry run must not report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "list" || last.Args[1] != "--short" {
		t.Errorf("expected `pipx list --short`, got %+v", last)
	}
}

func TestPipxApplyUpgradesAll(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("pipx", exec.Output{Stdout: "upgraded"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "upgrade-all" {
		t.Errorf("expected upgrade-all, got %+v", last.Args)
	}
}

func TestPipxMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["pipx"] = true
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing pipx should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
}
