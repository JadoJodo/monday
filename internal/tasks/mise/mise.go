// Package mise updates tools managed by mise (formerly rtx).
package mise

import (
	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the mise task. Dry-run lists outdated tools (`mise outdated`,
// which can exit 1 depending on version, tolerated as success); apply upgrades
// them (`mise upgrade`).
//
// Tolerance of exit 1 is dry-run-only: the mutating `mise upgrade` step is left
// strict so a genuine upgrade failure surfaces instead of being reported as
// success.
func New() task.Task {
	return task.NewSteps(task.StepsSpec{
		Name:        "mise",
		Description: "Update mise-managed tools",
		Bin:         "mise",
		Dry:         []task.Step{{Args: []string{"outdated"}, Tolerate: func(code int) bool { return code == 1 }}},
		Apply:       []task.Step{{Args: []string{"upgrade"}}}, // strict: a failed upgrade is reported
		Enabled: func(c config.Config) bool {
			return c.Tasks.Mise.Enabled
		},
	})
}
