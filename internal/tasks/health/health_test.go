package health

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/task"
)

const dfFixture = `Filesystem        Size    Used   Avail Capacity iused ifree %iused  Mounted on
/dev/disk3s1s1   460Gi    12Gi    74Gi    72%    459k  774M    0%   /`

const pmsetLaptop = `Now drawing from 'Battery Power'
 -InternalBattery-0 (id=25296995)	98%; discharging; 1:40 remaining present: true`

const pmsetDesktop = `Now drawing from 'AC Power'`

const ioregFixture = `  | {
    | "CycleCount" = 412
    | "DesignCapacity" = 5088
  | }`

func TestHealthMetadata(t *testing.T) {
	tk := New()
	if tk.Name() != "health" {
		t.Errorf("name = %q", tk.Name())
	}
	if tk.Description() == "" {
		t.Error("description empty")
	}
}

func TestHealthEnabled(t *testing.T) {
	tk := New()
	if !tk.Enabled(config.Default()) {
		t.Error("health should be enabled by default")
	}
	cfg := config.Default()
	cfg.Tasks.Health.Enabled = false
	if tk.Enabled(cfg) {
		t.Error("health should be disabled")
	}
}

func TestHealthLaptopDiskAndBattery(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("df", exec.Output{Stdout: dfFixture}, nil)
	fake.AddResult("pmset", exec.Output{Stdout: pmsetLaptop}, nil)
	fake.AddResult("ioreg", exec.Output{Stdout: ioregFixture}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if res.Changed {
		t.Error("health is read-only and must never report Changed")
	}
	if !strings.Contains(res.Summary, "disk 72% used") {
		t.Errorf("summary missing disk metric: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "battery 98% (412 cycles)") {
		t.Errorf("summary missing battery metric: %q", res.Summary)
	}
}

func TestHealthDesktopOmitsBattery(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("df", exec.Output{Stdout: dfFixture}, nil)
	fake.AddResult("pmset", exec.Output{Stdout: pmsetDesktop}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if res.Err != nil {
		t.Fatalf("unexpected err: %v", res.Err)
	}
	if strings.Contains(res.Summary, "battery") {
		t.Errorf("desktop should omit battery: %q", res.Summary)
	}
	if !strings.Contains(res.Summary, "disk 72% used") {
		t.Errorf("summary missing disk metric: %q", res.Summary)
	}
	// ioreg must not be consulted when there is no internal battery.
	for _, c := range fake.Calls {
		if c.Name == "ioreg" {
			t.Error("ioreg should not run on a desktop")
		}
	}
}

func TestHealthBatteryWithoutCycles(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("df", exec.Output{Stdout: dfFixture}, nil)
	fake.AddResult("pmset", exec.Output{Stdout: pmsetLaptop}, nil)
	fake.AddResult("ioreg", exec.Output{Stdout: "no cycle field here"}, nil)

	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !strings.Contains(res.Summary, "battery 98%") || strings.Contains(res.Summary, "cycles") {
		t.Errorf("battery without cycles wrong: %q", res.Summary)
	}
}

func TestHealthNoMetricsSkips(t *testing.T) {
	fake := exec.NewFake()
	fake.AddResult("df", exec.Output{}, errors.New("df failed"))
	fake.AddResult("pmset", exec.Output{}, errors.New("pmset failed"))
	res := New().Run(context.Background(), config.Default(), task.Options{Commander: fake})
	if !res.Skipped {
		t.Errorf("no metrics should skip, got summary %q", res.Summary)
	}
}
