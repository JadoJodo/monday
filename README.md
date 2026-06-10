# monday

Automate routine macOS maintenance from one command. `monday` runs your
chores — system software updates, Mac App Store updates, Homebrew/npm/pipx/Rust/mise
package upgrades, your own custom scripts, plus read-only disk-cleanup and
health reports — on a schedule, with a single CLI. Tasks are bundled into named
day-**profiles**, toggled via a YAML config, and the whole thing is exposed to
AI agents over MCP. Runs report their outcome via macOS notifications and/or
[ntfy](https://ntfy.sh) so headless (launchd) runs stay visible.

## Install

Via Homebrew:

```sh
brew install JadoJodo/tap/monday
```

Or build from source (Go 1.26+):

```sh
go install github.com/JadoJodo/monday@latest
```

## Quick start

```sh
monday config init      # write a sample ~/.monday.yaml (required before running)
monday                  # show profiles, tasks and their enabled state
monday run              # run any profiles due today
monday run --dry-run    # preview what would happen, changing nothing
monday run --force      # run every profile now, regardless of the day
```

By default `monday run` **applies** updates. Use `--dry-run` to preview first.

## Tasks

| Task             | Dry-run                          | Apply                                       |
| ---------------- | -------------------------------- | ------------------------------------------- |
| `softwareupdate` | `softwareupdate -l`              | `softwareupdate -ia`                        |
| `mas`            | `mas outdated`                   | `mas upgrade`                               |
| `brew`           | `brew update` + `brew outdated`  | `brew update` + `brew upgrade` + `cleanup`  |
| `npm`            | `npm -g outdated`                | `npm -g update`                             |
| `pipx`           | `pipx list --short`              | `pipx upgrade-all`                          |
| `rustup`         | `rustup check`                   | `rustup update`                             |
| `mise`           | `mise outdated`                  | `mise upgrade`                              |
| `custom`         | lists configured cmds            | runs each via `sh -c`                       |
| `cleanup`        | report-only — reclaimable disk   | _same as dry-run; never deletes_            |
| `health`         | report-only — disk %, battery    | _same as dry-run; never changes anything_   |

Tasks whose underlying tool is not installed (e.g. `mas`, `pipx`) are skipped,
not failed. `cleanup` and `health` are **report-only**: they run regardless of
`--dry-run` and never modify the system.

## Configuration

`monday config init` writes `~/.monday.yaml`. **Profiles** bundle tasks onto
weekdays; `monday` decides which profiles are due each day.

```yaml
profiles:
  weekly:
    days: [monday]
    tasks: [softwareupdate, mas, brew, npm, pipx, rustup, mise, custom, cleanup, health]
  # daily:
  #   days: [tuesday, wednesday, thursday, friday]
  #   tasks: [npm, health]

tasks:
  softwareupdate: { enabled: true }
  mas:            { enabled: true }
  brew:           { enabled: true }
  npm:            { enabled: true }
  pipx:           { enabled: true }
  rustup:         { enabled: true }
  mise:           { enabled: true }
  custom:
    enabled: true
    scripts: []
  cleanup:        { enabled: true }   # report-only; never deletes
  health:         { enabled: true }

notify:
  on_success: false        # failures always notify; set true to also notify on clean runs
  macos: { enabled: true }
  ntfy:
    enabled: false
    server: https://ntfy.sh
    topic: my-monday
    priority: default      # min|low|default|high|urgent (bumped to high on failure)
```

`monday` needs a config file before it will run maintenance — create one with
`monday config init`. Every task is enabled by default; a task is only disabled
by an explicit `enabled: false`. A user-defined `profiles:` block fully replaces
the default `weekly` profile.

> **Upgrading from the old `schedule:` schema?** There is no automatic
> migration. `monday` rejects an old-schema config with a clear error; run
> `monday config init` (your file is preserved until you overwrite it).

Useful flags:

- `--config <path>` — use a non-default config file
- `--dry-run` — preview without changing anything
- `--only npm,custom` — run just the named tasks, bypassing profiles (implies `--force`)
- `--profile weekly` — run named profiles regardless of the day (repeatable)
- `--day friday` — pretend today is this weekday
- `--force` — run every profile now
- `-V`, `--verbose` — show command output detail

## Notifications

After a run, `monday` reports the outcome so launchd-triggered runs are visible:

- **macOS notification** (on by default) — a native banner with the run summary.
- **ntfy** (off by default) — a POST to `{server}/{topic}`; works with the public
  `ntfy.sh` or a self-hosted server, and on your phone via the ntfy app.

Failures always notify; clean runs notify only when `notify.on_success: true`.
On failure an unset/default ntfy priority is bumped to `high`. Dry-runs and
not-due runs never notify.

## Automatic scheduling (launchd)

Install a per-user LaunchAgent that runs `monday run` **daily** (run
`monday config init` first — `install` refuses without a config):

```sh
monday install --dry-run            # preview the generated plist
monday install --hour 9 --minute 0  # install it (runs daily at 09:00)
monday uninstall                    # remove it
```

The agent fires every day and lets `monday` decide which profiles are due, so
the plist never desyncs from your config — change a profile's `days` and the
agent keeps working without reinstalling. (launchd also coalesces runs missed
while the Mac was asleep.) The agent is written to
`~/Library/LaunchAgents/io.monday.agent.plist` and logs to
`~/Library/Logs/monday.log`.

## MCP server (AI integration)

`monday` ships an MCP server so AI agents can run maintenance for you:

```sh
monday mcp     # speaks MCP over stdio
```

It exposes one `run_<task>` tool per task, a `run_all` tool, and `list_tasks`,
all generated from the same task registry the CLI uses. Each tool accepts a
`dry_run` boolean. Example client config:

```json
{
  "mcpServers": {
    "monday": { "command": "monday", "args": ["mcp"] }
  }
}
```

Inspect it locally with the MCP Inspector:

```sh
npx @modelcontextprotocol/inspector monday mcp
```

## Architecture

Every maintenance feature implements a small `Task` interface
(`internal/task`). A registry (`internal/registry`) holds them in order; the
schedule (`internal/schedule`) decides which profiles are due, and the runner
(`internal/runner`) runs the union of those profiles' tasks (intersected with
the enabled set), or an explicit `--only` list. The CLI (`cmd/`) and the MCP
server (`internal/mcpserver`) are two front-ends over that same registry, so
adding a task makes it available everywhere at once. Notifications live in
`internal/notify`.

Adding a task: implement `task.Task` — or reuse `task.NewCommand` for the
"run one command" shape, or `task.NewSteps` for a fixed sequence of commands
against one binary (as `brew` does) — register it in
`internal/registry/builtins.go`, and add a config block in `internal/config`.

## Development

```sh
go test ./...           # unit tests (no real commands are executed)
go test -cover ./...    # with coverage
go build ./...
```

All task tests inject a fake `Commander` (`internal/exec`), so nothing shells
out to the real system.

## Releasing

Releases are cut by GoReleaser on a `v*` tag via GitHub Actions. One-time setup:

1. Create the tap repo `github.com/JadoJodo/homebrew-tap` (can be empty).
2. Add a `HOMEBREW_TAP_GITHUB_TOKEN` repository secret — a PAT with `repo`
   scope on the tap — to this repo's Actions secrets.
3. Tag and push: `git tag v0.1.0 && git push origin v0.1.0`.

Try a local build without publishing:

```sh
goreleaser release --snapshot --clean
```

## License

MIT — see [LICENSE](LICENSE).
