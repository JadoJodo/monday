// Package cleanup reports disk space reclaimable by common developer caches.
// It is strictly report-only: it runs each tool in a non-destructive mode and
// never deletes anything. A future mode may opt into actually reclaiming space.
package cleanup

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/task"
)

type cleanupTask struct{}

// New returns the cleanup task.
func New() task.Task { return cleanupTask{} }

func (cleanupTask) Name() string                 { return "cleanup" }
func (cleanupTask) Description() string          { return "Report reclaimable disk space (read-only)" }
func (cleanupTask) Enabled(c config.Config) bool { return c.Tasks.Cleanup.Enabled }

// brewFreeRe captures the figure from brew's "would free approximately N" line.
var brewFreeRe = regexp.MustCompile(`approximately\s+([\d.]+\s?[KMGTP]?i?B)`)

// Run inspects each category best-effort. A category that errors contributes a
// note to Details but never fails the task (Result.Err stays nil); the task is
// purely informational and never reports Changed.
func (cleanupTask) Run(ctx context.Context, _ config.Config, opts task.Options) task.Result {
	res := task.Result{Name: "cleanup"}
	cmd := opts.Commander
	var figures []string // e.g. "brew 503.9MB"

	// 1. Homebrew: parse the "would free approximately N" summary line.
	if _, err := cmd.LookPath("brew"); err == nil {
		out, _ := cmd.Run(ctx, "brew", "cleanup", "--dry-run")
		res.Details = append(res.Details, "$ brew cleanup --dry-run")
		res.Details = appendLines(res.Details, out)
		if m := brewFreeRe.FindStringSubmatch(out.Stdout + "\n" + out.Stderr); m != nil {
			figures = append(figures, "brew "+strings.ReplaceAll(m[1], " ", ""))
		}
	}

	// 2. Docker: sum the RECLAIMABLE column; a stopped daemon is just noted.
	if _, err := cmd.LookPath("docker"); err == nil {
		out, err := cmd.Run(ctx, "docker", "system", "df")
		res.Details = append(res.Details, "$ docker system df")
		if err != nil {
			res.Details = append(res.Details, "  daemon not running; skipped")
		} else {
			res.Details = appendLines(res.Details, out)
			if total, ok := dockerReclaimable(out.Stdout); ok {
				figures = append(figures, "docker "+formatBytes(total))
			}
		}
	}

	// 3. Xcode DerivedData size (read via du so it stays fakeable).
	if home, err := os.UserHomeDir(); err == nil {
		derived := filepath.Join(home, "Library", "Developer", "Xcode", "DerivedData")
		if kb, ok := duKB(ctx, cmd, derived); ok {
			figures = append(figures, "DerivedData "+formatKB(kb))
		}
		// 4. npm cache size (du, not `npm cache verify`, which mutates).
		npmCache := filepath.Join(home, ".npm")
		if kb, ok := duKB(ctx, cmd, npmCache); ok {
			figures = append(figures, "npm "+formatKB(kb))
		}
	}

	if len(figures) == 0 {
		res.Skipped = true
		res.Summary = "no reclaimable space detected"
		return res
	}
	res.Summary = "reclaimable: " + strings.Join(figures, ", ")
	return res
}

// duKB runs `du -sk path` and returns the size in kilobytes. A non-zero exit
// (e.g. the path does not exist) yields ok=false so the category is omitted.
func duKB(ctx context.Context, cmd exec.Commander, path string) (float64, bool) {
	out, err := cmd.Run(ctx, "du", "-sk", path)
	if err != nil {
		return 0, false
	}
	fields := strings.Fields(out.Stdout)
	if len(fields) == 0 {
		return 0, false
	}
	kb, perr := strconv.ParseFloat(fields[0], 64)
	if perr != nil || kb == 0 {
		return 0, false
	}
	return kb, true
}

// dockerReclaimable sums the RECLAIMABLE column of `docker system df` output.
// The reclaimable figure is the last size-shaped token on each data row, which
// is robust to multi-word type names ("Local Volumes") that shift fixed columns
// and to the trailing "(NN%)" annotation.
func dockerReclaimable(stdout string) (float64, bool) {
	lines := strings.Split(stdout, "\n")
	if len(lines) == 0 || !strings.Contains(strings.ToUpper(lines[0]), "RECLAIMABLE") {
		return 0, false
	}
	var total float64
	var any bool
	for _, ln := range lines[1:] {
		last, ok := lastSize(ln)
		if !ok {
			continue
		}
		total += last
		any = true
	}
	return total, any
}

// lastSize returns the bytes of the last size-shaped token in s.
func lastSize(s string) (float64, bool) {
	var val float64
	var ok bool
	for _, f := range strings.Fields(s) {
		if b, good := parseSize(f); good {
			val, ok = b, true
		}
	}
	return val, ok
}

// appendLines appends the command's non-empty output lines, indented.
func appendLines(details []string, out exec.Output) []string {
	for ln := range strings.SplitSeq(out.Stdout+"\n"+out.Stderr, "\n") {
		if t := strings.TrimRight(ln, "\r "); strings.TrimSpace(t) != "" {
			details = append(details, "  "+t)
		}
	}
	return details
}
