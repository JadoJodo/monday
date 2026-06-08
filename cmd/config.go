package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/monday/internal/config"
)

func newConfigCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the monday configuration file",
	}
	cmd.AddCommand(newConfigInitCmd(gf), newConfigPathCmd(gf), newConfigShowCmd(gf))
	return cmd
}

// resolvePath returns the active config path (flag override or default).
func resolvePath(gf *globalFlags) (string, error) {
	if gf.configPath != "" {
		return gf.configPath, nil
	}
	return config.DefaultPath()
}

func newConfigInitCmd(gf *globalFlags) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Write a sample config to ~/.monday.yaml",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolvePath(gf)
			if err != nil {
				return err
			}
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("%s already exists (use --force to overwrite)", path)
			} else if err != nil && !errors.Is(err, fs.ErrNotExist) {
				return err
			}
			if err := os.WriteFile(path, config.Sample(), 0o644); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config file")
	return cmd
}

func newConfigPathCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "path",
		Short:         "Print the config file path",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := resolvePath(gf)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), path)
			return nil
		},
	}
}

func newConfigShowCmd(gf *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "show",
		Short:         "Print the effective configuration",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := loadConfig(gf)
			if err != nil {
				return err
			}
			data, err := cfg.Marshal()
			if err != nil {
				return err
			}
			fmt.Fprint(cmd.OutOrStdout(), string(data))
			return nil
		},
	}
}
