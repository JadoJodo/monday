// Package launchd generates a LaunchAgent property list so rundown can run
// automatically on a schedule via macOS launchd.
package launchd

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
)

// Label is the launchd job label used for the rundown agent.
const Label = "io.rundown.agent"

// LegacyLabel is the launchd job label used by the tool's previous name
// (monday). `install` cleans up an agent left under this label so it does not
// keep firing a binary removed by the rename.
const LegacyLabel = "io.monday.agent"

// PlistConfig describes the LaunchAgent to generate. The agent fires daily at
// Hour:Minute; rundown itself decides which profiles are due, so the plist never
// desyncs from the config and launchd coalesces runs missed while asleep.
type PlistConfig struct {
	Label      string
	Program    string   // absolute path to the rundown binary
	Args       []string // arguments passed to Program (e.g. ["run"])
	Hour       int
	Minute     int
	StdoutPath string
	StderrPath string
}

var plistTmpl = template.Must(template.New("plist").Parse(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>{{.Label}}</string>
	<key>ProgramArguments</key>
	<array>
		<string>{{.Program}}</string>
{{- range .Args}}
		<string>{{.}}</string>
{{- end}}
	</array>
	<key>StartCalendarInterval</key>
	<dict>
		<key>Hour</key>
		<integer>{{.Hour}}</integer>
		<key>Minute</key>
		<integer>{{.Minute}}</integer>
	</dict>
	<key>StandardOutPath</key>
	<string>{{.StdoutPath}}</string>
	<key>StandardErrorPath</key>
	<string>{{.StderrPath}}</string>
</dict>
</plist>
`))

// Plist renders the LaunchAgent XML for c.
func Plist(c PlistConfig) (string, error) {
	if c.Label == "" {
		c.Label = Label
	}
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, c); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// AgentPath returns the per-user LaunchAgents path for the given label.
func AgentPath(label string) (string, error) {
	if label == "" {
		label = Label
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "LaunchAgents", label+".plist"), nil
}
