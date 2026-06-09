package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
	"github.com/JadoJodo/monday/internal/launchd"
	"github.com/JadoJodo/monday/internal/schedule"
)

func newInstallCmd(gf *globalFlags) *cobra.Command {
	var (
		hour    int
		minute  int
		dryRun  bool
		logPath string
	)
	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install a launchd agent to run monday automatically",
		Long: "Generates a LaunchAgent that runs monday on the weekday configured " +
			"in your config file. Use --dry-run to preview the plist without " +
			"installing it.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Require a config before generating an agent: the plist runs
			// `monday run --force`, which refuses to execute without one. This
			// also blocks --dry-run, since previewing an always-failing agent
			// is exactly the bug we are guarding against.
			path, err := resolvePath(gf)
			if err != nil {
				return err
			}
			configured, err := config.Exists(path)
			if err != nil {
				return err
			}
			if !configured {
				return fmt.Errorf("no configuration found at %s; run `monday config init`", path)
			}

			cfg, err := loadConfig(gf)
			if err != nil {
				return err
			}
			weekday, err := schedule.ParseWeekday(cfg.Schedule.Day)
			if err != nil {
				return err
			}
			bin, err := os.Executable()
			if err != nil {
				return err
			}
			if logPath == "" {
				logPath = fmt.Sprintf("%s/Library/Logs/monday.log", os.Getenv("HOME"))
			}

			args := []string{"run", "--force"}
			if gf.configPath != "" {
				args = append(args, "--config", gf.configPath)
			}

			plist, err := launchd.Plist(launchd.PlistConfig{
				Program:    bin,
				Args:       args,
				Weekday:    weekday,
				Hour:       hour,
				Minute:     minute,
				StdoutPath: logPath,
				StderrPath: logPath,
			})
			if err != nil {
				return err
			}
			path, err = launchd.AgentPath(launchd.Label)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			if dryRun {
				fmt.Fprintf(out, "# would write %s\n\n%s\n", path, plist)
				fmt.Fprintf(out, "# then: launchctl load -w %s\n", path)
				return nil
			}

			if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
				return err
			}
			// Reload to pick up changes if it already exists.
			sys := exec.System{}
			_, _ = sys.Run(cmd.Context(), "launchctl", "unload", path)
			if _, err := sys.Run(cmd.Context(), "launchctl", "load", "-w", path); err != nil {
				return fmt.Errorf("wrote %s but launchctl load failed: %w", path, err)
			}
			fmt.Fprintf(out, "installed launchd agent: %s (runs %s at %02d:%02d)\n",
				path, weekday, hour, minute)
			return nil
		},
	}
	cmd.Flags().IntVar(&hour, "hour", 9, "hour of day to run (0-23)")
	cmd.Flags().IntVar(&minute, "minute", 0, "minute of hour to run (0-59)")
	cmd.Flags().StringVar(&logPath, "log", "", "log file path (default ~/Library/Logs/monday.log)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "print the plist without installing")
	return cmd
}

func newUninstallCmd(_ *globalFlags) *cobra.Command {
	return &cobra.Command{
		Use:           "uninstall",
		Short:         "Remove the launchd agent",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := launchd.AgentPath(launchd.Label)
			if err != nil {
				return err
			}
			sys := exec.System{}
			_, _ = sys.Run(cmd.Context(), "launchctl", "unload", path)
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "removed launchd agent: %s\n", path)
			return nil
		},
	}
}
