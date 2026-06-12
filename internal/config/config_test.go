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

func TestToggleableTaskNamesExcludesCustom(t *testing.T) {
	names := ToggleableTaskNames()
	if len(names) != len(AllTaskNames)-1 {
		t.Fatalf("ToggleableTaskNames len = %d, want %d", len(names), len(AllTaskNames)-1)
	}
	for _, n := range names {
		if n == "custom" {
			t.Errorf("ToggleableTaskNames should exclude custom, got %v", names)
		}
	}
	// Order must follow AllTaskNames (custom removed).
	want := []string{"softwareupdate", "mas", "brew", "npm", "pipx", "rustup", "mise", "cleanup", "health"}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("names[%d] = %q, want %q", i, names[i], want[i])
		}
	}
}

func TestApplySetsEnabledFlagsAndWeeklyList(t *testing.T) {
	sel := []string{"brew", "npm", "health"}
	cfg := Default().Apply(sel, nil)

	tk := cfg.Tasks
	if !tk.Brew.Enabled || !tk.Npm.Enabled || !tk.Health.Enabled {
		t.Errorf("selected tasks should be enabled, got %+v", tk)
	}
	if tk.SoftwareUpdate.Enabled || tk.Mas.Enabled || tk.Pipx.Enabled ||
		tk.Rustup.Enabled || tk.Mise.Enabled || tk.Cleanup.Enabled {
		t.Errorf("unselected tasks should be disabled, got %+v", tk)
	}
	if tk.Custom.Enabled {
		t.Errorf("custom should be disabled with no scripts, got %+v", tk.Custom)
	}

	// Weekly task list = selected names in AllTaskNames order; Days preserved.
	wantWeekly := []string{"brew", "npm", "health"}
	got := cfg.Profiles["weekly"].Tasks
	if len(got) != len(wantWeekly) {
		t.Fatalf("weekly tasks = %v, want %v", got, wantWeekly)
	}
	for i := range wantWeekly {
		if got[i] != wantWeekly[i] {
			t.Errorf("weekly tasks[%d] = %q, want %q", i, got[i], wantWeekly[i])
		}
	}
	if days := cfg.Profiles["weekly"].Days; len(days) != 1 || days[0] != "monday" {
		t.Errorf("weekly days = %v, want [monday]", days)
	}
}

func TestApplyCustomScripts(t *testing.T) {
	scripts := []Script{{Name: "cargo", Run: "cargo install-update -a"}}
	cfg := Default().Apply([]string{"brew"}, scripts)

	if !cfg.Tasks.Custom.Enabled {
		t.Error("custom should be enabled with ≥1 script")
	}
	if len(cfg.Tasks.Custom.Scripts) != 1 || cfg.Tasks.Custom.Scripts[0].Run != "cargo install-update -a" {
		t.Errorf("custom scripts = %+v", cfg.Tasks.Custom.Scripts)
	}
	// custom appears in the weekly list, in AllTaskNames order (after brew).
	got := cfg.Profiles["weekly"].Tasks
	want := []string{"brew", "custom"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("weekly tasks = %v, want %v", got, want)
	}

	// Empty scripts → custom disabled and absent from weekly list.
	empty := Default().Apply([]string{"brew"}, nil)
	if empty.Tasks.Custom.Enabled {
		t.Error("custom should be disabled with no scripts")
	}
	for _, n := range empty.Profiles["weekly"].Tasks {
		if n == "custom" {
			t.Errorf("custom should be absent from weekly list, got %v", empty.Profiles["weekly"].Tasks)
		}
	}
}

func TestApplyDoesNotMutateBase(t *testing.T) {
	base := Default()
	_ = base.Apply([]string{"brew"}, []Script{{Name: "x", Run: "echo x"}})

	if !base.Tasks.SoftwareUpdate.Enabled || !base.Tasks.Health.Enabled {
		t.Error("Apply mutated base task flags")
	}
	if len(base.Tasks.Custom.Scripts) != 0 {
		t.Errorf("Apply mutated base custom scripts: %+v", base.Tasks.Custom.Scripts)
	}
	if len(base.Profiles["weekly"].Tasks) != len(AllTaskNames) {
		t.Errorf("Apply mutated base weekly tasks: %v", base.Profiles["weekly"].Tasks)
	}
}

