package runner

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/task"
)

// fakeTask is a controllable Task for runner tests.
type fakeTask struct {
	name    string
	enabled bool
	ran     *bool
	dryRan  *bool
	err     error
}

func (f fakeTask) Name() string               { return f.name }
func (f fakeTask) Description() string        { return "fake " + f.name }
func (f fakeTask) Enabled(config.Config) bool { return f.enabled }
func (f fakeTask) Run(_ context.Context, _ config.Config, opts task.Options) task.Result {
	if f.ran != nil {
		*f.ran = true
	}
	if opts.DryRun && f.dryRan != nil {
		*f.dryRan = true
	}
	return task.Result{Name: f.name, Err: f.err, Summary: "ok"}
}

var monday = time.Date(2026, time.June, 8, 9, 0, 0, 0, time.UTC)
var tuesday = monday.AddDate(0, 0, 1)

func regWith(tasks ...task.Task) *registry.Registry {
	r := registry.New()
	for _, t := range tasks {
		r.Register(t)
	}
	return r
}

func TestNotDueRunsNothing(t *testing.T) {
	ran := false
	reg := regWith(fakeTask{name: "a", enabled: true, ran: &ran})
	sum, err := Run(context.Background(), reg, config.Default(), Options{Now: tuesday})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Decision.Due {
		t.Error("should not be due on Tuesday")
	}
	if ran {
		t.Error("no task should run when not due")
	}
	if len(sum.Results) != 0 {
		t.Errorf("expected no results, got %v", sum.Results)
	}
}

func TestDueRunsEnabledOnly(t *testing.T) {
	ranA, ranB := false, false
	reg := regWith(
		fakeTask{name: "a", enabled: true, ran: &ranA},
		fakeTask{name: "b", enabled: false, ran: &ranB},
	)
	sum, err := Run(context.Background(), reg, config.Default(), Options{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	if !ranA {
		t.Error("enabled task a should run")
	}
	if ranB {
		t.Error("disabled task b should not run")
	}
	if len(sum.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(sum.Results))
	}
}

func TestForceRunsOffSchedule(t *testing.T) {
	ran := false
	reg := regWith(fakeTask{name: "a", enabled: true, ran: &ran})
	_, err := Run(context.Background(), reg, config.Default(), Options{Now: tuesday, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("force should run despite wrong day")
	}
}

func TestOnlyRunsNamedEvenIfDisabled(t *testing.T) {
	ranA, ranB := false, false
	reg := regWith(
		fakeTask{name: "a", enabled: false, ran: &ranA},
		fakeTask{name: "b", enabled: true, ran: &ranB},
	)
	// Off-schedule + force, restrict to disabled task "a".
	_, err := Run(context.Background(), reg, config.Default(), Options{
		Now: tuesday, Force: true, Only: []string{"a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ranA {
		t.Error("--only a should run a even though disabled")
	}
	if ranB {
		t.Error("--only a should not run b")
	}
}

func TestOnlyUnknownTaskErrors(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true})
	_, err := Run(context.Background(), reg, config.Default(), Options{
		Now: monday, Only: []string{"nope"},
	})
	if err == nil {
		t.Error("unknown --only task should error")
	}
}

func TestDryRunPropagates(t *testing.T) {
	dry := false
	reg := regWith(fakeTask{name: "a", enabled: true, dryRan: &dry})
	_, err := Run(context.Background(), reg, config.Default(), Options{Now: monday, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !dry {
		t.Error("DryRun option should reach the task")
	}
}

func TestSummaryFailed(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true, err: errors.New("boom")})
	sum, err := Run(context.Background(), reg, config.Default(), Options{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	if !sum.Failed() {
		t.Error("Summary.Failed should be true when a task errors")
	}
}

func TestInvalidDayErrors(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true})
	cfg := config.Config{Schedule: config.ScheduleConfig{Day: "noday"}}
	if _, err := Run(context.Background(), reg, cfg, Options{Now: monday}); err == nil {
		t.Error("invalid weekday should error")
	}
}
