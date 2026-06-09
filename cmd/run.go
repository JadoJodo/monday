package cmd

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/registry"
	"github.com/JadoJodo/monday/internal/runner"
	"github.com/JadoJodo/monday/internal/ui"
)

// runFlags holds the options for a maintenance run.
type runFlags struct {
	dryRun bool
	force  bool
	day    string
	only   []string
}

func addRunFlags(fs *pflag.FlagSet, f *runFlags) {
	fs.BoolVar(&f.dryRun, "dry-run", false, "preview actions without making changes")
	fs.BoolVar(&f.force, "force", false, "run regardless of the configured weekday")
	fs.StringVar(&f.day, "day", "", "override the configured weekday (e.g. friday)")
	fs.StringSliceVar(&f.only, "only", nil, "run only these tasks (comma-separated)")
}

func newRunCmd(gf *globalFlags) *cobra.Command {
	rf := &runFlags{}
	cmd := &cobra.Command{
		Use:           "run",
		Short:         "Run maintenance tasks",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return doRun(cmd, gf, rf)
		},
	}
	addRunFlags(cmd.Flags(), rf)
	return cmd
}

// doRun loads config, runs the selected tasks and prints results. It returns a
// non-nil error when any task fails so the process exits non-zero.
func doRun(cmd *cobra.Command, gf *globalFlags, rf *runFlags) error {
	// Safety gate: never execute maintenance until a config file exists. On an
	// interactive terminal we offer to create one (but do NOT chain into a run);
	// otherwise — e.g. the launchd agent — we refuse loudly so nothing runs.
	path, err := resolvePath(gf)
	if err != nil {
		return err
	}
	configured, err := config.Exists(path)
	if err != nil {
		return err
	}
	if !configured {
		if interactive() {
			return onboard(cmd, path)
		}
		return fmt.Errorf("no configuration found at %s; run `monday config init`", path)
	}

	cfg, err := loadConfig(gf)
	if err != nil {
		return err
	}

	// Explicitly selecting tasks implies the user wants them run now.
	force := rf.force || len(rf.only) > 0

	sum, err := runner.Run(cmd.Context(), registry.Default(), cfg, runner.Options{
		DryRun:      rf.dryRun,
		Only:        rf.only,
		Force:       force,
		DayOverride: rf.day,
		Commander:   exec.System{},
	})
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out, ui.Decision(sum.Decision))
	if len(sum.Results) > 0 {
		fmt.Fprintln(out, ui.Results(sum.Results, gf.verbose))
	}

	if sum.Failed() {
		return errors.New("one or more tasks failed")
	}
	return nil
}

// loadConfig resolves the config path (flag or default) and loads it.
func loadConfig(gf *globalFlags) (config.Config, error) {
	path := gf.configPath
	if path == "" {
		p, err := config.DefaultPath()
		if err != nil {
			return config.Config{}, err
		}
		path = p
	}
	return config.Load(path)
}
