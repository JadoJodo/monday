// Package brew updates Homebrew formulae and casks.
package brew

import (
	"fmt"
	"strings"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

// New returns the brew task. Dry-run refreshes metadata and lists outdated
// formulae (`brew update`, `brew outdated`); apply upgrades and prunes
// (`brew update`, `brew upgrade`, `brew cleanup`).
//
// `brew update`, `brew outdated` and `brew cleanup` all exit 1 on benign
// conditions (tap migrations/deprecation notices, stale formulae present,
// nothing to prune), so exit 1 is tolerated on those. The mutating
// `brew upgrade` step is left strict so a genuine upgrade failure surfaces.
func New() task.Task {
	tolerate1 := func(code int) bool { return code == 1 }
	return task.NewSteps(task.StepsSpec{
		Name:        "brew",
		Description: "Update Homebrew formulae and casks",
		Bin:         "brew",
		Dry: []task.Step{
			{Args: []string{"update"}, Tolerate: tolerate1},
			{Args: []string{"outdated"}, Tolerate: tolerate1},
		},
		Apply: []task.Step{
			{Args: []string{"update"}, Tolerate: tolerate1},
			{Args: []string{"upgrade"}},
			{Args: []string{"cleanup"}, Tolerate: tolerate1},
		},
		Enabled: func(c config.Config) bool { return c.Tasks.Brew.Enabled },
		Summarize: func(dryRun bool, outs []exec.Output) string {
			if !dryRun || len(outs) < 2 {
				return ""
			}
			n := countLines(outs[1].Stdout) // outs[1] is `brew outdated`
			if n == 0 {
				return "everything up to date"
			}
			noun := "formulae"
			if n == 1 {
				noun = "formula"
			}
			return fmt.Sprintf("%d outdated %s", n, noun)
		},
	})
}

// countLines counts non-empty lines in s (one outdated formula per line).
func countLines(s string) int {
	n := 0
	for ln := range strings.SplitSeq(s, "\n") {
		if strings.TrimSpace(ln) != "" {
			n++
		}
	}
	return n
}
