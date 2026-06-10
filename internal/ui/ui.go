// Package ui renders monday's output with lipgloss. Functions operate on
// primitive types (task results, registry data) so the package stays decoupled
// from the runner and command layers. Color is automatically disabled when
// output is not a terminal (lipgloss honors NO_COLOR and TTY detection).
package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/schedule"
	"github.com/JadoJodo/monday/internal/task"
)

var (
	titleStyle   = lipgloss.NewStyle().Bold(true)
	okStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("2")) // green
	failStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("1")) // red
	skipStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("3")) // yellow
	mutedStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")) // grey
	enabledLabel = okStyle.Render("enabled")
	disabledLbl  = mutedStyle.Render("disabled")
)

// Decision renders a one-line description of the schedule decision.
func Decision(d schedule.Decision) string {
	if d.Due {
		return mutedStyle.Render(fmt.Sprintf("· due: %s", d.Reason))
	}
	return mutedStyle.Render(fmt.Sprintf("· skipped: %s (use --force to run now)", d.Reason))
}

// List renders the configured profiles and the available tasks (with each
// task's enabled state) for cfg. Tasks named by a profile but disabled in
// config are marked so the discrepancy is visible.
func List(reg *registry.Registry, cfg config.Config) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Profiles") + "\n")
	if len(cfg.Profiles) == 0 {
		b.WriteString("  " + mutedStyle.Render("(none configured)") + "\n")
	}
	for _, name := range cfg.ProfileNames() {
		p := cfg.Profiles[name]
		b.WriteString(fmt.Sprintf("  %s %s\n",
			titleStyle.Render(name),
			mutedStyle.Render("· "+strings.Join(p.Days, ", "))))
		labels := make([]string, 0, len(p.Tasks))
		for _, tn := range p.Tasks {
			if t, ok := reg.Get(tn); ok && !t.Enabled(cfg) {
				labels = append(labels, mutedStyle.Render(tn+" (disabled)"))
			} else {
				labels = append(labels, tn)
			}
		}
		b.WriteString("    " + mutedStyle.Render("tasks: ") + strings.Join(labels, ", ") + "\n")
	}
	b.WriteString("\n")

	b.WriteString(titleStyle.Render("Tasks") + "\n")
	for _, t := range reg.All() {
		state := disabledLbl
		if t.Enabled(cfg) {
			state = enabledLabel
		}
		b.WriteString(fmt.Sprintf("  %-16s %-9s %s\n",
			t.Name(), state, mutedStyle.Render(t.Description())))
	}
	return strings.TrimRight(b.String(), "\n")
}

// Result renders a single task result. When verbose, command output detail
// lines are included.
func Result(r task.Result, verbose bool) string {
	var marker, label string
	switch {
	case r.Err != nil:
		marker, label = failStyle.Render("✗"), failStyle.Render(r.Name)
	case r.Skipped:
		marker, label = skipStyle.Render("⊘"), skipStyle.Render(r.Name)
	default:
		marker, label = okStyle.Render("✓"), okStyle.Render(r.Name)
	}

	summary := r.Summary
	if r.Err != nil {
		summary = r.Err.Error()
	}

	var b strings.Builder
	b.WriteString(fmt.Sprintf("%s %s — %s", marker, label, summary))
	if verbose && len(r.Details) > 0 {
		for _, ln := range r.Details {
			b.WriteString("\n    " + mutedStyle.Render(ln))
		}
	}
	return b.String()
}

// Results renders every result followed by a count summary line.
func Results(results []task.Result, verbose bool) string {
	var b strings.Builder
	var ok, failed, skipped int
	for _, r := range results {
		b.WriteString(Result(r, verbose) + "\n")
		switch {
		case r.Err != nil:
			failed++
		case r.Skipped:
			skipped++
		default:
			ok++
		}
	}
	b.WriteString(mutedStyle.Render(
		fmt.Sprintf("%d ok · %d skipped · %d failed", ok, skipped, failed)))
	return strings.TrimRight(b.String(), "\n")
}
