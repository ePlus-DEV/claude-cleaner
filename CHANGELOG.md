# Changelog

## Unreleased

- Added Project Detail with per-conversation session listing, message count, token usage, size, and timestamps.
- Added individual conversation deletion from Project Detail.
- Added Protected Projects with persistent lock/unlock state; destructive bulk actions skip locked projects.
- Added Forget Project to remove Claude-owned session data and matching `~/.claude.json` metadata without touching source code.
- Added project-level conversation counts and richer statistics in the main list.
- Added Windows-safe config replacement when forgetting projects.

## 1.2.0 - 2026-09-22

- Fixed Delete vs Purge semantics: normal Delete now removes only Claude session-history directories and never invokes `claude project purge`; explicit Purge/Force-purge retain Claude CLI integration.
- Token aggregation now handles JSONL lines larger than 1 MB without silently truncating the scan.
- Category cleanup now measures nested files recursively so previewed reclaimable size matches the cleanup scope.
- Plugin cleanup is limited to `~/.claude/plugins/cache` and preserves plugin installation/marketplace state.
- Config-backup cleanup now reports file-removal failures instead of silently succeeding.
- Token column now falls back to summing `message.usage` from session `.jsonl` files when `~/.claude.json` does not contain `lastTotal*` token fields (common on newer Claude Code installs).
- UI Project column shows only the last folder name (e.g. `g-front`) instead of the full path; full path is still used internally for correct deletion.
- Bumped minimum Go version to 1.25 (go.mod).
- Updated CI matrix to Go 1.25 / 1.26 across Windows, macOS, and Linux.
- Updated all workflows (ci, demo, release) to Go 1.25.
- Added Snyk security scanning workflow (push / PR + weekly schedule).
- Upgraded dependencies to fix HIGH/MEDIUM Snyk findings:
  - `golang.org/x/text` v0.3.8 → v0.38.0 (CWE-1327)
  - `golang.org/x/sys` v0.27.0 → v0.46.0 (CWE-190)
  - `github.com/charmbracelet/bubbletea` v1.2.4 → v1.3.10
  - `github.com/charmbracelet/bubbles` v0.20.0 → v1.0.0
  - `github.com/charmbracelet/lipgloss` v1.0.0 → v1.1.0
- Restructured README: install section promoted, dev content moved to CONTRIBUTING.md.
- Improved asynchronous directory scanning and deletion safety.
- Added automated tests on Windows, macOS, and Linux.
- Added OIDC-based npm publishing and tag-based GitHub Release automation.
- Added optional `NPM_TOKEN` bootstrap support for the first npm publication.
- Removed generated npm `always-auth` configuration from the release workflow.
- Expanded installation, usage, troubleshooting, and release documentation.

## 1.0.0

- Initial npm-ready release.
- Interactive project session selection.
- Cross-platform support for Windows, macOS, and Linux.
- Supports `--claude-dir`, `--help`, and `--version`.
