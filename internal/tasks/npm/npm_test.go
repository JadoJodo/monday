package npm

import (
	"context"
	"errors"
	"testing"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/task"
)

func TestNpmMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "npm" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestNpmEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("npm should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Npm.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("npm should be disabled")
	}
}

func TestNpmDryRunUsesOutdated(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("npm", exec.Output{Stdout: "left-pad  1.0.0  2.0.0"}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("dry run must not report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Name != "npm" || last.Args[0] != "-g" || last.Args[1] != "outdated" {
		t.Errorf("expected `npm -g outdated`, got %+v", last)
	}
}

func TestNpmApplyUsesUpdate(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("npm", exec.Output{Stdout: "updated"}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: false, Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if !res.Changed {
		t.Error("apply should report Changed")
	}
	last := fake.Calls[len(fake.Calls)-1]
	if last.Args[1] != "update" {
		t.Errorf("expected update, got %+v", last.Args)
	}
}

func TestNpmToleratesExitOne(t *testing.T) {
	fake := exec.NewFake()
	// `npm outdated` exits 1 when packages are stale.
	fake.AddResult("npm", exec.Output{Stdout: "stale", ExitCode: 1}, errors.New("exit status 1"))

	res := New().Run(context.Background(), config.Default(), task.Options{DryRun: true, Commander: fake})
	if res.Err != nil {
		t.Errorf("exit 1 should be tolerated, got %v", res.Err)
	}
}

func TestNpmMissingBinarySkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["npm"] = true

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Error("missing npm should skip")
	}
	if len(fake.Calls) != 0 {
		t.Error("missing binary should not run any command")
	}
}
