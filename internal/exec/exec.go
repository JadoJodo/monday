// Package exec provides a small abstraction over os/exec so that maintenance
// tasks can be unit-tested without shelling out to the real system. Tasks
// depend on the Commander interface; production code uses System while tests
// inject a fake.
package exec

import (
	"bytes"
	"context"
	"errors"
	"os/exec"
)

// Output is the captured result of running a command.
type Output struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Commander runs external commands. Implementations must be safe to reuse.
type Commander interface {
	// Run executes name with args and returns the captured Output. A non-zero
	// exit status is reported via Output.ExitCode and a non-nil error; callers
	// that tolerate failures can inspect ExitCode instead of the error.
	Run(ctx context.Context, name string, args ...string) (Output, error)
	// LookPath reports whether the named executable can be found on PATH.
	LookPath(name string) (string, error)
}

// System is the production Commander backed by os/exec.
type System struct{}

// Run executes the command, capturing stdout and stderr.
func (System) Run(ctx context.Context, name string, args ...string) (Output, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()

	out := Output{Stdout: stdout.String(), Stderr: stderr.String()}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		out.ExitCode = exitErr.ExitCode()
	} else if err != nil {
		out.ExitCode = -1
	}
	return out, err
}

// LookPath resolves an executable on PATH.
func (System) LookPath(name string) (string, error) {
	return exec.LookPath(name)
}
