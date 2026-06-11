package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// run executes the root command with args and returns combined output.
func run(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := NewRootCmd()
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	// Empty stdin so onboarding never blocks on a prompt if a test runs from a
	// real terminal; tests exercise the non-interactive branches.
	root.SetIn(strings.NewReader(""))
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestVersionCommand(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "rundown") {
		t.Errorf("version output = %q", out)
	}
}

func TestListCommand(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")
	out, err := run(t, "--config", cfg, "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"softwareupdate", "mas", "brew", "npm", "pipx", "rustup", "mise", "custom", "cleanup", "health"} {
		if !strings.Contains(out, name) {
			t.Errorf("list missing task %q in %q", name, out)
		}
	}
}

func TestConfigInitAndShow(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")

	out, err := run(t, "--config", cfg, "config", "init")
	if err != nil {
		t.Fatalf("config init: %v", err)
	}
	if !strings.Contains(out, "wrote") {
		t.Errorf("init output = %q", out)
	}
	if _, err := os.Stat(cfg); err != nil {
		t.Fatalf("config file not written: %v", err)
	}

	// Second init without --force must fail.
	if _, err := run(t, "--config", cfg, "config", "init"); err == nil {
		t.Error("re-init without --force should fail")
	}

	// --force overwrites.
	if _, err := run(t, "--config", cfg, "config", "init", "--force"); err != nil {
		t.Errorf("init --force should succeed: %v", err)
	}

	show, err := run(t, "--config", cfg, "config", "show")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(show, "profiles") || !strings.Contains(show, "weekly") {
		t.Errorf("show output = %q", show)
	}
}

func TestConfigPath(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "custom.yaml")
	out, err := run(t, "--config", cfg, "config", "path")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "custom.yaml") {
		t.Errorf("path output = %q", out)
	}
}

// TestRunUnconfiguredRefuses verifies the key safety fix: `rundown run` with no
// config (non-interactive, as under launchd) refuses with a non-zero exit and
// writes nothing — so no maintenance can run on an unconfigured machine.
func TestRunUnconfiguredRefuses(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")
	out, err := run(t, "--config", cfg, "run", "--force")
	if err == nil {
		t.Fatal("run with no config should return an error")
	}
	if !strings.Contains(err.Error(), "no configuration found") ||
		!strings.Contains(err.Error(), "config init") {
		t.Errorf("error = %q, want guidance toward `config init`", err)
	}
	if _, statErr := os.Stat(cfg); !os.IsNotExist(statErr) {
		t.Errorf("run must not create a config file, but %s exists", cfg)
	}
	_ = out
}

// TestDefaultUnconfiguredShowsModules verifies bare `rundown` with no config is
// informational: it lists modules, points at `config init`, exits 0, and never
// runs or writes anything.
func TestDefaultUnconfiguredShowsModules(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")
	out, err := run(t, "--config", cfg)
	if err != nil {
		t.Fatalf("bare rundown with no config should exit 0: %v", err)
	}
	for _, name := range []string{"softwareupdate", "mas", "brew", "npm", "pipx", "rustup", "mise", "custom", "cleanup", "health"} {
		if !strings.Contains(out, name) {
			t.Errorf("output missing module %q in %q", name, out)
		}
	}
	if !strings.Contains(out, "config init") {
		t.Errorf("output should hint at `config init`: %q", out)
	}
	if _, statErr := os.Stat(cfg); !os.IsNotExist(statErr) {
		t.Errorf("bare rundown must not create a config file, but %s exists", cfg)
	}
}

// TestDefaultConfiguredShowsStatus verifies bare `rundown` with a config shows
// status and a hint to run maintenance, and never executes tasks.
func TestDefaultConfiguredShowsStatus(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")
	if _, err := run(t, "--config", cfg, "config", "init"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfg)
	if err != nil {
		t.Fatalf("bare rundown with config: %v", err)
	}
	if !strings.Contains(out, "softwareupdate") {
		t.Errorf("status output missing modules: %q", out)
	}
	if !strings.Contains(out, "rundown run") {
		t.Errorf("status output should hint to run maintenance: %q", out)
	}
}

// install must reject a config that exists but does not parse under the current
// schema (a legacy `schedule:` file or malformed YAML) — otherwise it writes a
// LaunchAgent whose `rundown run` fails on every fire. The check covers --dry-run
// too, so previewing an always-failing agent is impossible.
func TestInstallRejectsUnparseableConfig(t *testing.T) {
	cases := map[string]string{
		"legacy schedule schema": "schedule:\n  days: [monday]\n",
		"malformed yaml":         "profiles: [this is: not valid\n",
	}
	for name, contents := range cases {
		t.Run(name, func(t *testing.T) {
			cfg := filepath.Join(t.TempDir(), "rundown.yaml")
			if err := os.WriteFile(cfg, []byte(contents), 0o644); err != nil {
				t.Fatal(err)
			}
			for _, extra := range [][]string{{}, {"--dry-run"}} {
				args := append([]string{"--config", cfg, "install"}, extra...)
				out, err := run(t, args...)
				if err == nil {
					t.Fatalf("install %v: expected error for %s, got output %q", extra, name, out)
				}
			}
		})
	}
}

func TestInstallDryRun(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml")
	if _, err := run(t, "--config", cfg, "config", "init"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfg, "install", "--dry-run", "--hour", "7", "--minute", "15")
	if err != nil {
		t.Fatalf("install --dry-run: %v", err)
	}
	if !strings.Contains(out, "io.rundown.agent") || !strings.Contains(out, "launchctl load") {
		t.Errorf("install dry-run output = %q", out)
	}
	if !strings.Contains(out, "<integer>7</integer>") {
		t.Errorf("expected hour 7 in plist: %q", out)
	}
}

// install must refuse without a config — otherwise it would write a LaunchAgent
// whose `rundown run --force` always fails the missing-config guard. The check
// also covers --dry-run, so previewing an always-failing agent is impossible.
func TestInstallRequiresConfig(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "rundown.yaml") // never created
	for _, extra := range [][]string{{}, {"--dry-run"}} {
		args := append([]string{"--config", cfg, "install"}, extra...)
		out, err := run(t, args...)
		if err == nil {
			t.Fatalf("install %v: expected error, got output %q", extra, out)
		}
		if !strings.Contains(err.Error(), "no configuration found") {
			t.Errorf("install %v error = %v", extra, err)
		}
	}
}
