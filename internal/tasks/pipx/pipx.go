// Package pipx updates pipx-installed Python applications.
package pipx

import (
	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the pipx task. pipx has no outdated-query, so dry-run lists the
// installed apps (`pipx list --short`); apply upgrades them all
// (`pipx upgrade-all`, which is non-interactive).
func New() task.Task {
	return task.NewCommand(task.CommandSpec{
		Name:        "pipx",
		Description: "Update pipx-installed applications",
		Bin:         "pipx",
		DryArgs:     []string{"list", "--short"},
		ApplyArgs:   []string{"upgrade-all"},
		Enabled: func(c config.Config) bool {
			return c.Tasks.Pipx.Enabled
		},
	})
}
