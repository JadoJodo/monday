package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/registry"
	"github.com/JadoJodo/rundown/internal/ui"
)

func newListCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "list",
		Short:         "List available tasks and their enabled state",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(gf)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), ui.List(registry.Default(), cfg))
			return nil
		},
	}
}
