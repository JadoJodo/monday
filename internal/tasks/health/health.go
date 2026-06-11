// Package health reports read-only macOS system health: disk usage and, on
// laptops, battery charge and cycle count. It never changes anything.
package health

import (
	"context"
	"regexp"
	"strings"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

type healthTask struct{}

// New returns the health task.
func New() task.Task { return healthTask{} }

func (healthTask) Name() string                 { return "health" }
func (healthTask) Description() string          { return "Report system health (disk, battery)" }
func (healthTask) Enabled(c config.Config) bool { return c.Tasks.Health.Enabled }

var (
	// battRe matches the charge and state in `pmset -g batt`, e.g. "98%; charged".
	battRe = regexp.MustCompile(`(\d+)%;\s*(\w+)`)
	// cycleRe matches the cycle count in `ioreg -r -c AppleSmartBattery`.
	cycleRe = regexp.MustCompile(`"CycleCount"\s*=\s*(\d+)`)
)

// Run gathers metrics best-effort. A metric that cannot be read is omitted; if
// none are available the task is skipped. It never reports Changed.
func (healthTask) Run(ctx context.Context, _ config.Config, opts task.Options) task.Result {
	res := task.Result{Name: "health"}
	cmd := opts.Commander
	var parts []string

	// Disk usage of the root volume.
	if out, err := cmd.Run(ctx, "df", "-h", "/"); err == nil {
		res.Details = append(res.Details, "$ df -h /")
		res.Details = appendLines(res.Details, out)
		if pct := diskCapacity(out.Stdout); pct != "" {
			parts = append(parts, "disk "+pct+" used")
		}
	}

	// Battery: omitted entirely on desktops (no internal battery).
	if out, err := cmd.Run(ctx, "pmset", "-g", "batt"); err == nil && strings.Contains(out.Stdout, "InternalBattery") {
		res.Details = append(res.Details, "$ pmset -g batt")
		res.Details = appendLines(res.Details, out)
		if m := battRe.FindStringSubmatch(out.Stdout); m != nil {
			seg := "battery " + m[1] + "%"
			if cycles := batteryCycles(ctx, cmd); cycles != "" {
				seg += " (" + cycles + " cycles)"
			}
			parts = append(parts, seg)
		}
	}

	if len(parts) == 0 {
		res.Skipped = true
		res.Summary = "no health metrics available"
		return res
	}
	res.Summary = strings.Join(parts, ", ")
	return res
}

// diskCapacity returns the used-percentage from `df -h /` output by locating
// the "Capacity" column in the header (macOS also has a "%iused" column, so a
// positional guess would be wrong).
func diskCapacity(stdout string) string {
	lines := strings.Split(stdout, "\n")
	if len(lines) < 2 {
		return ""
	}
	col := -1
	for i, h := range strings.Fields(lines[0]) {
		if strings.EqualFold(h, "Capacity") {
			col = i
			break
		}
	}
	if col < 0 {
		return ""
	}
	fields := strings.Fields(lines[1])
	if col >= len(fields) {
		return ""
	}
	return fields[col]
}

// batteryCycles reads the battery cycle count via ioreg, best-effort.
func batteryCycles(ctx context.Context, cmd exec.Commander) string {
	out, err := cmd.Run(ctx, "ioreg", "-r", "-c", "AppleSmartBattery")
	if err != nil {
		return ""
	}
	if m := cycleRe.FindStringSubmatch(out.Stdout); m != nil {
		return m[1]
	}
	return ""
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
