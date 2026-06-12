package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/registry"
	"github.com/JadoJodo/rundown/internal/ui"
)

// doDefault backs the bare `rundown` command. It is purely informational and
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
	fmt.Fprintln(out, "\nRun `rundown run` to perform maintenance.")
	return nil
}

// onboard runs the first-run flow for an unconfigured machine. On an
// interactive terminal it launches the task-configuration TUI (create flow,
// pre-selected from config.Default()); on a non-interactive one it prints
// guidance toward `rundown config init`. It NEVER executes maintenance and
// never chains into a run — creating config and running maintenance stay
// cleanly separated.
func onboard(cmd *cobra.Command, path string) error {
	if !interactive() {
		fmt.Fprintf(cmd.OutOrStdout(), "No configuration found at %s.\nRun `rundown config init` to create one, then `rundown run` to perform maintenance.\n", path)
		return nil
	}
	return launchConfigTUI(cmd, path, config.Default())
}
