// Package notify delivers a run's summary headlessly so rundown is useful when
// invoked by launchd with no terminal attached. Failures always notify; clean
// runs notify only when notify.on_success is set.
package notify

import (
	"context"
	"fmt"
	"strings"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/runner"
)

// Message is a delivery-agnostic notification payload.
type Message struct {
	Title string
	Body  string
	// Failed marks a run with at least one failed task, raising urgency.
	Failed bool
}

// Notifier delivers a Message through one channel.
type Notifier interface {
	// Name identifies the channel (e.g. "macos", "ntfy").
	Name() string
	// Enabled reports whether the channel is turned on for cfg.
	Enabled(cfg config.Config) bool
	// Send delivers msg. A non-nil error is a delivery failure, never fatal to a
	// run.
	Send(ctx context.Context, cfg config.Config, msg Message) error
}

// Default returns the built-in notifiers. cmd is injected into the macOS
// notifier so it stays testable.
func Default(cmd exec.Commander) []Notifier {
	return []Notifier{MacOS(cmd), Ntfy()}
}

// ShouldNotify reports whether a run with the given failure state warrants a
// notification under cfg.
func ShouldNotify(cfg config.Config, failed bool) bool {
	return failed || cfg.Notify.OnSuccess
}

// Dispatch sends msg through every enabled notifier, aggregating delivery
// errors. Disabled notifiers are skipped. A delivery failure never aborts the
// others and is never fatal.
func Dispatch(ctx context.Context, cfg config.Config, msg Message, notifiers ...Notifier) []error {
	var errs []error
	for _, n := range notifiers {
		if !n.Enabled(cfg) {
			continue
		}
		if err := n.Send(ctx, cfg, msg); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", n.Name(), err))
		}
	}
	return errs
}

// FromSummary builds a Message from a completed run.
func FromSummary(sum runner.Summary) Message {
	ok, _, failed := sum.Counts()
	title := fmt.Sprintf("rundown: %d ok", ok)
	if failed > 0 {
		title = fmt.Sprintf("rundown: %d failed", failed)
	}

	lines := make([]string, 0, len(sum.Results))
	for _, r := range sum.Results {
		detail := r.Summary
		if r.Err != nil {
			detail = r.Err.Error()
		}
		lines = append(lines, fmt.Sprintf("%s — %s", r.Name, detail))
	}
	return Message{
		Title:  title,
		Body:   strings.Join(lines, "\n"),
		Failed: failed > 0,
	}
}
