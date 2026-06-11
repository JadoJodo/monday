package rustup

import (
	"context"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestRustupMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "rustup" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestRustupEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("rustup should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Rustup.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("rustup should be disabled")
	}
}

func TestRustupDryRunChecks(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("rustup", exec.Output{Stdout: "stable-aarch64-apple-darwin - Up to date"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("dry run must not report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "check" {
		t.Errorf("expected `rustup check`, got %+v", last)
	}
}

func TestRustupApplyUpdates(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("rustup", exec.Output{Stdout: "updated"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "update" {
		t.Errorf("expected update, got %+v", last.Args)
	}
}

func TestRustupMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["rustup"] = true
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing rustup should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
}