func TestApplyPreservesNonWeeklyProfilesAndDays(t *testing.T) {
	base := Default()
	base.Profiles["weekly"] = Profile{Days: []string{"sunday", "wednesday"}, Tasks: []string{"brew"}}
	base.Profiles["daily"] = Profile{Days: []string{"tuesday"}, Tasks: []string{"health"}}

	cfg := base.Apply([]string{"npm"}, nil)

	if days := cfg.Profiles["weekly"].Days; len(days) != 2 || days[0] != "sunday" || days[1] != "wednesday" {
		t.Errorf("weekly days not preserved: %v", days)
	}
	daily, ok := cfg.Profiles["daily"]
	if !ok || len(daily.Tasks) != 1 || daily.Tasks[0] != "health" {
		t.Errorf("non-weekly profile not preserved: %+v", cfg.Profiles["daily"])
	}
}

func TestApplyPreservesCustomOnlyProfiles(t *testing.T) {
	base := Default()
	base.Profiles = map[string]Profile{
		"daily": {Days: []string{"tuesday"}, Tasks: []string{"health"}},
	}

	cfg := base.Apply([]string{"brew"}, nil)

	if _, ok := cfg.Profiles["weekly"]; ok {
		t.Errorf("Apply injected a weekly profile into a custom-only config: %+v", cfg.Profiles)
	}
	daily, ok := cfg.Profiles["daily"]
	if !ok || len(daily.Days) != 1 || daily.Days[0] != "tuesday" || len(daily.Tasks) != 1 || daily.Tasks[0] != "health" {
		t.Errorf("daily profile not preserved: %+v", cfg.Profiles["daily"])
	}
}

func TestApplyPreservesDisabledCustom(t *testing.T) {
	base := Default()
	script := Script{Name: "cargo", Run: "cargo install-update -a"}
	base.Tasks.Custom = CustomConfig{Enabled: false, Scripts: []Script{script}}

	// No-op save: re-apply the same single script.
	cfg := base.Apply([]string{"brew"}, []Script{script})

	if cfg.Tasks.Custom.Enabled {
		t.Error("custom should stay disabled on a no-op save")
	}
	for _, n := range cfg.Profiles["weekly"].Tasks {
		if n == "custom" {
			t.Errorf("disabled custom should be absent from weekly list, got %v", cfg.Profiles["weekly"].Tasks)
		}
	}
}

func TestApplyEnablesCustomOnAddedScript(t *testing.T) {
	base := Default()
	base.Tasks.Custom = CustomConfig{Enabled: false, Scripts: nil}

	cfg := base.Apply([]string{"brew"}, []Script{{Name: "cargo", Run: "cargo install-update -a"}})

	if !cfg.Tasks.Custom.Enabled {
		t.Error("adding a script should enable custom (intent to enable)")
	}
}

func TestEnabledTaskNames(t *testing.T) {
	cfg := Default().Apply([]string{"brew", "npm"}, nil)
	got := cfg.EnabledTaskNames()
	want := []string{"brew", "npm"}
	if len(got) != len(want) {
		t.Fatalf("EnabledTaskNames = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("EnabledTaskNames[%d] = %q, want %q", i, got[i], want[i])
		}
	}
	// custom is never reported (excluded from toggleables) even when enabled.
	withCustom := Default().Apply([]string{"brew"}, []Script{{Name: "x", Run: "echo x"}})
	for _, n := range withCustom.EnabledTaskNames() {
		if n == "custom" {
			t.Errorf("EnabledTaskNames should exclude custom, got %v", withCustom.EnabledTaskNames())
		}
	}
}

func TestApplyRoundTrip(t *testing.T) {
	sel := []string{"brew", "mise", "health"}
	scripts := []Script{{Name: "cargo", Run: "cargo install-update -a"}}
	cfg := Default().Apply(sel, scripts)

	data, err := cfg.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "rt.yaml")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	loaded, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}

	if !loaded.Tasks.Brew.Enabled || !loaded.Tasks.Mise.Enabled || !loaded.Tasks.Health.Enabled {
		t.Errorf("round-trip lost enabled flags: %+v", loaded.Tasks)
	}
	if loaded.Tasks.SoftwareUpdate.Enabled || loaded.Tasks.Npm.Enabled {
		t.Errorf("round-trip lost disabled flags: %+v", loaded.Tasks)
	}
	if !loaded.Tasks.Custom.Enabled || len(loaded.Tasks.Custom.Scripts) != 1 ||
		loaded.Tasks.Custom.Scripts[0].Run != "cargo install-update -a" {
		t.Errorf("round-trip lost custom scripts: %+v", loaded.Tasks.Custom)
	}
	wantWeekly := []string{"brew", "mise", "custom", "health"}
	got := loaded.Profiles["weekly"].Tasks
	if len(got) != len(wantWeekly) {
		t.Fatalf("round-trip weekly tasks = %v, want %v", got, wantWeekly)
	}
	for i := range wantWeekly {
		if got[i] != wantWeekly[i] {
			t.Errorf("round-trip weekly tasks[%d] = %q, want %q", i, got[i], wantWeekly[i])
		}
	}
}
