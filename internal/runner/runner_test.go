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
	order   *[]string
}

func (f fakeTask) Name() string               { return f.name }
func (f fakeTask) Description() string        { return "fake " + f.name }
func (f fakeTask) Enabled(config.Config) bool { return f.enabled }
func (f fakeTask) Run(_ context.Context, _ config.Config, opts task.Options) task.Result {
	if f.ran != nil {
		*f.ran = true
	}
	if f.order != nil {
		*f.order = append(*f.order, f.name)
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

// cfgProfiles builds a config with a single weekly (Monday) profile listing the
// given task names.
func cfgProfiles(tasks ...string) config.Config {
	c := config.Default()
	c.Profiles = map[string]config.Profile{
		"weekly": {Days: []string{"monday"}, Tasks: tasks},
	}
	return c
}

func TestNotDueRunsNothing(t *testing.T) {
	ran := false
	reg := regWith(fakeTask{name: "a", enabled: true, ran: &ran})
	sum, err := Run(context.Background(), reg, cfgProfiles("a"), Options{Now: tuesday})
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
	sum, err := Run(context.Background(), reg, cfgProfiles("a", "b"), Options{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	if !ranA {
		t.Error("enabled task a should run")
	}
	if ranB {
		t.Error("disabled task b (listed in profile) should not run")
	}
	if len(sum.Results) != 1 {
		t.Errorf("expected 1 result, got %d", len(sum.Results))
	}
}

func TestTwoProfilesUnionRegistryOrderOnce(t *testing.T) {
	var order []string
	reg := regWith(
		fakeTask{name: "a", enabled: true, order: &order},
		fakeTask{name: "b", enabled: true, order: &order},
		fakeTask{name: "c", enabled: true, order: &order},
	)
	cfg := config.Default()
	cfg.Profiles = map[string]config.Profile{
		// Both due Monday; overlapping on "b". Listed out of registry order.
		"p1": {Days: []string{"monday"}, Tasks: []string{"c", "b"}},
		"p2": {Days: []string{"monday"}, Tasks: []string{"b", "a"}},
	}
	sum, err := Run(context.Background(), reg, cfg, Options{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	// Union {a,b,c} run once each, in registry order.
	if len(order) != 3 || order[0] != "a" || order[1] != "b" || order[2] != "c" {
		t.Errorf("run order = %v, want [a b c]", order)
	}
	if len(sum.Results) != 3 {
		t.Errorf("expected 3 results, got %d", len(sum.Results))
	}
}

func TestUnknownTaskInProfileErrors(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true})
	_, err := Run(context.Background(), reg, cfgProfiles("a", "ghost"), Options{Now: monday})
	if err == nil {
		t.Error("profile referencing an unknown task should error")
	}
}

func TestForceRunsOffSchedule(t *testing.T) {
	ran := false
	reg := regWith(fakeTask{name: "a", enabled: true, ran: &ran})
	_, err := Run(context.Background(), reg, cfgProfiles("a"), Options{Now: tuesday, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !ran {
		t.Error("force should run despite wrong day")
	}
}

func TestOnlyRunsNamedEvenIfDisabledBypassingProfiles(t *testing.T) {
	ranA, ranB := false, false
	reg := regWith(
		fakeTask{name: "a", enabled: false, ran: &ranA},
		fakeTask{name: "b", enabled: true, ran: &ranB},
	)
	// --only bypasses profiles entirely: the profile lists only "b", but --only a
	// runs the disabled "a" and nothing else.
	_, err := Run(context.Background(), reg, cfgProfiles("b"), Options{
		Now: tuesday, Force: true, Only: []string{"a"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !ranA {
		t.Error("--only a should run a even though disabled and not in the profile")
	}
	if ranB {
		t.Error("--only a should not run b")
	}
}

func TestAllEnabledRunsEveryEnabledTaskIgnoringProfilesAndSchedule(t *testing.T) {
	ranA, ranB, ranC := false, false, false
	reg := regWith(
		fakeTask{name: "a", enabled: true, ran: &ranA},  // enabled, in profile
		fakeTask{name: "b", enabled: true, ran: &ranB},  // enabled, NOT in any profile
		fakeTask{name: "c", enabled: false, ran: &ranC}, // disabled
	)
	// Profile lists only "a" and is scheduled for Monday; run on Tuesday with
	// AllEnabled to prove it ignores both the profile membership and the schedule.
	sum, err := Run(context.Background(), reg, cfgProfiles("a"), Options{Now: tuesday, AllEnabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !ranA {
		t.Error("enabled task a should run")
	}
	if !ranB {
		t.Error("enabled task b should run even though no profile lists it")
	}
	if ranC {
		t.Error("disabled task c should not run")
	}
	if len(sum.Results) != 2 {
		t.Errorf("expected 2 results (a, b), got %d", len(sum.Results))
	}
}

func TestOnlyUnknownTaskErrors(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true})
	_, err := Run(context.Background(), reg, cfgProfiles("a"), Options{
		Now: monday, Only: []string{"nope"},
	})
	if err == nil {
		t.Error("unknown --only task should error")
	}
}

func TestDryRunPropagates(t *testing.T) {
	dry := false
	reg := regWith(fakeTask{name: "a", enabled: true, dryRan: &dry})
	_, err := Run(context.Background(), reg, cfgProfiles("a"), Options{Now: monday, DryRun: true})
	if err != nil {
		t.Fatal(err)
	}
	if !dry {
		t.Error("DryRun option should reach the task")
	}
}

func TestSummaryCountsAndFailedNames(t *testing.T) {
	reg := regWith(
		fakeTask{name: "a", enabled: true},
		fakeTask{name: "b", enabled: true, err: errors.New("boom")},
	)
	sum, err := Run(context.Background(), reg, cfgProfiles("a", "b"), Options{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	if !sum.Failed() {
		t.Error("Summary.Failed should be true when a task errors")
	}
	ok, skipped, failed := sum.Counts()
	if ok != 1 || skipped != 0 || failed != 1 {
		t.Errorf("counts = (%d,%d,%d), want (1,0,1)", ok, skipped, failed)
	}
	names := sum.FailedNames()
	if len(names) != 1 || names[0] != "b" {
		t.Errorf("FailedNames = %v, want [b]", names)
	}
	if sum.Started.IsZero() {
		t.Error("Started should be set")
	}
	if sum.Duration < 0 {
		t.Errorf("Duration should be non-negative, got %v", sum.Duration)
	}
}

func TestInvalidDayErrors(t *testing.T) {
	reg := regWith(fakeTask{name: "a", enabled: true})
	if _, err := Run(context.Background(), reg, cfgProfiles("a"), Options{Now: monday, Day: "noday"}); err == nil {
		t.Error("invalid --day weekday should error")
	}
}
