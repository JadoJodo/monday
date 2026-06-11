package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultAllEnabled(t *testing.T) {
	d := Default()
	p, ok := d.Profiles["weekly"]
	if !ok {
		t.Fatalf("default should have a weekly profile, got %+v", d.Profiles)
	}
	if len(p.Days) != 1 || p.Days[0] != "monday" {
		t.Errorf("default weekly days = %v, want [monday]", p.Days)
	}
	if len(p.Tasks) != len(AllTaskNames) {
		t.Errorf("default weekly tasks = %v, want %v", p.Tasks, AllTaskNames)
	}
	tk := d.Tasks
	if !tk.SoftwareUpdate.Enabled || !tk.Mas.Enabled || !tk.Brew.Enabled ||
		!tk.Npm.Enabled || !tk.Pipx.Enabled || !tk.Rustup.Enabled ||
		!tk.Mise.Enabled || !tk.Custom.Enabled || !tk.Cleanup.Enabled || !tk.Health.Enabled {
		t.Errorf("expected all tasks enabled by default, got %+v", tk)
	}
	if !d.Notify.MacOS.Enabled || d.Notify.OnSuccess || d.Notify.Ntfy.Enabled {
		t.Errorf("notify defaults wrong: %+v", d.Notify)
	}
	if d.Notify.Ntfy.Server != "https://ntfy.sh" {
		t.Errorf("ntfy server default = %q", d.Notify.Ntfy.Server)
	}
}

func TestProfileNamesSorted(t *testing.T) {
	c := Config{Profiles: map[string]Profile{"weekly": {}, "daily": {}, "ad-hoc": {}}}
	got := c.ProfileNames()
	want := []string{"ad-hoc", "daily", "weekly"}
	if len(got) != len(want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadMissingFileReturnsDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "does-not-exist.yaml")
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if _, ok := cfg.Profiles["weekly"]; !ok || !cfg.Tasks.Brew.Enabled {
		t.Errorf("missing file should yield defaults, got %+v", cfg)
	}
}

func TestExists(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.yaml")
	if ok, err := Exists(missing); err != nil || ok {
		t.Errorf("Exists(missing) = (%v, %v), want (false, nil)", ok, err)
	}

	present := filepath.Join(dir, "present.yaml")
	if err := os.WriteFile(present, Sample(), 0o644); err != nil {
		t.Fatal(err)
	}
	if ok, err := Exists(present); err != nil || !ok {
		t.Errorf("Exists(present) = (%v, %v), want (true, nil)", ok, err)
	}
}

func TestLoadOverridesAndPreservesDefaults(t *testing.T) {
	yaml := `
profiles:
  daily:
    days: [tuesday, friday]
    tasks: [npm, health]
tasks:
  npm:
    enabled: false
notify:
  ntfy:
    enabled: true
`
	path := filepath.Join(t.TempDir(), "rundown.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	// User profiles fully replace the default weekly profile.
	if _, ok := cfg.Profiles["weekly"]; ok {
		t.Errorf("user profiles should replace defaults, got %+v", cfg.Profiles)
	}
	if p, ok := cfg.Profiles["daily"]; !ok || len(p.Days) != 2 || len(p.Tasks) != 2 {
		t.Errorf("daily profile not parsed: %+v", cfg.Profiles)
	}
	if cfg.Tasks.Npm.Enabled {
		t.Error("npm should be disabled by explicit enabled: false")
	}
	// Tasks omitted from the file keep their enabled default.
	if !cfg.Tasks.SoftwareUpdate.Enabled || !cfg.Tasks.Brew.Enabled {
		t.Error("omitted tasks should remain enabled")
	}
	// Notify sub-fields omitted from the file keep their defaults.
	if !cfg.Notify.Ntfy.Enabled {
		t.Error("ntfy should be enabled from the file")
	}
	if cfg.Notify.Ntfy.Server != "https://ntfy.sh" || !cfg.Notify.MacOS.Enabled {
		t.Errorf("omitted notify fields should keep defaults: %+v", cfg.Notify)
	}
}

func TestLoadLegacyScheduleErrors(t *testing.T) {
	yaml := `
schedule:
  day: monday
tasks:
  npm:
    enabled: true
`
	path := filepath.Join(t.TempDir(), "legacy.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := Load(path)
	if err == nil {
		t.Fatal("legacy schedule schema should error")
	}
	if !strings.Contains(err.Error(), "config init") {
		t.Errorf("error should point at `config init`: %v", err)
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("profiles: : :\n  - bad"), 0o644); err != nil {
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
	p, ok := cfg.Profiles["weekly"]
	if !ok || len(p.Days) == 0 || p.Days[0] != "monday" {
		t.Errorf("sample weekly profile wrong: %+v", cfg.Profiles)
	}
	if len(p.Tasks) != len(AllTaskNames) {
		t.Errorf("sample weekly tasks = %v", p.Tasks)
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

func TestMarshalRoundTrip(t *testing.T) {
	data, err := Default().Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if len(data) == 0 {
		t.Error("marshalled config is empty")
	}
	path := filepath.Join(t.TempDir(), "rt.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("reload marshalled config: %v", err)
	}
	if _, ok := cfg.Profiles["weekly"]; !ok {
		t.Errorf("round-trip lost weekly profile: %+v", cfg.Profiles)
	}
}
