// Package mas updates Mac App Store applications via the mas CLI.
package mas

import (
	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the mas task. Dry-run lists outdated apps (`mas outdated`); apply
// upgrades them (`mas upgrade`). If mas is not installed the task is skipped.
func New() task.Task {
	return task.NewCommand(task.CommandSpec{
		Name:        "mas",
		Description: "Update Mac App Store applications",
		Bin:         "mas",
		DryArgs:     []string{"outdated"},
		ApplyArgs:   []string{"upgrade"},
		Enabled: func(c config.Config) bool {
			return c.Tasks.Mas.Enabled
		},
	})
}
