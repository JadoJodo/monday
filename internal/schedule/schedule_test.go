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

func TestEvaluateScheduledToday(t *testing.T) {
	cfg := config.Default() // day: monday
	d, err := Evaluate(cfg, monday, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due {
		t.Errorf("expected due on Monday, got %+v", d)
	}
	if d.Target != time.Monday {
		t.Errorf("target = %v, want Monday", d.Target)
	}
}

func TestEvaluateNotDue(t *testing.T) {
	cfg := config.Default()
	d, err := Evaluate(cfg, tuesday, "", false)
	if err != nil {
		t.Fatal(err)
	}
	if d.Due {
		t.Errorf("expected not due on Tuesday, got %+v", d)
	}
}

func TestEvaluateForce(t *testing.T) {
	cfg := config.Default()
	d, err := Evaluate(cfg, tuesday, "", true)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due || d.Reason != "forced" {
		t.Errorf("force should make due, got %+v", d)
	}
}

func TestEvaluateOverride(t *testing.T) {
	cfg := config.Default() // monday
	d, err := Evaluate(cfg, tuesday, "tuesday", false)
	if err != nil {
		t.Fatal(err)
	}
	if !d.Due || d.Target != time.Tuesday {
		t.Errorf("override to tuesday should be due, got %+v", d)
	}
}

func TestEvaluateInvalidDay(t *testing.T) {
	cfg := config.Config{Schedule: config.ScheduleConfig{Day: "noday"}}
	if _, err := Evaluate(cfg, monday, "", false); err == nil {
		t.Error("expected error for invalid configured day")
	}
	if _, err := Evaluate(config.Default(), monday, "noday", false); err == nil {
		t.Error("expected error for invalid override day")
	}
}
