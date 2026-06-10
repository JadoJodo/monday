package notify

import (
	"context"

	"github.com/JadoJodo/monday/internal/config"
	"github.com/JadoJodo/monday/internal/exec"
)

// macOSScript is run via `osascript -e <script> <title> <body>`. Title and body
// arrive as argv items rather than being interpolated into the AppleScript
// source, so arbitrary task output cannot break out of the string or inject
// AppleScript.
const macOSScript = `on run argv
	display notification (item 2 of argv) with title (item 1 of argv)
end run`

type macOS struct{ cmd exec.Commander }

// MacOS returns the native macOS notification notifier backed by cmd.
func MacOS(cmd exec.Commander) Notifier { return macOS{cmd: cmd} }

func (macOS) Name() string                   { return "macos" }
func (macOS) Enabled(cfg config.Config) bool { return cfg.Notify.MacOS.Enabled }

func (m macOS) Send(ctx context.Context, _ config.Config, msg Message) error {
	_, err := m.cmd.Run(ctx, "osascript", "-e", macOSScript, msg.Title, msg.Body)
	return err
}
