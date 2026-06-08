// Package launchd generates a LaunchAgent property list so monday can run
// automatically on a schedule via macOS launchd.
package launchd

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
	"time"
)

// Label is the launchd job label used for the monday agent.
const Label = "io.monday.agent"

// PlistConfig describes the LaunchAgent to generate.
type PlistConfig struct {
	Label      string
	Program    string   // absolute path to the monday binary
	Args       []string // arguments passed to Program (e.g. ["run", "--force"])
	Weekday    time.Weekday
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
		<key>Weekday</key>
		<integer>{{.WeekdayInt}}</integer>
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

// templateData mirrors PlistConfig but exposes the weekday as an integer.
type templateData struct {
	PlistConfig
	WeekdayInt int
}

// Plist renders the LaunchAgent XML for c.
func Plist(c PlistConfig) (string, error) {
	if c.Label == "" {
		c.Label = Label
	}
	var buf bytes.Buffer
	if err := plistTmpl.Execute(&buf, templateData{PlistConfig: c, WeekdayInt: int(c.Weekday)}); err != nil {
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
