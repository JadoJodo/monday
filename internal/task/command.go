package task

import (
	"context"
	"fmt"
	"strings"

	"github.com/JadoJodo/monday/internal/config"
)

// CommandSpec describes a task that runs a single external command, with one
// argument set for dry-run (preview) and another for apply. The four built-in
// system tasks (softwareupdate, mas, npm) are all instances of this shape.
type CommandSpec struct {
	Name        string
	Description string
	// Bin is the executable; if absent from PATH the task is skipped.
	Bin string
	// DryArgs previews changes (e.g. listing outdated packages).
	DryArgs []string
	// ApplyArgs performs the update.
	ApplyArgs []string
	// Enabled reports whether the task is on for a given config.
	Enabled func(config.Config) bool
	// Tolerate optionally reports whether a non-zero exit code should still be
	// treated as success (e.g. `npm outdated` exits 1 when packages are stale).
	Tolerate func(exitCode int) bool
}

type commandTask struct{ spec CommandSpec }

// NewCommand builds a Task that runs a single command per CommandSpec.
func NewCommand(spec CommandSpec) Task { return commandTask{spec: spec} }

func (c commandTask) Name() string                   { return c.spec.Name }
func (c commandTask) Description() string            { return c.spec.Description }
func (c commandTask) Enabled(cfg config.Config) bool { return c.spec.Enabled(cfg) }

func (c commandTask) Run(ctx context.Context, _ config.Config, opts Options) Result {
	res := Result{Name: c.spec.Name}

	if _, err := opts.Commander.LookPath(c.spec.Bin); err != nil {
		res.Skipped = true
		res.Summary = fmt.Sprintf("%q not found on PATH; skipped", c.spec.Bin)
		return res
	}

	args := c.spec.ApplyArgs
	if opts.DryRun {
		args = c.spec.DryArgs
	}

	out, err := opts.Commander.Run(ctx, c.spec.Bin, args...)
	res.Details = outputLines(out.Stdout, out.Stderr)

	if err != nil && c.spec.Tolerate != nil && c.spec.Tolerate(out.ExitCode) {
		err = nil
	}
	if err != nil {
		res.Err = fmt.Errorf("%s %s failed: %w", c.spec.Bin, strings.Join(args, " "), err)
		res.Summary = fmt.Sprintf("failed (exit %d)", out.ExitCode)
		return res
	}

	if opts.DryRun {
		res.Summary = "checked for updates (dry run)"
	} else {
		res.Changed = true
		res.Summary = "completed"
	}
	return res
}

// outputLines merges stdout and stderr into trimmed, non-empty lines for
// display in results.
func outputLines(stdout, stderr string) []string {
	var lines []string
	for _, block := range []string{stdout, stderr} {
		for ln := range strings.SplitSeq(block, "\n") {
			if t := strings.TrimRight(ln, "\r "); strings.TrimSpace(t) != "" {
				lines = append(lines, t)
			}
		}
	}
	return lines
}
