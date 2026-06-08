// Package runner orchestrates a maintenance run: it checks the schedule,
// selects the tasks to execute (enabled set, optionally narrowed by --only),
// runs each one, and collects their results.
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
	DryRun      bool
	Only        []string // restrict to these task names (bypasses enabled filter)
	Force       bool     // run regardless of the schedule
	DayOverride string   // override the configured weekday
	Commander   exec.Commander
	Now         time.Time // for schedule evaluation; zero means time.Now()
}

// Summary is the outcome of a run.
type Summary struct {
	Decision schedule.Decision
	Results  []task.Result
}

// Failed reports whether any executed task ended in error.
func (s Summary) Failed() bool {
	for _, r := range s.Results {
		if r.Err != nil {
			return true
		}
	}
	return false
}

// Run executes the selected tasks from reg against cfg. The returned error is
// reserved for setup problems (invalid weekday, unknown --only task); per-task
// failures are reported in Summary.Results.
func Run(ctx context.Context, reg *registry.Registry, cfg config.Config, opts Options) (Summary, error) {
	now := opts.Now
	if now.IsZero() {
		now = time.Now()
	}
	cmd := opts.Commander
	if cmd == nil {
		cmd = exec.System{}
	}

	dec, err := schedule.Evaluate(cfg, now, opts.DayOverride, opts.Force)
	if err != nil {
		return Summary{}, err
	}
	sum := Summary{Decision: dec}
	if !dec.Due {
		return sum, nil
	}

	only := map[string]bool{}
	for _, name := range opts.Only {
		if _, ok := reg.Get(name); !ok {
			return sum, fmt.Errorf("unknown task %q", name)
		}
		only[name] = true
	}

	for _, t := range reg.All() {
		switch {
		case len(only) > 0:
			// --only: run exactly the named tasks, regardless of enabled state.
			if !only[t.Name()] {
				continue
			}
		default:
			// Default: run every enabled task.
			if !t.Enabled(cfg) {
				continue
			}
		}
		res := t.Run(ctx, cfg, task.Options{DryRun: opts.DryRun, Commander: cmd})
		sum.Results = append(sum.Results, res)
	}
	return sum, nil
}
