// Package runner orchestrates a maintenance run: it evaluates the schedule,
// selects the tasks to execute (the union of due profiles' tasks intersected
// with the enabled set, or an explicit --only list), runs each one, and
// collects their results.
package runner

import (
	"context"
	"fmt"
	"time"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/schedule"
	"github.com/JadoJodo/monday/internal/task"
)

// Options controls a single run.
type Options struct {
	DryRun     bool
	Only       []string // restrict to these task names (bypasses profiles + enabled filter)
	Force      bool     // run every profile regardless of the schedule
	AllEnabled bool     // run every enabled task, ignoring profiles and the schedule
	Day        string   // pretend today is this weekday (the --day flag)
	Profiles   []string // run exactly these profiles regardless of the day
	Commander  exec.Commander
	Now        time.Time // for schedule evaluation; zero means time.Now()
}

// Summary is the outcome of a run.
type Summary struct {
	Decision schedule.Decision
	Results  []task.Result
	Started  time.Time
	Duration time.Duration
}

// Failed reports whether any executed task ended in error.
func (s Summary) Failed() bool {
	_, _, failed := s.Counts()
	return failed > 0
}

// Counts tallies the results by outcome.
func (s Summary) Counts() (ok, skipped, failed int) {
	for _, r := range s.Results {
		switch {
		case r.Err != nil:
			failed++
		case r.Skipped:
			skipped++
		default:
			ok++
		}
	}
	return ok, skipped, failed
}

// FailedNames returns the names of the tasks that ended in error, in run order.
func (s Summary) FailedNames() []string {
	var names []string
	for _, r := range s.Results {
		if r.Err != nil {
			names = append(names, r.Name)
		}
	}
	return names
}

// Run executes the selected tasks from reg against cfg. The returned error is
// reserved for setup problems (invalid weekday, unknown --only task, a profile
// referencing an unknown task); per-task failures are reported in
// Summary.Results.
func Run(ctx context.Context, reg *registry.Registry, cfg config.Config, opts Options) (Summary, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	cmd := opts.Commander
	if cmd == nil {
		cmd = exec.System{}
	}

	var (
		dec schedule.Decision
		err error
	)
	if opts.AllEnabled {
		// "all enabled" runs every enabled task regardless of profiles or the
		// schedule, so bypass evaluation with a synthetic always-due decision.
		dec = schedule.Decision{Due: true, Reason: "all enabled tasks"}
	} else {
		dec, err = schedule.Evaluate(cfg, schedule.Query{
			Now:      now,
			Day:      opts.Day,
			Force:    opts.Force,
			Profiles: opts.Profiles,
		})
		if err != nil {
			return Summary{}, err
		}
	}
	sum := Summary{Decision: dec, Started: time.Now()}
	if !dec.Due {
		sum.Duration = time.Since(sum.Started)
		return sum, nil
	}

	// Decide the set of task names to run. --only bypasses profiles entirely and
	// runs the named tasks regardless of their enabled state; otherwise the set
	// is the union of the due profiles' tasks, intersected with the enabled set.
	selected := map[string]bool{}
	onlyMode := len(opts.Only) > 0
	switch {
	case onlyMode:
		for _, name := range opts.Only {
			if _, ok := reg.Get(name); !ok {
				return sum, fmt.Errorf("unknown task %q", name)
			}
			selected[name] = true
		}
	case opts.AllEnabled:
		// Select every registered task; the enabled filter below narrows to the
		// enabled set (onlyMode is false, so the filter still applies).
		for _, t := range reg.All() {
			selected[t.Name()] = true
		}
	default:
		for _, pname := range dec.Profiles {
			for _, tname := range cfg.Profiles[pname].Tasks {
				if _, ok := reg.Get(tname); !ok {
					return sum, fmt.Errorf("profile %q references unknown task %q", pname, tname)
				}
				selected[tname] = true
			}
		}
	}

	for _, t := range reg.All() {
		if !selected[t.Name()] {
			continue
		}
		if !onlyMode && !t.Enabled(cfg) {
			continue
		}
		res := t.Run(ctx, cfg, task.Options{DryRun: opts.DryRun, Commander: cmd})
		sum.Results = append(sum.Results, res)
	}
	sum.Duration = time.Since(sum.Started)
	return sum, nil
}
