package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/config"
)

func newConfigCmd(gf *globalFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage the rundown configuration file",
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

// legacyConfigPath reports the path to a config left over from the tool's
// previous name (monday) sitting beside the active path, and whether such a
// file exists. It lets the missing-config error point users at a file to rename
// instead of telling them to start from scratch.
func legacyConfigPath(path string) (string, bool) {
	dir := filepath.Dir(path)
	legacy := filepath.Join(dir, ".monday.yaml")
	if legacy == path {
		return "", false
	}
	if _, err := os.Stat(legacy); err != nil {
		return "", false
	}
	return legacy, true
}

func newConfigInitCmd(gf *globalFlags) *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:           "init",
		Short:         "Write a sample config to ~/.rundown.yaml",
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
