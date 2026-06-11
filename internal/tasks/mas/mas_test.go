package mas

import (
	"context"
	"errors"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

func TestMasArgs(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mas", exec.Output{Stdout: "outdated list"}, nil)
	New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if fake.Calls[0].Args[0] != "outdated" {
		t.Errorf("dry-run args = %v, want [outdated]", fake.Calls[0].Args)
	}

	fake2 := exec.NewFake()
	fake2.AddResult("mas", exec.Output{}, nil)
	New().Run(context.Background(), config.Default(), task.Options{Commander: fake2})
	if fake2.Calls[0].Args[0] != "upgrade" {
		t.Errorf("apply args = %v, want [upgrade]", fake2.Calls[0].Args)
	}
}

func TestMasFailurePropagates(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("mas", exec.Output{Stderr: "boom", ExitCode: 2}, errors.New("exit status 2"))
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err == nil {
		t.Error("expected error to propagate (mas has no Tolerate)")
	}
	if res.Changed {
		t.Error("failed task should not report Changed")
	}
}

func TestMasEnabledToggle(t *testing.T) {
	cfg := config.Default()
	cfg.Tasks.Mas.Enabled = false
	if New().Enabled(cfg) {
		t.Error("mas should be disabled")
	}
}
