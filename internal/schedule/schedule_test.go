package schedule

import (
	"testing"
	"time"

	"github.com/JadoJodo/monday/internal/config"
)

func TestParseWeekday(t *testing.T) {
	cases := map[string]time.Weekday{
		"monday":    time.Monday,
		"Monday":    time.Monday,
		"  MONDAY ": time.Monday,
		"mon":       time.Monday,
		"sun":       time.Sunday,
		"sunday":    time.Sunday,
		"thurs":     time.Thursday,
		"saturday":  time.Saturday,
	}
	for in, want := range cases {
		got, err := ParseWeekday(in)
		if err != nil {
			t.Errorf("ParseWeekday(%q) error: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("ParseWeekday(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseWeekdayInvalid(t *testing.T) {
	if _, err := ParseWeekday("someday"); err == nil {
		t.Error("expected error for invalid weekday")
	}
}

// a Monday in 2026 for deterministic tests.
var monday = time.Date(2026, time.June, 8, 9, 0, 0, 0, time.UTC)
var tuesday = monday.AddDate(0, 0, 1)
var friday = monday.AddDate(0, 0, 4)

// multiProfile has a weekly profile (Monday) and a daily profile (Tue-Fri),
// overlapping on no day so unions are easy to assert.
func multiProfile() config.Config {
	c := config.Default()
	c.Profiles = map[string]config.Profile{
		"weekly": {Days: []string{"monday"}, Tasks: []string{"brew"}},
		"daily":  {Days: []string{"tuesday", "wednesday", "thursday", "friday"}, Tasks: []string{"npm"}},
		"extra":  {Days: []string{"monday"}, Tasks: []string{"health"}},
	}
	return c
}

func TestEvaluateScheduledToday(t *testing.T) {
	cfg := config.Default() // weekly: monday
	d, err := Evaluate(cfg, Query{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due {
		t.Errorf("expected due on Monday, got %+v", d)
	}
	if len(d.Profiles) != 1 || d.Profiles[0] != "weekly" {
		t.Errorf("profiles = %v, want [weekly]", d.Profiles)
	}
}

func TestEvaluateNotDue(t *testing.T) {
	d, err := Evaluate(config.Default(), Query{Now: tuesday})
	if err != nil {
		t.Fatal(err)
	}
	if d.Due {
		t.Errorf("expected not due on Tuesday, got %+v", d)
	}
	if len(d.Profiles) != 0 {
		t.Errorf("not-due should select no profiles, got %v", d.Profiles)
	}
}

func TestEvaluateMultiProfileMondayUnion(t *testing.T) {
	d, err := Evaluate(multiProfile(), Query{Now: monday})
	if err != nil {
		t.Fatal(err)
	}
	// weekly and extra both run Monday; daily does not. Sorted order.
	want := []string{"extra", "weekly"}
	if !equal(d.Profiles, want) {
		t.Errorf("Monday profiles = %v, want %v", d.Profiles, want)
	}
}

func TestEvaluateDailyProfileFriday(t *testing.T) {
	d, err := Evaluate(multiProfile(), Query{Now: friday})
	if err != nil {
		t.Fatal(err)
	}
	if !equal(d.Profiles, []string{"daily"}) {
		t.Errorf("Friday profiles = %v, want [daily]", d.Profiles)
	}
}

func TestEvaluateForce(t *testing.T) {
	d, err := Evaluate(multiProfile(), Query{Now: tuesday, Force: true})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due || d.Reason != "forced" {
		t.Errorf("force should make due, got %+v", d)
	}
	if !equal(d.Profiles, []string{"daily", "extra", "weekly"}) {
		t.Errorf("force should select all profiles, got %v", d.Profiles)
	}
}

func TestEvaluateDayOverride(t *testing.T) {
	// Pretend Tuesday is Monday: weekly + extra become due.
	d, err := Evaluate(multiProfile(), Query{Now: tuesday, Day: "monday"})
	if err != nil {
		t.Fatal(err)
	}
	if !equal(d.Profiles, []string{"extra", "weekly"}) {
		t.Errorf("--day monday profiles = %v, want [extra weekly]", d.Profiles)
	}
}

func TestEvaluateExplicitProfile(t *testing.T) {
	// On a non-matching day, an explicit --profile still runs it.
	d, err := Evaluate(multiProfile(), Query{Now: tuesday, Profiles: []string{"weekly"}})
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due || !equal(d.Profiles, []string{"weekly"}) {
		t.Errorf("explicit profile should run, got %+v", d)
	}
}

func TestEvaluateUnknownProfile(t *testing.T) {
	if _, err := Evaluate(multiProfile(), Query{Now: monday, Profiles: []string{"nope"}}); err == nil {
		t.Error("unknown --profile should error")
	}
}

func TestEvaluateInvalidDay(t *testing.T) {
	if _, err := Evaluate(config.Default(), Query{Now: monday, Day: "noday"}); err == nil {
		t.Error("expected error for invalid --day")
	}
}

func TestEvaluateInvalidProfileDay(t *testing.T) {
	cfg := config.Default()
	cfg.Profiles = map[string]config.Profile{"bad": {Days: []string{"noday"}, Tasks: []string{"npm"}}}
	if _, err := Evaluate(cfg, Query{Now: monday}); err == nil {
		t.Error("invalid weekday in a profile should error")
	}
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
