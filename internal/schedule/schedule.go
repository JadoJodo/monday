// Package schedule decides which profiles are due to run on a given day and
// parses weekday names from configuration and flags.
package schedule

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/JadoJodo/rundown/internal/config"
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
	Due      bool
	Profiles []string // profiles selected to run, sorted
	Reason   string
}

// Query carries the inputs to a schedule evaluation.
type Query struct {
	// Now is the current time; its weekday is used unless Day is set.
	Now time.Time
	// Day, when non-empty, pretends today is this weekday (the --day flag).
	Day string
	// Force makes every profile due regardless of the weekday.
	Force bool
	// Profiles, when non-empty, selects exactly these profiles regardless of the
	// weekday (the --profile flag).
	Profiles []string
}

// Evaluate reports which profiles rundown should run.
//
//   - --profile names run those profiles regardless of the day (unknown name is
//     an error).
//   - --force makes every configured profile due.
//   - otherwise a profile is due when today (or the --day pretend-day) is one of
//     its configured days.
func Evaluate(cfg config.Config, q Query) (Decision, error) {
	today := q.Now.Weekday()
	if q.Day != "" {
		wd, err := ParseWeekday(q.Day)
		if err != nil {
			return Decision{}, err
		}
		today = wd
	}

	// Explicit profile selection wins and ignores the weekday.
	if len(q.Profiles) > 0 {
		for _, name := range q.Profiles {
			if _, ok := cfg.Profiles[name]; !ok {
				return Decision{}, fmt.Errorf("unknown profile %q", name)
			}
		}
		profs := slices.Clone(q.Profiles)
		sort.Strings(profs)
		return Decision{
			Due:      true,
			Profiles: profs,
			Reason:   "selected profile(s): " + strings.Join(profs, ", "),
		}, nil
	}

	if q.Force {
		profs := cfg.ProfileNames()
		return Decision{
			Due:      len(profs) > 0,
			Profiles: profs,
			Reason:   "forced",
		}, nil
	}

	var due []string
	for _, name := range cfg.ProfileNames() {
		p := cfg.Profiles[name]
		for _, d := range p.Days {
			wd, err := ParseWeekday(d)
			if err != nil {
				return Decision{}, fmt.Errorf("profile %q: %w", name, err)
			}
			if wd == today {
				due = append(due, name)
				break
			}
		}
	}
	if len(due) > 0 {
		return Decision{
			Due:      true,
			Profiles: due,
			Reason:   fmt.Sprintf("scheduled today (%s): %s", today, strings.Join(due, ", ")),
		}, nil
	}
	return Decision{
		Due:    false,
		Reason: fmt.Sprintf("no profile scheduled for %s", today),
	}, nil
}
