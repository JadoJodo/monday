// Package config defines the on-disk configuration for rundown and how it is
// loaded from ~/.rundown.yaml. A missing file yields sensible defaults (all
// built-in tasks enabled, one weekly profile scheduled for Monday).
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"sort"

	"gopkg.in/yaml.v3"
)

// FileName is the default config file name within the user's home directory.
const FileName = ".rundown.yaml"

// TaskConfig holds settings common to a simple toggleable task.
type TaskConfig struct {
	Enabled bool `yaml:"enabled"`
}

// Script is a single user-defined command run by the custom task.
type Script struct {
	Name string `yaml:"name"`
	Run  string `yaml:"run"`
}

// CustomConfig configures the custom (user-script) task.
type CustomConfig struct {
	Enabled bool     `yaml:"enabled"`
	Scripts []Script `yaml:"scripts"`
}

// CleanupConfig configures the cleanup task. Cleanup is report-only today.
type CleanupConfig struct {
	Enabled bool `yaml:"enabled"`
	// Mode is reserved for a future destructive ("apply") cleanup mode; today
	// cleanup never deletes anything, so this field is intentionally unused.
	// Mode string `yaml:"mode"`
}

// Profile is a named bundle of tasks scheduled on one or more weekdays. rundown
// decides which profiles are due on a given day from these.
type Profile struct {
	// Days lists the weekday names (e.g. "monday") this profile runs on.
	Days []string `yaml:"days"`
	// Tasks lists the task names this profile runs, in registry order.
	Tasks []string `yaml:"tasks"`
}

// MacOSNotify configures the native macOS notification delivery.
type MacOSNotify struct {
	Enabled bool `yaml:"enabled"`
}

// NtfyConfig configures ntfy.sh (or a self-hosted ntfy server) delivery.
type NtfyConfig struct {
	Enabled bool   `yaml:"enabled"`
	Server  string `yaml:"server"`
	Topic   string `yaml:"topic"`
	// Priority is min|low|default|high|urgent. On failure an unset/default
	// priority is bumped to high.
	Priority string `yaml:"priority"`
}

// NotifyConfig controls how rundown reports a run's outcome headlessly.
type NotifyConfig struct {
	// OnSuccess sends notifications even when every task succeeds. Failures
	// always notify regardless.
	OnSuccess bool        `yaml:"on_success"`
	MacOS     MacOSNotify `yaml:"macos"`
	Ntfy      NtfyConfig  `yaml:"ntfy"`
}

// TasksConfig groups the per-task configuration blocks.
type TasksConfig struct {
	SoftwareUpdate TaskConfig    `yaml:"softwareupdate"`
	Mas            TaskConfig    `yaml:"mas"`
	Brew           TaskConfig    `yaml:"brew"`
	Npm            TaskConfig    `yaml:"npm"`
	Pipx           TaskConfig    `yaml:"pipx"`
	Rustup         TaskConfig    `yaml:"rustup"`
	Mise           TaskConfig    `yaml:"mise"`
	Custom         CustomConfig  `yaml:"custom"`
	Cleanup        CleanupConfig `yaml:"cleanup"`
	Health         TaskConfig    `yaml:"health"`
}

// Config is the root configuration document.
type Config struct {
	Profiles map[string]Profile `yaml:"profiles"`
	Tasks    TasksConfig        `yaml:"tasks"`
	Notify   NotifyConfig       `yaml:"notify"`
}

// AllTaskNames lists every built-in task name in registry order. It is the
// default task set for the weekly profile.
var AllTaskNames = []string{
	"softwareupdate", "mas", "brew", "npm", "pipx", "rustup", "mise",
	"custom", "cleanup", "health",
}

