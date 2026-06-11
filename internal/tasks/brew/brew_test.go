package brew

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestBrewMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "brew" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestBrewEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("brew should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Brew.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("brew should be disabled")
	}
}

func TestBrewDryRunUpdateThenOutdated(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("brew", exec.Output{Stdout: "Already up-to-date."}, nil) // update
	fake.AddResult("brew", exec.Output{Stdout: "wget\njq\nripgrep\n"}, nil) // outdated

	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("dry run must not report Changed")
	}
	want := []string{"update", "outdated"}
	if len(fake.Calls) != 2 {
		t.Fatalf("expected 2 calls, got %d", len(fake.Calls))
	}
	for i, w := range want {
		if fake.Calls[i].Args[0] != w {
			t.Errorf("call %d = %v, want %s", i, fake.Calls[i].Args, w)
		}
	}
	if res.Summary != "3 outdated formulae" {
		t.Errorf("summary = %q, want '3 outdated formulae'", res.Summary)
	}
}

func TestBrewDryRunUpToDate(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("brew", exec.Output{Stdout: ""}, nil) // update
	fake.AddResult("brew", exec.Output{Stdout: ""}, nil) // outdated: nothing
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Summary != "everything up to date" {
		t.Errorf("summary = %q, want 'everything up to date'", res.Summary)
	}
}

func TestBrewApplyUpdateUpgradeCleanup(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("brew", exec.Output{Stdout: "updated"}, nil)
	fake.AddResult("brew", exec.Output{Stdout: "upgraded"}, nil)
	fake.AddResult("brew", exec.Output{Stdout: "cleaned"}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should set Changed")
	}
	want := []string{"update", "upgrade", "cleanup"}
	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(fake.Calls))
	}
	for i, w := range want {
		if fake.Calls[i].Args[0] != w {
			t.Errorf("call %d = %v, want %s", i, fake.Calls[i].Args, w)
		}
	}
}

func TestBrewToleratesOutdatedExitOne(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("brew", exec.Output{Stdout: "ok"}, nil)                                   // update
	fake.AddResult("brew", exec.Output{Stdout: "wget\n", ExitCode: 1}, errors.New("exit 1")) // outdated
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Errorf("brew outdated exit 1 should be tolerated, got %v", res.Err)
	}
}

func TestBrewMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["brew"] = true
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing brew should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
	if !strings.Contains(res.Summary, "skipped") {
		t.Errorf("summary should mention skip: %q", res.Summary)
	}
}
