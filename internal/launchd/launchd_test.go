package launchd

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestPlistContainsKeyFields(t *testing.T) {
	xml, err := Plist(PlistConfig{
		Program:    "/usr/local/bin/monday",
		Args:       []string{"run", "--force"},
		Weekday:    time.Monday,
		Hour:       9,
		Minute:     30,
		StdoutPath: "/tmp/monday.log",
		StderrPath: "/tmp/monday.log",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<string>io.monday.agent</string>",
		"<string>/usr/local/bin/monday</string>",
		"<string>run</string>",
		"<string>--force</string>",
		"<key>Weekday</key>\n\t\t<integer>1</integer>", // Monday == 1
		"<integer>9</integer>",
		"<integer>30</integer>",
		"/tmp/monday.log",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("plist missing %q\n%s", want, xml)
		}
	}
}

func TestPlistWeekdayInteger(t *testing.T) {
	xml, _ := Plist(PlistConfig{Program: "/x", Weekday: time.Sunday})
	if !strings.Contains(xml, "<key>Weekday</key>\n\t\t<integer>0</integer>") {
		t.Errorf("Sunday should map to 0:\n%s", xml)
	}
}

func TestPlistDefaultsLabel(t *testing.T) {
	xml, _ := Plist(PlistConfig{Program: "/x"})
	if !strings.Contains(xml, Label) {
		t.Error("empty label should default to the package Label")
	}
}

func TestAgentPath(t *testing.T) {
	p, err := AgentPath("")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(p) != Label+".plist" {
		t.Errorf("agent path base = %q", filepath.Base(p))
	}
	if !strings.Contains(p, filepath.Join("Library", "LaunchAgents")) {
		t.Errorf("agent path should be under LaunchAgents: %q", p)
	}
}
