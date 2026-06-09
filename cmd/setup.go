package cmd

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/ui"
)

var promptStyle = lipgloss.NewStyle().Bold(true)

// doDefault backs the bare `monday` command. It is purely informational and
// NEVER executes tasks: when unconfigured it runs first-run onboarding, and
// when configured it prints the module status plus a hint to run maintenance.
func doDefault(cmd *cobra.Command, gf *globalFlags) error {
	path, err := resolvePath(gf)
	if err != nil {
		return err
	}
	ok, err := config.Exists(path)
	if err != nil {
		return err
	}
	if !ok {
		return onboard(cmd, path)
	}

	cfg, err := loadConfig(gf)
	if err != nil {
		return err
	}
	out := cmd.OutOrStdout()
	fmt.Fprintln(out, ui.List(registry.Default(), cfg))
	fmt.Fprintf(out, "\nScheduled for %s. Run `monday run` to perform maintenance.\n",
		cfg.Schedule.Day)
	return nil
}

// showModules lists the available maintenance modules and their default
// enabled state, reusing the same renderer as `monday list`.
func showModules(out io.Writer) {
	fmt.Fprintln(out, ui.List(registry.Default(), config.Default()))
}

// confirmCreate prompts the user to write a starter config at path and reports
// their answer. An empty response defaults to yes ([Y/n]). It must only be
// called on an interactive terminal.
func confirmCreate(cmd *cobra.Command, path string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s ",
		promptStyle.Render(fmt.Sprintf("Create a config at %s now? [Y/n]", path)))

	reader := bufio.NewReader(cmd.InOrStdin())
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return false, err
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "", "y", "yes":
		return true, nil
	default:
		return false, nil
	}
}

// onboard runs the first-run flow for an unconfigured machine: it always lists
// the available modules, and on an interactive terminal offers to write a
// starter config. It NEVER executes maintenance and never chains into a run —
// creating config and running maintenance stay cleanly separated.
func onboard(cmd *cobra.Command, path string) error {
	out := cmd.OutOrStdout()
	showModules(out)

	if !interactive() {
		fmt.Fprintf(out, "\nNo configuration found at %s.\nRun `monday config init` to create one, then `monday run` to perform maintenance.\n", path)
		return nil
	}

	create, err := confirmCreate(cmd, path)
	if err != nil {
		return err
	}
	if !create {
		fmt.Fprintln(out, "\nNo configuration written. Run `monday config init` later to create one.")
		return nil
	}
	if err := os.WriteFile(path, config.Sample(), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "\nWrote %s\nReview it, then run `monday run` to perform maintenance.\n", path)
	return nil
}
