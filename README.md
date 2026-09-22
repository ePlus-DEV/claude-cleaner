# Claude Cleaner

[![CI](https://github.com/ePlus-DEV/claude-cleaner/actions/workflows/ci.yml/badge.svg)](https://github.com/ePlus-DEV/claude-cleaner/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/ePlus-DEV/claude-cleaner)](https://github.com/ePlus-DEV/claude-cleaner/releases)
[![npm version](https://img.shields.io/npm/v/claude-cleaner.svg)](https://www.npmjs.com/package/claude-cleaner)
[![Go version](https://img.shields.io/github/go-mod/go-version/ePlus-DEV/claude-cleaner)](go.mod)
[![Go Report Card](https://goreportcard.com/badge/github.com/ePlus-DEV/claude-cleaner)](https://goreportcard.com/report/github.com/ePlus-DEV/claude-cleaner)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Known Vulnerabilities](https://snyk.io/test/github/ePlus-DEV/claude-cleaner/badge.svg)](https://snyk.io/test/github/ePlus-DEV/claude-cleaner)

**Claude Cleaner** is an interactive terminal UI — built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss) — that inspects Claude Code project session history, displays disk usage, and safely deletes only the sessions you select.

Runs on Windows, macOS, and Linux. No runtime required when using a pre-built binary.

![Full demo](demo/full.gif)

> See [SCREENSHOTS.md](SCREENSHOTS.md) for all scenario walkthroughs.

## Install

### Run without installing

```bash
npx claude-cleaner
```

### npm (global)

```bash
npm install --global claude-cleaner
claude-cleaner
```

> The npm package is a thin wrapper. On install it automatically downloads the correct pre-built binary for your platform from GitHub Releases. No Go required.

### Download binary

Go to [Releases](https://github.com/ePlus-DEV/claude-cleaner/releases), download the archive for your platform, extract, and run.

| Platform | File |
| --- | --- |
| Linux x64 | `claude-cleaner_*_linux_amd64.tar.gz` |
| Linux ARM64 | `claude-cleaner_*_linux_arm64.tar.gz` |
| macOS x64 | `claude-cleaner_*_darwin_amd64.tar.gz` |
| macOS Apple Silicon | `claude-cleaner_*_darwin_arm64.tar.gz` |
| Windows x64 | `claude-cleaner_*_windows_amd64.zip` |
| Windows ARM64 | `claude-cleaner_*_windows_arm64.zip` |

### Go

```bash
go install github.com/ePlus-DEV/claude-cleaner@latest
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
--dry-run             Preview deletions without modifying any files
--mock-update         Simulate a newer version available (for testing the update flow)
-h, --help            Show help
-v, --version         Show version
```

### Key bindings

| Key | Action |
| --- | --- |
| `↑` / `↓` or `j` / `k` | Navigate list |
| `g` / `G` | Jump to top / bottom |
| `space` | Toggle selection |
| `a` | Select / deselect all visible items |
| `n` | Unselect all |
| `o` | Select orphaned projects only |
| `enter` | Open Project Detail when nothing is selected; otherwise delete selected project session-history directories |
| `l` | Lock / unlock project (protected projects are skipped by destructive bulk actions) |
| `X` | Forget project: remove Claude session data + project metadata, never source code |
| `p` | Full purge selected projects through Claude CLI when available |
| `x` | Force-purge item at cursor — no confirm |
| `s` | Cycle sort: recent / size / tokens / name |
| `f` | Cycle filter: all / has data / orphaned |
| `e` | Cycle expiry filter: off / 7 / 14 / 30 / 60 / 90 days |
| `/` | Search by project name or path |
| `c` | Open category cleanup |
| `r` | Rescan / refresh project list |
| `u` | Update claude-cleaner in-place (shown when update available) |
| `?` | Show key bindings |
| `esc` | Go back / clear search and filters / cancel |
| `q` / `ctrl+c` | Quit (works on every screen) |

## Features

- Reads project list from `~/.claude.json` — shows all projects Claude Code knows about, even those with no local session files.
- Displays **token usage** per project — reads `lastTotal*` fields from `~/.claude.json` when available, otherwise aggregates `message.usage` from session `.jsonl` files. Formatted as K / M / B / T / P / E.
- Status column `●` (session files on disk) / `○` (config only, no local data).
- Windows path dedup — `d:/foo` and `D:/foo` treated as the same project; higher-token entry wins.
- Project Detail screen with per-conversation JSONL session count, modified time, message count, token usage, and size.
- Delete individual conversation sessions without removing the entire project history.
- Protected Projects with `l`: locked projects are excluded from select-all, orphan selection, delete, purge, and forget operations.
- Forget Project with `X`: removes Claude-owned session data and the matching `~/.claude.json` project entry while preserving the real source project.
- Project list includes conversation/session counts alongside size, token usage, and last-modified time.
- Multi-select with `space`, select all with `a`, confirm with `enter`.
- Separate deletion backends: normal **Delete** removes only the selected Claude session-history directory; **Purge** uses `claude project purge` when available and falls back to session-directory removal.
- `--dry-run` previews exactly which projects/categories would be cleaned without touching files.
- Search, sort, orphan filters, and age/expiry filters for large project lists.
- Category cleanup for disposable Claude data such as debug logs, telemetry, history, backups, and plugin **cache** while preserving plugin installation state.
- Live progress bar during deletion.
- Claude CLI integration is used only for explicit purge operations.
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
| Delete | `enter` with selected projects | ✓ screen | selected items | removes only the matching directory under `~/.claude/projects`; never invokes `claude project purge` |
| Delete conversation | Project Detail → `enter` | ✓ screen | selected JSONL sessions | removes only selected conversation files inside one project |
| Forget project | `X` | ✓ screen | current/selected projects | removes Claude session data and matching `~/.claude.json` metadata; source code is never touched |
| Purge | `p` | ✓ screen | selected items | runs `claude project purge -y <path>` when available; falls back to the matching session directory |
| Force-purge | `x` | ✗ | cursor item only | same purge chain as `p`, without a confirm screen |
| Delete all | `a` then `enter` | ✓ screen | all visible/selected items | deletes selected session directories individually; does not call `purge --all` |
| Purge all | `a` then `p` | ✓ screen | all projects | may use `claude project purge --all -y` for efficiency |

All modes validate that the target path is inside the Claude projects directory before deleting.

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

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, build, test, and release instructions. For internal data flow diagrams see [ARCHITECTURE.md](ARCHITECTURE.md).

## License

[MIT](LICENSE) © ePlus.DEV
