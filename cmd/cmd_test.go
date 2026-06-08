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
	root.SetArgs(args)
	err := root.Execute()
	return buf.String(), err
}

func TestVersionCommand(t *testing.T) {
	out, err := run(t, "version")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "monday") {
		t.Errorf("version output = %q", out)
	}
}

func TestListCommand(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "monday.yaml")
	out, err := run(t, "--config", cfg, "list")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"softwareupdate", "mas", "npm", "custom"} {
		if !strings.Contains(out, name) {
			t.Errorf("list missing task %q in %q", name, out)
		}
	}
}

func TestConfigInitAndShow(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "monday.yaml")

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
	if !strings.Contains(show, "schedule") || !strings.Contains(show, "monday") {
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

func TestInstallDryRun(t *testing.T) {
	cfg := filepath.Join(t.TempDir(), "monday.yaml")
	if _, err := run(t, "--config", cfg, "config", "init"); err != nil {
		t.Fatal(err)
	}
	out, err := run(t, "--config", cfg, "install", "--dry-run", "--hour", "7", "--minute", "15")
	if err != nil {
		t.Fatalf("install --dry-run: %v", err)
	}
	if !strings.Contains(out, "io.monday.agent") || !strings.Contains(out, "launchctl load") {
		t.Errorf("install dry-run output = %q", out)
	}
	if !strings.Contains(out, "<integer>7</integer>") {
		t.Errorf("expected hour 7 in plist: %q", out)
	}
}
