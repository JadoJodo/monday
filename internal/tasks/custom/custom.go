// Package custom runs user-defined maintenance commands from configuration.
// Each script is executed via "sh -c" in the order it appears in the config.
package custom

import (
	"context"
	"fmt"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

type customTask struct{}

// New returns the custom task.
func New() task.Task { return customTask{} }

func (customTask) Name() string        { return "custom" }
func (customTask) Description() string { return "Run user-defined maintenance scripts" }

func (customTask) Enabled(c config.Config) bool { return c.Tasks.Custom.Enabled }

// Run executes each configured script via "sh -c". In dry-run mode it reports
// the commands that would run without executing them. Execution stops at the
// first script that fails.
func (customTask) Run(ctx context.Context, cfg config.Config, opts task.Options) task.Result {
	res := task.Result{Name: "custom"}
	scripts := cfg.Tasks.Custom.Scripts

	if len(scripts) == 0 {
		res.Skipped = true
		res.Summary = "no scripts configured; skipped"
		return res
	}

	if opts.DryRun {
		for _, s := range scripts {
			res.Details = append(res.Details, fmt.Sprintf("would run %s: %s", label(s), s.Run))
		}
		res.Summary = fmt.Sprintf("%d script(s) to run (dry run)", len(scripts))
		return res
	}

	ran := 0
	for _, s := range scripts {
		if s.Run == "" {
			continue
		}
		out, err := opts.Commander.Run(ctx, "sh", "-c", s.Run)
		ran++
		res.Details = append(res.Details, fmt.Sprintf("$ %s", s.Run))
		for _, ln := range nonEmptyLines(out.Stdout, out.Stderr) {
			res.Details = append(res.Details, "  "+ln)
		}
		if err != nil {
			res.Err = fmt.Errorf("script %q failed (exit %d): %w", label(s), out.ExitCode, err)
			res.Summary = fmt.Sprintf("script %q failed", label(s))
			res.Changed = ran > 1
			return res
		}
	}

	res.Changed = ran > 0
	res.Summary = fmt.Sprintf("ran %d script(s)", ran)
	return res
}

func label(s config.Script) string {
	if s.Name != "" {
		return s.Name
	}
	return s.Run
}
