// Package softwareupdate runs macOS system software updates via the
// softwareupdate(8) tool.
package softwareupdate

import (
	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the softwareupdate task. Dry-run lists available updates
// (`softwareupdate -l`); apply installs all of them (`softwareupdate -ia`).
func New() task.Task {
	return task.NewCommand(task.CommandSpec{
		Name:        "softwareupdate",
		Description: "Install macOS system software updates",
		Bin:         "softwareupdate",
		DryArgs:     []string{"-l"},
		ApplyArgs:   []string{"-i", "-a"},
		Enabled: func(c config.Config) bool {
			return c.Tasks.SoftwareUpdate.Enabled
		},
	})
}
