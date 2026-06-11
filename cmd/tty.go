package cmd

import (
	"os"

	"github.com/mattn/go-isatty"
)

// interactive reports whether rundown can safely prompt the user: both stdin and
// stdout must be connected to a terminal. Under launchd, a pipe or any
// redirection this is false, so onboarding falls back to printed guidance.
func interactive() bool {
	return isTerminal(os.Stdin.Fd()) && isTerminal(os.Stdout.Fd())
}

func isTerminal(fd uintptr) bool {
	return isatty.IsTerminal(fd) || isatty.IsCygwinTerminal(fd)
}
