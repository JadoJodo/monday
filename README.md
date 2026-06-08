# monday

Automate routine macOS maintenance from one command. `monday` runs your weekly
chores — system software updates, Mac App Store updates, global npm packages and
your own custom scripts — on a schedule, with a single CLI. Tasks are modular and
toggled via a YAML config, and the whole thing is exposed to AI agents over MCP.

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
monday config init      # write a sample ~/.monday.yaml
monday list             # show tasks and their enabled state
monday run --dry-run    # preview what would happen, changing nothing
monday                  # run maintenance (applies updates) if today is the scheduled day
monday run --force      # run now regardless of the schedule
```

By default `monday` **applies** updates. Use `--dry-run` to preview first.

## Tasks

| Task             | Dry-run                | Apply                |
| ---------------- | ---------------------- | -------------------- |
| `softwareupdate` | `softwareupdate -l`    | `softwareupdate -ia` |
| `mas`            | `mas outdated`         | `mas upgrade`        |
| `npm`            | `npm -g outdated`      | `npm -g update`      |
| `custom`         | lists configured cmds  | runs each via `sh -c`|

Tasks whose underlying tool is not installed (e.g. `mas`) are skipped, not failed.

## Configuration

`monday config init` writes `~/.monday.yaml`:

```yaml
schedule:
  day: monday          # any weekday; override per-run with --day or --force

tasks:
  softwareupdate:
    enabled: true
  mas:
    enabled: true
  npm:
    enabled: true
  custom:
    enabled: true
    scripts:
      - name: brew-upgrade
        run: brew upgrade
```

A missing config file is fine — every task is enabled and the schedule defaults
to Monday. A task is only disabled by an explicit `enabled: false`.

Useful flags:

- `--config <path>` — use a non-default config file
- `--dry-run` — preview without changing anything
- `--only npm,custom` — run just the named tasks (implies `--force`)
- `--day friday` / `--force` — override the schedule
- `-V`, `--verbose` — show command output detail

## Automatic scheduling (launchd)

Install a per-user LaunchAgent that runs `monday` on the configured weekday:

```sh
monday install --dry-run            # preview the generated plist
monday install --hour 9 --minute 0  # install it (runs at 09:00 on the scheduled day)
monday uninstall                    # remove it
```

The agent is written to `~/Library/LaunchAgents/io.monday.agent.plist` and logs
to `~/Library/Logs/monday.log`. The weekday is taken from your config's
`schedule.day` at install time, so re-run `monday install` after changing it.

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
runner (`internal/runner`) checks the schedule, filters by config/`--only`, and
executes each one. The CLI (`cmd/`) and the MCP server (`internal/mcpserver`)
are two front-ends over that same registry, so adding a task makes it available
everywhere at once.

Adding a task: implement `task.Task` (or reuse `task.NewCommand` for the common
"run one command" shape), register it in `internal/registry/builtins.go`, and
add a config block in `internal/config`.

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
