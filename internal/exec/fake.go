package exec

import (
	"context"
	"fmt"
)

// Call records a single invocation captured by Fake.
type Call struct {
	Name string
	Args []string
}

// FakeResult is the canned response for a matched command.
type FakeResult struct {
	Output Output
	Err    error
}

// Fake is a test Commander. It records calls and returns canned results keyed
// by the executable name. Unmatched commands return an empty success unless
// DefaultErr is set. Missing executables can be simulated via MissingPaths.
type Fake struct {
	// Results maps an executable name to the responses returned on successive
	// calls. The last entry is reused once the slice is exhausted.
	Results map[string][]FakeResult
	// MissingPaths lists executable names that LookPath should report as absent.
	MissingPaths map[string]bool
	// DefaultErr, when set, is returned for any name without a Results entry.
	DefaultErr error

	Calls   []Call
	callIdx map[string]int
}

// NewFake returns an empty Fake ready for use.
func NewFake() *Fake {
	return &Fake{
		Results:      map[string][]FakeResult{},
		MissingPaths: map[string]bool{},
		callIdx:      map[string]int{},
	}
}

// AddResult appends a canned response for the given executable name.
func (f *Fake) AddResult(name string, out Output, err error) *Fake {
	if f.Results == nil {
		f.Results = map[string][]FakeResult{}
	}
	f.Results[name] = append(f.Results[name], FakeResult{Output: out, Err: err})
	return f
}

// Run records the call and returns the next canned result for name.
func (f *Fake) Run(_ context.Context, name string, args ...string) (Output, error) {
	f.Calls = append(f.Calls, Call{Name: name, Args: args})
	if f.callIdx == nil {
		f.callIdx = map[string]int{}
	}

	results := f.Results[name]
	if len(results) == 0 {
		return Output{}, f.DefaultErr
	}
	idx := f.callIdx[name]
	if idx >= len(results) {
		idx = len(results) - 1
	}
	f.callIdx[name]++
	r := results[idx]
	return r.Output, r.Err
}

// LookPath reports the executable as found unless listed in MissingPaths.
func (f *Fake) LookPath(name string) (string, error) {
	if f.MissingPaths[name] {
		return "", fmt.Errorf("exec: %q not found in fake PATH", name)
	}
	return "/usr/bin/" + name, nil
}
