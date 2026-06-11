package mise

import (
	"context"
	"errors"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestMiseMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "mise" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestMiseEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("mise should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Mise.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("mise should be disabled")
	}
}

func TestMiseDryRunOutdated(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mise", exec.Output{Stdout: "node  20.0.0  21.0.0"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("dry run must not report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "outdated" {
		t.Errorf("expected `mise outdated`, got %+v", last)
	}
}

func TestMiseApplyUpgrades(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mise", exec.Output{Stdout: "upgraded"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[0] != "upgrade" {
		t.Errorf("expected upgrade, got %+v", last.Args)
	}
}

func TestMiseToleratesOutdatedExitOne(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mise", exec.Output{Stdout: "node 20 21", ExitCode: 1}, errors.New("exit 1"))
	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Errorf("mise outdated exit 1 should be tolerated, got %v", res.Err)
	}
}

func TestMiseApplyUpgradeFailureSurfaces(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mise", exec.Output{Stderr: "upgrade failed", ExitCode: 1}, errors.New("exit 1"))
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err == nil {
		t.Error("a failing `mise upgrade` (exit 1) must surface as an error, not be tolerated")
	}
}

func TestMiseMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["mise"] = true
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing mise should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
}
