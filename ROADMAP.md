# Roadmap

Claude Cleaner evolves from a safe interactive session cleaner into a complete local Claude Code data manager.

**Status:** ✅ Released · 🚧 In progress · 🗓️ Planned · 🔭 Exploring

```text
v1.0          v1.1          v1.2          v1.3          v1.4          v1.5          v1.6          v2.0
  ●─────────────●─────────────●─────────────●─────────────◐─────────────○─────────────○─────────────○
  │             │             │             │             │             │             │             │
Initial       UX &          Safety &      Project &      Automation    Storage       Policies      Extensible
cleaner       workflow      reliability   sessions       & CLI         insights      & profiles    data manager
  ✅            ✅            ✅            ✅            🚧            🗓️            🗓️            🔭
```

## Progress

| Version | Status | Theme | Highlights |
| --- | --- | --- | --- |
| **v1.0.0** | ✅ Released | Initial cleaner | Interactive project selection, safe session-history cleanup, custom Claude directory, Windows/macOS/Linux |
| **v1.1.x** | ✅ Released | UX & workflow | TUI workflow improvements and the foundation for the richer cleanup experience |
| **v1.2.0** | ✅ Released | Safety & reliability | Delete vs Purge separation, recursive cleanup sizing, token fallback from JSONL, plugin-cache safety, faster scanning, cross-platform tests, automated releases |
| **v1.3.0** | ✅ Released | Project & session manager | Project Detail, per-conversation inspection/deletion, Protected Projects, Forget Project, conversation counts, richer project statistics |
| **v1.4.0** | 🚧 In progress | Automation & non-interactive CLI | Scriptable scan/clean workflows, JSON output, reusable filters and cleanup configuration |
| **v1.5.0** | 🗓️ Planned | Storage insights | Storage dashboard, cleanup history, largest projects/conversations, stale-session detection |
| **v1.6.0** | 🗓️ Planned | Profiles & policy engine | Named profiles, retention rules, include/exclude policies, permanent protection, config import/export |
| **v2.0.0** | 🔭 Exploring | Extensible Claude data manager | Backup/restore, storage migrations, health checks, extensible cleanup providers, stable command API |

---

## ✅ v1.0.0 — Initial cleaner

The first usable Claude Cleaner release.

- Interactive Claude Code project-session selection.
- Safe deletion scoped to Claude-owned session history.
- Cross-platform support for Windows, macOS, and Linux.
- Custom Claude directory support with `--claude-dir`.
- CLI help and version commands.

## ✅ v1.1.x — UX & workflow

The first iteration focused on making the cleaner practical for repeated interactive use.

- Improved interactive TUI workflow.
- Expanded project inspection and cleanup controls.
- Established the workflow that later releases extended with filtering, categories, purge modes, and project management.

## ✅ v1.2.0 — Safety & reliability

A major hardening release.

- Clear separation between normal **Delete** and explicit **Purge**.
- Token aggregation from session JSONL when Claude metadata lacks token totals.
- Recursive reclaimable-size calculation for category cleanup.
- Plugin cleanup restricted to cache data so installation state is preserved.
- Improved filesystem scanning and deletion safety.
- Go 1.25 baseline and dependency/security upgrades.
- Automated Windows, macOS, and Linux tests.
- Automated GitHub Release and npm publishing flow.

## ✅ v1.3.0 — Project & session manager

Claude Cleaner moved beyond deleting entire project histories and became a project/session manager.

- Project Detail with individual conversation sessions.
- Per-session message count, token usage, size, and timestamps.
- Delete individual conversations.
- Protected Projects with persistent lock/unlock state.
- Destructive bulk actions automatically skip protected projects.
- Forget Project removes Claude-owned data and metadata without touching source code.
- Project-level conversation counts and richer statistics.
- Windows-safe Claude config replacement.

---

## 🚧 v1.4.0 — Automation & non-interactive CLI

**Current direction:** make Claude Cleaner useful outside the interactive TUI.

- [ ] Non-interactive `scan`, `inspect`, `clean`, and `purge` commands.
- [ ] Machine-readable JSON output.
- [ ] Filters for age, size, token usage, project path, and orphan state.
- [ ] Reusable cleanup configuration.
- [ ] Stable exit codes for shell scripts and CI.
- [ ] Cron, Task Scheduler, and launchd-friendly execution.
- [ ] Optional target-size cleanup, e.g. clean until Claude data is below a configured limit.
- [ ] Cleanup summary suitable for automation logs.

### Example direction

```bash
claude-cleaner scan --json
claude-cleaner clean --older-than 30d --dry-run
claude-cleaner clean --profile conservative
claude-cleaner clean --target-size 2GB
```

## 🗓️ v1.5.0 — Storage insights

Make it easier to understand what is consuming local Claude Code storage.

- [ ] Storage overview dashboard.
- [ ] Historical cleanup statistics.
- [ ] Largest-project and largest-conversation views.
- [ ] Token usage trends where source data is available.
- [ ] Duplicate/stale session detection.
- [ ] Better orphan detection with an explanation of why an item is orphaned.
- [ ] Storage breakdown across sessions, logs, history, backups, telemetry, and caches.

## 🗓️ v1.6.0 — Profiles & policy engine

Turn repeated cleanup decisions into reusable policies.

- [ ] Named cleanup profiles.
- [ ] Include/exclude path rules.
- [ ] Per-project retention periods.
- [ ] Permanent project protection rules.
- [ ] Explainable dry-run policy evaluation.
- [ ] Import/export configuration.
- [ ] Environment-variable overrides for automation.

## 🔭 v2.0.0 — Extensible Claude data manager

Long-term direction rather than a committed release scope.

- [ ] Backup and restore workflows for supported Claude-owned data.
- [ ] Extensible cleanup providers for new Claude Code data locations.
- [ ] Compatibility layer for Claude Code storage-format changes.
- [ ] Migration helpers when Claude changes its local data layout.
- [ ] Health checks for malformed or inconsistent metadata.
- [ ] Optional interactive setup wizard.
- [ ] Stable command API for integrations and third-party tooling.

---

## Continuous work

These do not wait for a specific release:

- Cross-platform compatibility.
- Source-code deletion safeguards.
- Claude Code storage-format compatibility.
- Scan/startup performance.
- Keyboard-first TUI and accessibility.
- Destructive-operation and path-validation tests.
- npm/GitHub release reliability.
- Documentation and demo coverage.

## Ideas under evaluation

Not assigned to a release yet:

- Trash/quarantine mode and restore points before permanent deletion.
- Cleanup presets such as Conservative / Balanced / Aggressive.
- Interactive disk-usage visualization.
- Shell completion for Bash, Zsh, Fish, and PowerShell.
- Optional scheduled-cleanup notifications.
- Integration hooks for external developer tools.
- Additional Claude-compatible local clients when they can be handled safely.

> Roadmap scope is intentionally flexible. Released functionality reflects the repository history; future milestones may move as Claude Code changes and feedback arrives.
