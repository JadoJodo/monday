package cleanup

import (
	"context"
	"strings"
	"testing"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/task"
)

func TestCleanupMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "cleanup" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestCleanupEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("cleanup should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Cleanup.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("cleanup should be disabled")
	}
}

func TestCleanupAggregatesFigures(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("brew", exec.Output{Stdout: `Would remove: /opt/homebrew/Cellar/old (10 files, 4.2MB)
==> This operation would free approximately 503.9MB of disk space.`}, nil)
	fake.AddResult("docker", exec.Output{Stdout: `TYPE            TOTAL     ACTIVE    SIZE      RECLAIMABLE
Images          5         2         2.0GB     1.5GB (75%)
Build Cache     10        0         300MB     300MB`}, nil)
	// du is called for DerivedData then npm cache, in that order.
	fake.AddResult("du", exec.Output{Stdout: "5000000\t/Users/x/Library/Developer/Xcode/DerivedData"}, nil)
	fake.AddResult("du", exec.Output{Stdout: "800000\t/Users/x/.npm"}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("cleanup must never fail the task: %v", res.Err)
	}
	if res.Changed {
		t.Error("cleanup is report-only and must never report Changed")
	}
	if res.Skipped {
		t.Error("with figures present, cleanup should not be skipped")
	}
	for _, want := range []string{"reclaimable:", "brew 503.9MB", "docker", "DerivedData 5.1GB", "npm 819.2MB"} {
		if !strings.Contains(res.Summary, want) {
			t.Errorf("summary %q missing %q", res.Summary, want)
		}
	}

	// The du calls must target the right paths (asserted by suffix).
	var duArgs [][]string
	for _, c := range fake.Calls {
		if c.Name == "du" {
			duArgs = append(duArgs, c.Args)
		}
	}
	if len(duArgs) != 2 {
		t.Fatalf("expected 2 du calls, got %d", len(duArgs))
	}
	if !strings.HasSuffix(duArgs[0][len(duArgs[0])-1], "DerivedData") {
		t.Errorf("first du should target DerivedData, got %v", duArgs[0])
	}
	if !strings.HasSuffix(duArgs[1][len(duArgs[1])-1], ".npm") {
		t.Errorf("second du should target .npm, got %v", duArgs[1])
	}
}

func TestCleanupDockerDaemonDown(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["brew"] = true
	// docker present but the daemon is down (non-nil error).
	fake.AddResult("docker", exec.Output{Stderr: "Cannot connect to the Docker daemon"}, errContext())
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("daemon-down must not fail the task: %v", res.Err)
	}
	joined := strings.Join(res.Details, "\n")
	if !strings.Contains(joined, "daemon not running") {
		t.Errorf("expected a daemon-not-running note in details: %q", joined)
	}
}

func TestCleanupNothingAvailableSkips(t *testing.T) {
	fake := exec.NewFake()
	fake.MissingPaths["brew"] = true
	fake.MissingPaths["docker"] = true
	// no du results → duKB returns ok=false for both paths.
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Errorf("with no figures, cleanup should be Skipped; summary=%q", res.Summary)
	}
}

// errContext returns a non-nil error to simulate a failed command.
func errContext() error { return context.Canceled }
