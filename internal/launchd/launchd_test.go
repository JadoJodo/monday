package launchd

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestPlistContainsKeyFields(t *testing.T) {
	xml, err := Plist(PlistConfig{
		Program:    "/usr/local/bin/rundown",
		Args:       []string{"run"},
		Hour:       9,
		Minute:     30,
		StdoutPath: "/tmp/rundown.log",
		StderrPath: "/tmp/rundown.log",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		"<string>io.rundown.agent</string>",
		"<string>/usr/local/bin/rundown</string>",
		"<string>run</string>",
		"<key>Hour</key>\n\t\t<integer>9</integer>",
		"<key>Minute</key>\n\t\t<integer>30</integer>",
		"/tmp/rundown.log",
	} {
		if !strings.Contains(xml, want) {
			t.Errorf("plist missing %q\n%s", want, xml)
		}
	}
}

func TestPlistIsDailyNoWeekday(t *testing.T) {
	// The agent fires daily; rundown decides which profiles are due, so the plist
	// must NOT pin a weekday and must NOT pass --force.
	xml, _ := Plist(PlistConfig{Program: "/x", Args: []string{"run"}, Hour: 8})
	if strings.Contains(xml, "Weekday") {
		t.Errorf("daily plist should not contain Weekday:\n%s", xml)
	}
	if strings.Contains(xml, "--force") {
		t.Errorf("daily plist should not pass --force:\n%s", xml)
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
