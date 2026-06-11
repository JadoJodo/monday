package softwareupdate

import (
	"context"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestArgsAndEnabled(t *testing.T) {
	tk := New()
	if tk.Name() != "softwareupdate" {
		t.Fatalf("name = %q", tk.Name())
	}
	cfg := config.Default()
	if !tk.Enabled(cfg) {
		t.Error("enabled by default")
	}
	cfg.Tasks.SoftwareUpdate.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("should be disabled")
	}
}

func TestDryRunLists(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("softwareupdate", exec.Output{Stdout: "No new software available."}, nil)
	New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	got := fake.Calls[0].Args
	if len(got) != 1 || got[0] != "-l" {
		t.Errorf("dry-run args = %v, want [-l]", got)
	}
}

func TestApplyInstalls(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("softwareupdate", exec.Output{Stdout: "done"}, nil)
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	got := fake.Calls[0].Args
	if len(got) != 2 || got[0] != "-i" || got[1] != "-a" {
		t.Errorf("apply args = %v, want [-i -a]", got)
	}
	if !res.Changed {
		t.Error("apply should set Changed")
	}
}
