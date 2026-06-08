// Package schedule decides whether monday is due to run on a given day and
// parses weekday names from configuration and flags.
package schedule

import (
	"fmt"
	"strings"
	"time"

	"github.com/JadoJodo/monday/internal/config"
)

var weekdays = map[string]time.Weekday{
	"sunday":    time.Sunday,
	"sun":       time.Sunday,
	"monday":    time.Monday,
	"mon":       time.Monday,
	"tuesday":   time.Tuesday,
	"tue":       time.Tuesday,
	"tues":      time.Tuesday,
	"wednesday": time.Wednesday,
	"wed":       time.Wednesday,
	"thursday":  time.Thursday,
	"thu":       time.Thursday,
	"thur":      time.Thursday,
	"thurs":     time.Thursday,
	"friday":    time.Friday,
	"fri":       time.Friday,
	"saturday":  time.Saturday,
	"sat":       time.Saturday,
}

// ParseWeekday converts a weekday name (case-insensitive, full or common
// abbreviation) to a time.Weekday.
func ParseWeekday(s string) (time.Weekday, error) {
	wd, ok := weekdays[strings.ToLower(strings.TrimSpace(s))]
	if !ok {
		return 0, fmt.Errorf("invalid weekday %q", s)
	}
	return wd, nil
}

// Decision is the result of evaluating the schedule.
type Decision struct {
	Due    bool
	Target time.Weekday
	Reason string
}

// Evaluate reports whether monday should run.
//
// override, when non-empty, replaces the configured weekday (the --day flag).
// force makes the run due regardless of the weekday. If the configured or
// overridden weekday cannot be parsed, an error is returned.
func Evaluate(cfg config.Config, now time.Time, override string, force bool) (Decision, error) {
	dayName := cfg.Schedule.Day
	if override != "" {
		dayName = override
	}
	target, err := ParseWeekday(dayName)
	if err != nil {
		return Decision{}, err
	}

	if force {
		return Decision{Due: true, Target: target, Reason: "forced"}, nil
	}
	if now.Weekday() == target {
		return Decision{Due: true, Target: target, Reason: "scheduled today"}, nil
	}
	return Decision{
		Due:    false,
		Target: target,
		Reason: fmt.Sprintf("scheduled for %s, today is %s", target, now.Weekday()),
	}, nil
}
