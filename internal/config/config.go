// Package config defines the on-disk configuration for monday and how it is
// loaded from ~/.monday.yaml. A missing file yields sensible defaults (all
// built-in tasks enabled, scheduled for Monday).
package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// FileName is the default config file name within the user's home directory.
const FileName = ".monday.yaml"

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

// ScheduleConfig controls when monday considers itself due to run.
type ScheduleConfig struct {
	// Day is the weekday name (e.g. "monday") monday runs on by default.
	Day string `yaml:"day"`
}

// TasksConfig groups the per-task configuration blocks.
type TasksConfig struct {
	SoftwareUpdate TaskConfig   `yaml:"softwareupdate"`
	Mas            TaskConfig   `yaml:"mas"`
	Npm            TaskConfig   `yaml:"npm"`
	Custom         CustomConfig `yaml:"custom"`
}

// Config is the root configuration document.
type Config struct {
	Schedule ScheduleConfig `yaml:"schedule"`
	Tasks    TasksConfig    `yaml:"tasks"`
}

// Default returns the configuration used when no file exists: every built-in
// task enabled and a Monday schedule.
func Default() Config {
	return Config{
		Schedule: ScheduleConfig{Day: "monday"},
		Tasks: TasksConfig{
			SoftwareUpdate: TaskConfig{Enabled: true},
			Mas:            TaskConfig{Enabled: true},
			Npm:            TaskConfig{Enabled: true},
			Custom:         CustomConfig{Enabled: true},
		},
	}
}

// DefaultPath returns the absolute path to the user's config file.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, FileName), nil
}

// Load reads the config at path. A non-existent file is not an error: defaults
// are returned. Values present in the file override the corresponding defaults;
// omitted keys keep their default (so a task is only disabled by an explicit
// "enabled: false").
func Load(path string) (Config, error) {
	cfg := Default()
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, err
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
// fresh ~/.monday.yaml via `monday config init`.
func Sample() []byte {
	return []byte(sampleYAML)
}

const sampleYAML = `# monday configuration (~/.monday.yaml)
# Run "monday --help" for usage.

schedule:
  # Weekday monday runs on by default. Any weekday name is accepted.
  # Override per-invocation with --day <weekday> or --force.
  day: monday

tasks:
  # macOS software updates: softwareupdate -ia
  softwareupdate:
    enabled: true

  # Mac App Store updates via mas: mas upgrade
  mas:
    enabled: true

  # npm global package updates: npm -g update
  npm:
    enabled: true

  # Arbitrary user-defined commands, run in order.
  custom:
    enabled: true
    scripts:
      - name: brew-upgrade
        run: brew upgrade
`
