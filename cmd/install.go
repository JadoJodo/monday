package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/config"
	"github.com/JadoJodo/rundown/internal/exec"
	"github.com/JadoJodo/rundown/internal/launchd"
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
		Short: "Install a launchd agent to run rundown automatically",
		Long: "Generates a LaunchAgent that runs rundown daily; rundown itself " +
			"decides which profiles are due each day. Use --dry-run to preview " +
			"the plist without installing it.",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Require a config before generating an agent: the plist runs
			// `rundown run`, which refuses to execute without one. This also
			// blocks --dry-run, since previewing an always-failing agent is
			// exactly the bug we are guarding against.
			path, err := resolvePath(gf)
			if err != nil {
				return err
			}
			configured, err := config.Exists(path)
			if err != nil {
				return err
			}
			if !configured {
				return fmt.Errorf("no configuration found at %s; run `rundown config init`", path)
			}
			// Validate the config parses under the current schema before generating
			// an agent: the plist runs `rundown run`, which would fail every fire on a
			// legacy or malformed config otherwise.
			if _, err := config.Load(path); err != nil {
				return err
			}

			bin, err := os.Executable()
			if err != nil {
				return err
			}
			if logPath == "" {
				logPath = fmt.Sprintf("%s/Library/Logs/rundown.log", os.Getenv("HOME"))
			}

			// The agent runs daily and lets rundown's schedule decide which
			// profiles are due, so it must NOT pass --force.
			args := []string{"run"}
			if gf.configPath != "" {
				args = append(args, "--config", gf.configPath)
			}

			plist, err := launchd.Plist(launchd.PlistConfig{
				Program:    bin,
				Args:       args,
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

			sys := exec.System{}

			// Clean up an agent left by the tool's previous name (monday). It
			// points at a binary the cask rename removed, so it would keep firing
			// and failing. Bootout and delete it before installing the new one.
			if legacyPath, err := launchd.AgentPath(launchd.LegacyLabel); err == nil {
				if _, statErr := os.Stat(legacyPath); statErr == nil {
					_, _ = sys.Run(cmd.Context(), "launchctl", "unload", legacyPath)
					if rmErr := os.Remove(legacyPath); rmErr != nil && !os.IsNotExist(rmErr) {
						return fmt.Errorf("removing legacy agent %s: %w", legacyPath, rmErr)
					}
					fmt.Fprintf(out, "removed legacy launchd agent: %s\n", legacyPath)
				}
			}

			if err := os.WriteFile(path, []byte(plist), 0o644); err != nil {
				return err
			}
			// Reload to pick up changes if it already exists.
			_, _ = sys.Run(cmd.Context(), "launchctl", "unload", path)
			if _, err := sys.Run(cmd.Context(), "launchctl", "load", "-w", path); err != nil {
				return fmt.Errorf("wrote %s but launchctl load failed: %w", path, err)
			}
			fmt.Fprintf(out, "installed launchd agent: %s (runs daily at %02d:%02d)\n",
				path, hour, minute)
			return nil
		},
	}
	cmd.Flags().IntVar(&hour, "hour", 9, "hour of day to run (0-23)")
	cmd.Flags().IntVar(&minute, "minute", 0, "minute of hour to run (0-59)")
	cmd.Flags().StringVar(&logPath, "log", "", "log file path (default ~/Library/Logs/rundown.log)")
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