// Default returns the configuration used when no file exists: every built-in
// task enabled, a single weekly profile on Monday, and macOS notifications on.
func Default() Config {
	tasks := make([]string, len(AllTaskNames))
	copy(tasks, AllTaskNames)
	return Config{
		Profiles: map[string]Profile{
			"weekly": {Days: []string{"monday"}, Tasks: tasks},
		},
		Tasks: TasksConfig{
			SoftwareUpdate: TaskConfig{Enabled: true},
			Mas:            TaskConfig{Enabled: true},
			Brew:           TaskConfig{Enabled: true},
			Npm:            TaskConfig{Enabled: true},
			Pipx:           TaskConfig{Enabled: true},
			Rustup:         TaskConfig{Enabled: true},
			Mise:           TaskConfig{Enabled: true},
			Custom:         CustomConfig{Enabled: true},
			Cleanup:        CleanupConfig{Enabled: true},
			Health:         TaskConfig{Enabled: true},
		},
		Notify: NotifyConfig{
			OnSuccess: false,
			MacOS:     MacOSNotify{Enabled: true},
			Ntfy: NtfyConfig{
				Enabled:  false,
				Server:   "https://ntfy.sh",
				Topic:    "my-rundown",
				Priority: "default",
			},
		},
	}
}

// ProfileNames returns the configured profile names in sorted order.
func (c Config) ProfileNames() []string {
	names := make([]string, 0, len(c.Profiles))
	for name := range c.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// DefaultPath returns the absolute path to the user's config file.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, FileName), nil
}

// Exists reports whether a config file is present at path. A non-existent file
// yields (false, nil); any other stat error is returned.
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Load reads the config at path. A non-existent file is not an error: defaults
// are returned. Values present in the file override the corresponding defaults;
// omitted keys keep their default (so a task is only disabled by an explicit
// "enabled: false"). A user-defined "profiles:" block fully replaces the
// default weekly profile rather than merging into it.
//
// A file using the old "schedule:" schema is rejected with guidance toward
// `rundown config init` — there is no automatic migration.
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
	}

	// Probe the top-level keys to detect the legacy schema and to know whether
	// the user supplied their own profiles (which must replace, not merge with,
	// the defaults — YAML merges into existing maps).
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return cfg, err
	}
	_, hasSchedule := raw["schedule"]
	_, hasProfiles := raw["profiles"]
	if hasSchedule && !hasProfiles {
		return cfg, errors.New(`config uses the old schema; run "rundown config init" to regenerate (your file is preserved until you overwrite it)`)
	}
	if hasProfiles {
		cfg.Profiles = nil
	}

	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

// Marshal renders the config back to YAML.
func (c Config) Marshal() ([]byte, error) {
	return yaml.Marshal(c)
}

// Sample returns an annotated example configuration suitable for writing to a
// fresh ~/.rundown.yaml via `rundown config init`.
func Sample() []byte {
	return []byte(sampleYAML)
}

const sampleYAML = `# rundown configuration (~/.rundown.yaml)
# Run "rundown --help" for usage.

# Profiles bundle tasks onto weekdays. rundown decides which profiles are due
# each day; the launchd agent triggers daily and lets rundown choose.
profiles:
  weekly:
    days: [monday]
    tasks: [softwareupdate, mas, brew, npm, pipx, rustup, mise, custom, cleanup, health]
  # daily:
  #   days: [tuesday, wednesday, thursday, friday]
  #   tasks: [npm, health]

tasks:
  # macOS software updates: softwareupdate -ia
  softwareupdate:
    enabled: true

  # Mac App Store updates via mas: mas upgrade
  mas:
    enabled: true

  # Homebrew: brew update && brew upgrade && brew cleanup
  brew:
    enabled: true

  # npm global package updates: npm -g update
  npm:
    enabled: true

  # pipx package updates: pipx upgrade-all
  pipx:
    enabled: true

  # Rust toolchain updates: rustup update
  rustup:
    enabled: true

  # mise tool updates: mise upgrade
  mise:
    enabled: true

  # Arbitrary user-defined commands, run in order.
  custom:
    enabled: true
    scripts: []

  # Report-only: shows reclaimable disk space, never deletes.
  cleanup:
    enabled: true

  # Read-only system health: disk usage and battery.
  health:
    enabled: true

# How to report a run's outcome. Failures always notify; on_success controls
# whether clean runs notify too.
notify:
  on_success: false
  macos:
    enabled: true
  ntfy:
    enabled: false
    server: https://ntfy.sh
    topic: my-rundown
    priority: default  # min|low|default|high|urgent (bumped to high on failure)
`
