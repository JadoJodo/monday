// Package npm updates globally installed npm packages.
package npm

import (
	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/task"
)

// New returns the npm task. Dry-run lists outdated globals
// (`npm -g outdated`); apply updates them (`npm -g update`). `npm outdated`
// exits 1 when packages are stale, which is tolerated as success.
func New() task.Task {
	return task.NewCommand(task.CommandSpec{
		Name:        "npm",
		Description: "Update global npm packages",
		Bin:         "npm",
		DryArgs:     []string{"-g", "outdated"},
		ApplyArgs:   []string{"-g", "update"},
		Enabled: func(c config.Config) bool {
			return c.Tasks.Npm.Enabled
		},
		// `npm outdated` returns exit code 1 when outdated packages exist.
		Tolerate: func(code int) bool { return code == 1 },
	})
}
