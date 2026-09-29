# Roadmap

This roadmap tracks the planned direction for Claude Cleaner. Priorities may change based on Claude Code changes, user feedback, and safety requirements.

## v1.3.0 — Safer cleanup workflows

Focus: make destructive actions easier to understand and harder to misuse.

- Cleanup presets: Conservative, Balanced, and Aggressive.
- Estimated reclaimable disk space before cleanup.
- Better cleanup summary grouped by projects, conversations, and disposable categories.
- Undo-friendly cleanup through an optional local trash/quarantine mode before permanent deletion.
- Configurable protection rules for frequently used or recently active projects.
- Improved confirmation screens with clearer impact summaries.
- Better handling of partially failed deletions and locked files.
- Export cleanup results as JSON for scripting and diagnostics.

## v1.4.0 — Automation and non-interactive mode

Focus: support repeatable cleanup without requiring the TUI.

- Non-interactive CLI commands for scan, inspect, clean, and purge.
- Machine-readable JSON output.
- Filters by age, size, token usage, project path, and orphan state.
- Config file support for reusable cleanup policies.
- Exit codes suitable for shell scripts and CI.
- Scheduled-cleanup friendly commands for cron, Task Scheduler, and launchd.
- Optional maximum disk-usage target, for example clean until Claude data is below a configured size.

## v1.5.0 — Storage insights

Focus: explain where Claude Code storage is going.

- Storage overview dashboard.
- Historical cleanup statistics.
- Largest-project and largest-conversation views.
- Token usage trends where source data is available.
- Duplicate/stale session detection.
- Better orphan detection and explanation of why a project is considered orphaned.
- Per-category storage breakdown for logs, history, backups, telemetry, caches, and sessions.

## v1.6.0 — Profiles and policy engine

Focus: make cleanup behavior reusable across machines and teams.

- Named cleanup profiles.
- Include/exclude path rules.
- Per-project retention periods.
- Permanent project protection rules.
- Dry-run policy evaluation with an explanation for every selected item.
- Import/export of configuration.
- Environment-variable overrides for automation environments.

## v2.0.0 — Extensible Claude data manager

Focus: evolve from a session cleaner into a broader Claude Code local-data management tool while keeping source-code safety guarantees.

- Plugin-based cleanup providers for new Claude Code data locations.
- Backup and restore workflows for supported Claude-owned data.
- Cross-version compatibility layer for Claude Code storage format changes.
- Migration helpers when Claude Code changes its local data layout.
- Health checks for malformed or inconsistent Claude metadata.
- Optional interactive setup wizard.
- Stable command API for third-party tooling.

## Ongoing

These areas are continuously maintained across releases:

- Windows, macOS, and Linux compatibility.
- Safety checks that prevent source project deletion.
- Claude Code storage-format compatibility.
- Startup and scan performance.
- Accessibility and keyboard-first TUI behavior.
- Release automation and npm binary distribution.
- Tests for destructive operations and path validation.

## Ideas under evaluation

These are intentionally not committed to a release yet:

- Interactive disk-usage visualization.
- Restore points before bulk destructive actions.
- Shell completion for Bash, Zsh, Fish, and PowerShell.
- Optional notification after scheduled cleanup.
- Integration hooks for external developer tools.
- Additional Claude-compatible local clients if their data model can be handled safely.

Have a feature request? Open a GitHub issue with the use case, expected behavior, platform, and Claude Code version.
