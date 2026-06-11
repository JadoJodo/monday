package task

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
)

func stepsSpec() StepsSpec {
	return StepsSpec{
		Name:        "demo",
		Description: "demo steps",
		Bin:         "demo",
		Dry:         []Step{{Args: []string{"check"}}},
		Apply: []Step{
			{Args: []string{"update"}},
			{Args: []string{"upgrade"}},
			{Args: []string{"clean"}},
		},
		Enabled: func(config.Config) bool { return true },
	}
}

func TestStepsMetadata(t *testing.T) {
	tk := NewSteps(stepsSpec())
	if tk.Name() != "demo" || tk.Description() != "demo steps" {
		t.Errorf("metadata wrong: %s / %s", tk.Name(), tk.Description())
	}
	if !tk.Enabled(config.Default()) {
		t.Error("expected enabled")
	}
}

func TestStepsApplyRunsAllInOrder(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "a"}, nil)
	fake.AddResult("demo", exec.Output{Stdout: "b"}, nil)
	fake.AddResult("demo", exec.Output{Stdout: "c"}, nil)

	res := NewSteps(stepsSpec()).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should set Changed")
	}
	if len(fake.Calls) != 3 {
		t.Fatalf("expected 3 calls, got %d", len(fake.Calls))
	}
	want := [][]string{{"update"}, {"upgrade"}, {"clean"}}
	for i, w := range want {
		if strings.Join(fake.Calls[i].Args, " ") != strings.Join(w, " ") {
			t.Errorf("call %d = %v, want %v", i, fake.Calls[i].Args, w)
		}
	}
}

func TestStepsDryRunUsesDrySteps(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "checked"}, nil)
	res := NewSteps(stepsSpec()).Run(context.Background(), config.Default(), Options{DryRun: true, Commander: fake})
	if res.Changed {
		t.Error("dry run must not be Changed")
	}
	if len(fake.Calls) != 1 || fake.Calls[0].Args[0] != "check" {
		t.Errorf("dry run should run the single dry step, got %v", fake.Calls)
	}
}

func TestStepsStopsOnErrorWithStepIndex(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "ok"}, nil)               // update ok
	fake.AddResult("demo", exec.Output{ExitCode: 2}, errors.New("boom")) // upgrade fails
	res := NewSteps(stepsSpec()).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Err == nil {
		t.Fatal("expected error from failed step")
	}
	if !strings.Contains(res.Err.Error(), "step 2 of 3") {
		t.Errorf("error should mention the failing step index: %v", res.Err)
	}
	if len(fake.Calls) != 2 {
		t.Errorf("should stop after the failing step, got %d calls", len(fake.Calls))
	}
	// A prior apply step (update) already ran, so the system may have changed.
	if !res.Changed {
		t.Error("partial progress should set Changed")
	}
}

func TestStepsPerStepTolerate(t *testing.T) {
	spec := StepsSpec{
		Name: "demo", Description: "d", Bin: "demo",
		Apply: []Step{
			{Args: []string{"a"}, Tolerate: func(c int) bool { return c == 1 }},
			{Args: []string{"b"}},
		},
		Enabled: func(config.Config) bool { return true },
	}
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{ExitCode: 1}, errors.New("exit 1")) // tolerated
	fake.AddResult("demo", exec.Output{Stdout: "ok"}, nil)
	res := NewSteps(spec).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Err != nil {
		t.Errorf("tolerated exit should not fail: %v", res.Err)
	}
	if len(fake.Calls) != 2 {
		t.Errorf("should continue past tolerated step, got %d calls", len(fake.Calls))
	}
}

func TestStepsSummarizeHook(t *testing.T) {
	spec := stepsSpec()
	spec.Summarize = func(dryRun bool, outs []exec.Output) string {
		if dryRun {
			return "preview"
		}
		return "applied " + outs[0].Stdout
	}
	fake := exec.NewFake()
	fake.AddResult("demo", exec.Output{Stdout: "X"}, nil)
	fake.AddResult("demo", exec.Output{Stdout: "Y"}, nil)
	fake.AddResult("demo", exec.Output{Stdout: "Z"}, nil)
	res := NewSteps(spec).Run(context.Background(), config.Default(), Options{Commander: fake})
	if res.Summary != "applied X" {
		t.Errorf("summary = %q, want 'applied X'", res.Summary)
	}
}

func TestStepsMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["demo"] = true
	res := NewSteps(stepsSpec()).Run(context.Background(), config.Default(), Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing binary should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run anything")
	}
}
