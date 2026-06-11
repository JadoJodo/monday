// Package rustup updates the Rust toolchains managed by rustup.
package rustup

import (
	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the rustup task. Dry-run checks for toolchain updates
// (`rustup check`); apply installs them (`rustup update`).
func New() task.Task {
	return task.NewCommand(task.CommandSpec{
		Name:        "rustup",
		Description: "Update Rust toolchains",
		Bin:         "rustup",
		DryArgs:     []string{"check"},
		ApplyArgs:   []string{"update"},
		Enabled: func(c config.Config) bool {
			return c.Tasks.Rustup.Enabled
		},
	})
}
