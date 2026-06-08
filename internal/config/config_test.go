package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultAllEnabled(t *testing.T) {
	d := Default()
	if d.Schedule.Day != "monday" {
		t.Errorf("default day = %q, want monday", d.Schedule.Day)
	}
	if !d.Tasks.SoftwareUpdate.Enabled || !d.Tasks.Mas.Enabled ||
		!d.Tasks.Npm.Enabled || !d.Tasks.Custom.Enabled {
		t.Errorf("expected all tasks enabled by default, got %+v", d.Tasks)
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if cfg.Schedule != Default().Schedule || cfg.Tasks.SoftwareUpdate != Default().Tasks.SoftwareUpdate ||
		!cfg.Tasks.Npm.Enabled || !cfg.Tasks.Custom.Enabled {
		t.Errorf("missing file should yield defaults, got %+v", cfg)
	}
}

func TestLoadOverridesAndPreservesDefaults(t *testing.T) {
	yaml := `
schedule:
  day: friday
tasks:
  npm:
    enabled: false
  custom:
    enabled: true
    scripts:
      - name: brew
        run: brew upgrade
`
	path := filepath.Join(t.TempDir(), "monday.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Schedule.Day != "friday" {
		t.Errorf("day = %q, want friday", cfg.Schedule.Day)
	}
	if cfg.Tasks.Npm.Enabled {
		t.Error("npm should be disabled by explicit enabled: false")
	}
	// Tasks omitted from the file keep their enabled default.
	if !cfg.Tasks.SoftwareUpdate.Enabled || !cfg.Tasks.Mas.Enabled {
		t.Error("omitted tasks should remain enabled")
	}
	if len(cfg.Tasks.Custom.Scripts) != 1 || cfg.Tasks.Custom.Scripts[0].Run != "brew upgrade" {
		t.Errorf("custom scripts not parsed: %+v", cfg.Tasks.Custom.Scripts)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("schedule: : :\n  - bad"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestSampleRoundTrips(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.yaml")
	if err := os.WriteFile(path, Sample(), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("loading Sample(): %v", err)
	}
	if cfg.Schedule.Day != "monday" {
		t.Errorf("sample day = %q, want monday", cfg.Schedule.Day)
	}
	if len(cfg.Tasks.Custom.Scripts) == 0 {
		t.Error("sample should include a custom script")
	}
}

func TestDefaultPath(t *testing.T) {
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if filepath.Base(p) != FileName {
		t.Errorf("DefaultPath base = %q, want %q", filepath.Base(p), FileName)
	}
}

func TestMarshal(t *testing.T) {
	data, err := Default().Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("marshalled config is empty")
	}
}
