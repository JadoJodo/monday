package ui

import (
	"errors"
	"strings"
	"testing"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/schedule"
	"github.com/JadoJodo/monday/internal/task"
)

func TestListShowsEnabledState(t *testing.T) {
	cfg := config.Default()
	cfg.Tasks.Npm.Enabled = false
	out := List(registry.Default(), cfg)
	if !strings.Contains(out, "softwareupdate") || !strings.Contains(out, "npm") {
		t.Errorf("list missing tasks: %s", out)
	}
	if !strings.Contains(out, "enabled") || !strings.Contains(out, "disabled") {
		t.Errorf("list missing state labels: %s", out)
	}
}

func TestResultMarkers(t *testing.T) {
	ok := Result(task.Result{Name: "a", Summary: "done"}, false)
	if !strings.Contains(ok, "✓") || !strings.Contains(ok, "done") {
		t.Errorf("ok result wrong: %q", ok)
	}

	fail := Result(task.Result{Name: "b", Err: errors.New("boom")}, false)
	if !strings.Contains(fail, "✗") || !strings.Contains(fail, "boom") {
		t.Errorf("fail result wrong: %q", fail)
	}

	skip := Result(task.Result{Name: "c", Skipped: true, Summary: "skipped"}, false)
	if !strings.Contains(skip, "⊘") {
		t.Errorf("skip result wrong: %q", skip)
	}
}

func TestResultVerboseDetails(t *testing.T) {
	r := task.Result{Name: "a", Summary: "ok", Details: []string{"line one", "line two"}}
	plain := Result(r, false)
	if strings.Contains(plain, "line one") {
		t.Error("non-verbose should hide details")
	}
	verbose := Result(r, true)
	if !strings.Contains(verbose, "line one") || !strings.Contains(verbose, "line two") {
		t.Errorf("verbose should show details: %q", verbose)
	}
}

func TestResultsCounts(t *testing.T) {
	out := Results([]task.Result{
		{Name: "a", Summary: "ok"},
		{Name: "b", Skipped: true},
		{Name: "c", Err: errors.New("x")},
	}, false)
	if !strings.Contains(out, "1 ok · 1 skipped · 1 failed") {
		t.Errorf("counts wrong: %q", out)
	}
}

func TestDecision(t *testing.T) {
	due := Decision(schedule.Decision{Due: true, Reason: "forced"})
	if !strings.Contains(due, "due") {
		t.Errorf("due decision wrong: %q", due)
	}
	skip := Decision(schedule.Decision{Due: false, Reason: "scheduled for Monday, today is Tuesday"})
	if !strings.Contains(skip, "skipped") || !strings.Contains(skip, "--force") {
		t.Errorf("skip decision wrong: %q", skip)
	}
}
