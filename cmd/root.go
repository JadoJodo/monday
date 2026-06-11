// Package cmd implements rundown's command-line interface using Cobra, wrapped
// by Fang for styled help, errors, version handling and completions.
package cmd

import (
	"context"
	"os"

	"github.com/charmbracelet/fang"
	"github.com/spf13/cobra"
)

// Build information, overridable at link time via -ldflags
// "-X github.com/JadoJodo/rundown/cmd.version=... -X ...commit=...".
var (
	version = "dev"
	commit  = "none"
)

// globalFlags holds options shared across all commands.
type globalFlags struct {
	configPath string
	verbose    bool
}

// NewRootCmd builds the root command tree.
func NewRootCmd() *cobra.Command {
	gf := &globalFlags{}

	root := &cobra.Command{
		Use:   "rundown",
		Short: "Automate routine macOS maintenance",
		Long: "rundown runs your weekly macOS maintenance chores — system updates, " +
			"Mac App Store updates, npm globals and custom scripts — from one command.\n\n" +
			"Running `rundown` with no subcommand shows status and the available " +
			"modules; it never executes anything. Run `rundown run` to perform " +
			"maintenance.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return doDefault(cmd, gf)
		},
	}

	root.PersistentFlags().StringVar(&gf.configPath, "config", "",
		"path to config file (default ~/.rundown.yaml)")
	root.PersistentFlags().BoolVarP(&gf.verbose, "verbose", "V", false,
		"show command output detail")

	root.AddCommand(
		newRunCmd(gf),
		newListCmd(gf),
		newConfigCmd(gf),
		newVersionCmd(),
		newMCPCmd(gf),
		newInstallCmd(gf),
		newUninstallCmd(gf),
	)
	return root
}

// Execute runs the CLI through Fang.
func Execute(ctx context.Context) error {
	return fang.Execute(
		ctx,
		NewRootCmd(),
		fang.WithVersion(version),
		fang.WithCommit(commit),
		fang.WithNotifySignal(os.Interrupt),
	)
}
