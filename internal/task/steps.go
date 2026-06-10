package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
)

// Step is a single command invocation within a multi-step task. All steps in a
// StepsSpec share the same Bin.
type Step struct {
	// Args are passed to the spec's Bin.
	Args []string
	// Tolerate optionally reports whether a non-zero exit code should be treated
	// as success (e.g. `brew outdated` exits 1 when formulae are stale).
	Tolerate func(exitCode int) bool
}

// StepsSpec describes a task that runs a sequence of commands against a single
// Bin, with one sequence for dry-run (preview) and another for apply. It is the
// multi-command sibling of CommandSpec; brew (update/upgrade/cleanup) is the
// canonical user.
type StepsSpec struct {
	Name, Description, Bin string
	// Dry runs in dry-run mode; Apply runs otherwise.
	Dry, Apply []Step
	// Enabled reports whether the task is on for a given config.
	Enabled func(config.Config) bool
	// Summarize optionally builds the one-line summary from the executed step
	// outputs (one entry per step, in order). Returning "" falls back to the
	// generic summary. nil means always use the generic summary.
	Summarize func(dryRun bool, stepOutputs []exec.Output) string
}

type stepsTask struct{ spec StepsSpec }

// NewSteps builds a Task that runs a sequence of commands per StepsSpec.
func NewSteps(spec StepsSpec) Task { return stepsTask{spec: spec} }

func (s stepsTask) Name() string                   { return s.spec.Name }
func (s stepsTask) Description() string            { return s.spec.Description }
func (s stepsTask) Enabled(cfg config.Config) bool { return s.spec.Enabled(cfg) }

func (s stepsTask) Run(ctx context.Context, _ config.Config, opts Options) Result {
	res := Result{Name: s.spec.Name}

	if _, err := opts.Commander.LookPath(s.spec.Bin); err != nil {
		res.Skipped = true
		res.Summary = fmt.Sprintf("%q not found on PATH; skipped", s.spec.Bin)
		return res
	}

	steps := s.spec.Apply
	if opts.DryRun {
		steps = s.spec.Dry
	}

	outputs := make([]exec.Output, 0, len(steps))
	for i, step := range steps {
		out, err := opts.Commander.Run(ctx, s.spec.Bin, step.Args...)
		outputs = append(outputs, out)
		res.Details = append(res.Details, fmt.Sprintf("$ %s %s", s.spec.Bin, strings.Join(step.Args, " ")))
		for _, ln := range outputLines(out.Stdout, out.Stderr) {
			res.Details = append(res.Details, "  "+ln)
		}

		if err != nil && step.Tolerate != nil && step.Tolerate(out.ExitCode) {
			err = nil
		}
		if err != nil {
			res.Err = fmt.Errorf("%s %s failed (step %d of %d): %w",
				s.spec.Bin, strings.Join(step.Args, " "), i+1, len(steps), err)
			res.Summary = fmt.Sprintf("failed (exit %d)", out.ExitCode)
			// In apply mode, a non-first step failing means an earlier step
			// already ran successfully, so the system may have changed.
			if !opts.DryRun && i > 0 {
				res.Changed = true
			}
			return res
		}
	}

	if opts.DryRun {
		res.Summary = "checked for updates (dry run)"
	} else {
		res.Changed = len(steps) > 0
		res.Summary = "completed"
	}
	if s.spec.Summarize != nil {
		if summary := s.spec.Summarize(opts.DryRun, outputs); summary != "" {
			res.Summary = summary
		}
	}
	return res
}
