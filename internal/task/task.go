// Package task defines the plugin contract every maintenance module
// implements. A Task is a self-contained unit of work that can be toggled via
// configuration and executed by the runner in either apply or dry-run mode.
package task

import (
	"context"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
)

// Options carries per-run settings into a Task.
type Options struct {
	// DryRun previews actions (e.g. listing outdated packages) without making
	// changes.
	DryRun bool
	// Commander runs external commands. The runner injects exec.System in
	// production and a fake in tests.
	Commander exec.Commander
}

// Result summarizes the outcome of running a Task.
type Result struct {
	Name    string
	Summary string   // one-line human summary
	Details []string // optional extra lines (e.g. command output)
	Changed bool     // true if the task modified the system
	Skipped bool     // true if the task chose not to run (e.g. tool missing)
	Err     error    // non-nil if the task failed
}

// Task is the plugin interface implemented by every maintenance module.
type Task interface {
	// Name is the stable identifier used in config and the CLI (e.g. "npm").
	Name() string
	// Description is a short human-readable summary shown in `rundown list`.
	Description() string
	// Enabled reports whether the task is turned on for the given config.
	Enabled(cfg config.Config) bool
	// Run performs the task. Implementations should not panic; failures are
	// reported via Result.Err and the returned error (the same value).
	Run(ctx context.Context, cfg config.Config, opts Options) Result
}
