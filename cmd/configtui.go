package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/JadoJodo/rundown/internal/config"
)

// launchConfigTUI runs the interactive task-configuration TUI for both the
// create flow (base = config.Default()) and the edit flow (base = the loaded
// config). It pre-selects toggleable tasks from base.EnabledTaskNames() and
// pre-seeds custom scripts from base, lets the user pick tasks and add scripts,
// then writes the resulting config to path. It never chains into a maintenance
// run. A user abort is handled cleanly (no error, nothing written).
func launchConfigTUI(cmd *cobra.Command, path string, base config.Config) error {
	in := cmd.InOrStdin()
	out := cmd.OutOrStdout()

	selected := base.EnabledTaskNames()

	scripts := make([]config.Script, len(base.Tasks.Custom.Scripts))
	copy(scripts, base.Tasks.Custom.Scripts)

	// Phase A — task multiselect, pre-checked to the current/default state.
	enabled := make(map[string]bool, len(selected))
	for _, name := range selected {
		enabled[name] = true
	}
	opts := make([]huh.Option[string], 0, len(config.ToggleableTaskNames()))
	for _, name := range config.ToggleableTaskNames() {
		opts = append(opts, huh.NewOption(name, name).Selected(enabled[name]))
	}
	taskForm := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select the maintenance tasks to enable").
				Options(opts...).
				Value(&selected),
		),
	).WithInput(in).WithOutput(out)
	if err := taskForm.Run(); err != nil {
		return tuiAbortOrErr(out, err)
	}

	// Phase B — custom script loop. huh forms are static, so we loop a form per
	// script: confirm, then collect name + command.
	for {
		add := false
		prompt := "Add a custom script?"
		if len(scripts) > 0 {
			prompt = fmt.Sprintf("Add another custom script? (%d so far)", len(scripts))
		}
		confirmForm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().Title(prompt).Value(&add),
			),
		).WithInput(in).WithOutput(out)
		if err := confirmForm.Run(); err != nil {
			return tuiAbortOrErr(out, err)
		}
		if !add {
			break
		}

		var name, run string
		scriptForm := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("Script name").
					Value(&name),
				huh.NewInput().
					Title("Shell command").
					Value(&run).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return errors.New("command cannot be empty")
						}
						return nil
					}),
			),
		).WithInput(in).WithOutput(out)
		if err := scriptForm.Run(); err != nil {
			return tuiAbortOrErr(out, err)
		}
		scripts = append(scripts, config.Script{Name: strings.TrimSpace(name), Run: strings.TrimSpace(run)})
	}

	// Phase C — save confirmation.
	save := true
	saveForm := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Write configuration to %s?", path)).
				Value(&save),
		),
	).WithInput(in).WithOutput(out)
	if err := saveForm.Run(); err != nil {
		return tuiAbortOrErr(out, err)
	}
	if !save {
		fmt.Fprintln(out, "No changes written.")
		return nil
	}

	cfg := base.Apply(selected, scripts)
	data, err := cfg.Marshal()
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	fmt.Fprintf(out, "Wrote %s — run `rundown run` to perform maintenance.\n", path)
	return nil
}

// tuiAbortOrErr treats a user abort (Esc/Ctrl-C) as a clean cancellation:
// it prints a note and returns nil. Any other error is returned as-is.
func tuiAbortOrErr(out io.Writer, err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Fprintln(out, "No changes written.")
		return nil
	}
	return err
}
