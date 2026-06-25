# Claude Cleaner

[![CI](https://github.com/ePlus-DEV/claude-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/ePlus-DEV/claude-cleaner/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ePlus-DEV/claude-cleaner)](https://github.com/ePlus-DEV/claude-cleaner/releases)
[![npm version](https://img.shields.io/npm/v/claude-cleaner.svg)](https://www.npmjs.com/package/claude-cleaner)
[![Go version](https://img.shields.io/github/go-mod/go-version/ePlus-DEV/claude-cleaner)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/ePlus-DEV/claude-cleaner)](https://goreportcard.com/report/github.com/ePlus-DEV/claude-cleaner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**Claude Cleaner** is an interactive terminal UI — built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) — that inspects Claude Code project session history, displays disk usage, and safely deletes only the sessions you select.

Runs on Windows, macOS, and Linux. No runtime required when using a pre-built binary.

![Full demo](demo/full.gif)

## Demos

| Scenario | Preview |
| --- | --- |
| `--help` | ![Help](demo/help.gif) |
| Delete a session | ![Full flow](demo/full.gif) |
| Cancel confirmation | ![Cancel](demo/cancel.gif) |
| In-place update | ![Update](demo/update.gif) |

## Features

- Reads project list from `~/.claude.json` — shows all projects Claude Code knows about, even those with no local session files.
- Displays **token usage** per project (from `lastTotal*` fields in `~/.claude.json`), formatted as K / M / B / T / P / E.
- Status column `●` (session files on disk) / `○` (config only, no local data).
- Windows path dedup — `d:/foo` and `D:/foo` treated as the same project; higher-token entry wins.
- Multi-select with `space`, select all with `a`, confirm with `enter`.
- Three deletion modes: session-files delete, full **purge** (via `claude project purge`), and instant **force-purge** (`x`).
- Live progress bar during deletion.
- Claude CLI integration — tries `claude project purge` first, falls back to direct removal.
- Auto update check against npm registry on startup; `u` to update in-place.
- Claude CLI presence and version shown in header.
- `r` to rescan at any time.
- `q` quits from every screen.
- Rejects paths outside the Claude `projects` directory.
- Concurrent filesystem scanning.
- Supports custom Claude configuration directories via `--claude-dir` or `CLAUDE_CONFIG_DIR`.

## What it deletes

Only project session folders directly inside `~/.claude/projects` (or `$CLAUDE_CONFIG_DIR/projects`).

These folders contain Claude Code session and conversation history. Source code directories are never touched.

### Deletion modes

| Mode | Key | Confirm | Scope | How |
| --- | --- | --- | --- | --- |
| Delete | `enter` | ✓ screen | selected items | tries `claude project purge -y <path>`, falls back to `os.RemoveAll` |
| Purge | `p` | ✓ screen | selected items | same as delete, confirm text emphasises full purge |
| Force-purge | `x` | ✗ | cursor item only | same deletion chain, no confirm screen |
| Delete all | `a` then `enter` | ✓ screen | all items | uses `claude project purge --all -y` (single call), then cleans remaining folders |

All modes validate that the target path is inside the Claude projects directory before deleting.

## Install

### Run without installing

```bash
npx claude-cleaner
```

### Install globally

```bash
npm install --global claude-cleaner
claude-cleaner
```

> The npm package is a thin wrapper. On install it automatically downloads the correct pre-built binary for your platform from GitHub Releases. No Go required.

### Download binary directly

Go to [Releases](https://github.com/ePlus-DEV/claude-cleaner/releases), download the archive for your platform, extract, and run.

| Platform | File |
| --- | --- |
| Linux x64 | `claude-cleaner_*_linux_amd64.tar.gz` |
| Linux ARM64 | `claude-cleaner_*_linux_arm64.tar.gz` |
| macOS x64 | `claude-cleaner_*_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `claude-cleaner_*_darwin_arm64.tar.gz` |
| Windows x64 | `claude-cleaner_*_windows_amd64.zip` |
| Windows ARM64 | `claude-cleaner_*_windows_arm64.zip` |

### Install with Go

```bash
go install github.com/ePlus-DEV/claude-cleaner@latest
```

### Build from source

```bash
git clone https://github.com/ePlus-DEV/claude-cleaner.git
cd claude-cleaner
go build -o claude-cleaner .
./claude-cleaner
```

## Usage

```bash
claude-cleaner
claude-cleaner --claude-dir "/path/to/.claude"
claude-cleaner --help
claude-cleaner --version
```

### Options

```text
--claude-dir <path>   Custom Claude config directory (default: ~/.claude)
--mock-update         Simulate a newer version available (for testing the update flow)
-h, --help            Show help
-v, --version         Show version
```

### Key bindings

| Key | Action |
| --- | --- |
| `↑` / `↓` or `j` / `k` | Navigate list |
| `space` | Toggle selection |
| `enter` | Proceed — show delete confirm (when items selected) |
| `a` | Select all / deselect all |
| `p` | Purge selected (confirm screen, purge mode) |
| `x` | Force-purge item at cursor — no confirm |
| `r` | Rescan / refresh project list |
| `u` | Update claude-cleaner in-place (shown when update available) |
| `esc` | Go back / cancel |
| `q` / `ctrl+c` | Quit (works on every screen) |

## Configure a custom Claude directory

Priority order: `--claude-dir` > `CLAUDE_CONFIG_DIR` > `~/.claude`

```bash
# macOS / Linux
export CLAUDE_CONFIG_DIR="/mnt/data/claude"
claude-cleaner
```

```powershell
# Windows PowerShell
$env:CLAUDE_CONFIG_DIR = "D:\ClaudeData"
claude-cleaner
```

## Troubleshooting

**Claude directory not found** — Run Claude Code at least once so the directory is created, or point to the correct path:

```bash
claude-cleaner --claude-dir "/correct/path/.claude"
```

**Permission denied** — Run as the same OS user that owns the Claude config directory.

**Binary not found after `npx`** — Try reinstalling:

```bash
npm install --global claude-cleaner
```

**Windows: `Access is denied` when running `go run .`** — Windows locks the temp executable while it's in use. Kill any other running instances, or build once and run the binary directly:

```powershell
go build -o claude-cleaner.exe .
.\claude-cleaner.exe
```

## Development

See [CONTRIBUTING.md](CONTRIBUTING.md) for full setup, build, and release instructions.

### Requirements

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) (only for seed data / demo GIFs)

### Quick start

```bash
git clone https://github.com/ePlus-DEV/claude-cleaner.git
cd claude-cleaner
go mod tidy
```

### Run in dev (without installing Claude)

Use the included seed script to create fake session data for testing:

```bash
# macOS / Linux
node demo/seed.js /tmp/claude-demo
go run . --claude-dir /tmp/claude-demo
```

```powershell
# Windows
node demo/seed.js $env:TEMP\claude-demo
go run . --claude-dir $env:TEMP\claude-demo
```

This creates 5 fake project sessions of various sizes — enough to test all TUI flows (navigate, select, delete, cancel) without touching any real Claude data.

### Common commands

```bash
go mod tidy          # install / tidy dependencies
go build -o claude-cleaner .  # build binary
go run .             # run without building
go test -v ./...     # run tests
go run . --version   # smoke test
```

## CI / CD

| Workflow | Trigger | What it does |
| --- | --- | --- |
| [ci.yml](.github/workflows/ci.yml) | push / PR | Go tests on 1.22, 1.23, 1.24 × Windows, macOS, Linux |
| [release.yml](.github/workflows/release.yml) | push `v*` tag | GoReleaser builds binaries → GitHub Release → npm publish |
| [demo.yml](.github/workflows/demo.yml) | push to main (Go / tape files) | Regenerates demo GIFs via VHS |

### Publishing a release

```bash
npm version patch     # or minor / major
git push --follow-tags
```

`npm version` automatically syncs the version to `main.go` and creates a git tag. Pushing the tag triggers GoReleaser and npm publish.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE) © ePlus.DEV
