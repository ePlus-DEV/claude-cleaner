package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Session struct {
	Index        int
	Name         string
	Path         string
	ProjectPath  string // actual working dir from ~/.claude.json; empty if unknown
	Modified     time.Time
	Oldest       time.Time
	Size         int64
	TotalTokens  int64
	SessionCount int
	HasTokenData bool // false = no token data in ~/.claude.json or session .jsonl files (show "—")
	HasData      bool // false = directory absent or empty (no session files found)
}

// ProjectSession is one Claude conversation JSONL file inside a project.
type ProjectSession struct {
	Index        int
	ID           string
	Path         string
	Modified     time.Time
	Size         int64
	TotalTokens  int64
	MessageCount int
	HasTokenData bool
}

// projectEntry mirrors the token fields stored per-project in ~/.claude.json.
type projectEntry struct {
	LastTotalInputTokens              *int64 `json:"lastTotalInputTokens"`
	LastTotalOutputTokens             *int64 `json:"lastTotalOutputTokens"`
	LastTotalCacheCreationInputTokens *int64 `json:"lastTotalCacheCreationInputTokens"`
	LastTotalCacheReadInputTokens     *int64 `json:"lastTotalCacheReadInputTokens"`
}

func (e projectEntry) total() int64 {
	var n int64
	if e.LastTotalInputTokens != nil {
		n += *e.LastTotalInputTokens
	}
	if e.LastTotalOutputTokens != nil {
		n += *e.LastTotalOutputTokens
	}
	if e.LastTotalCacheCreationInputTokens != nil {
		n += *e.LastTotalCacheCreationInputTokens
	}
	if e.LastTotalCacheReadInputTokens != nil {
		n += *e.LastTotalCacheReadInputTokens
	}
	return n
}

func (e projectEntry) hasAnyField() bool {
	return e.LastTotalInputTokens != nil ||
		e.LastTotalOutputTokens != nil ||
		e.LastTotalCacheCreationInputTokens != nil ||
		e.LastTotalCacheReadInputTokens != nil
}

// jsonlTokens holds just the four token fields from message.usage in a .jsonl line.
type jsonlTokens struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
}

// scanSessionFile reads one Claude conversation JSONL file.
func scanSessionFile(path string) (total int64, hasTokenData bool, messageCount int) {
	f, err := os.Open(path)
	if err != nil {
		return 0, false, 0
	}
	defer f.Close()

	r := bufio.NewReaderSize(f, 64*1024)
	for {
		line, readErr := r.ReadBytes('\n')
		if len(line) > 0 {
			var row struct {
				Type    string          `json:"type"`
				Message json.RawMessage `json:"message"`
			}
			if json.Unmarshal(line, &row) == nil && len(row.Message) > 0 && string(row.Message) != "null" {
				messageCount++
				if row.Type == "assistant" {
					var msg struct {
						Usage *jsonlTokens `json:"usage"`
					}
					if json.Unmarshal(row.Message, &msg) == nil && msg.Usage != nil {
						u := msg.Usage
						total += u.InputTokens + u.OutputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens
						hasTokenData = true
					}
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	return total, hasTokenData, messageCount
}

// scanProjectTokens sums token usage from all top-level .jsonl session files.
func scanProjectTokens(dirPath string) (int64, bool) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, false
	}
	var total int64
	var hasData bool
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".jsonl") {
			continue
		}
		tokens, hasTokens, _ := scanSessionFile(filepath.Join(dirPath, e.Name()))
		total += tokens
		hasData = hasData || hasTokens
	}
	return total, hasData
}

// scanProjectSessions returns individual conversation files for Project Detail.
func scanProjectSessions(dirPath string) ([]ProjectSession, error) {
	entries, err := os.ReadDir(dirPath)
	if os.IsNotExist(err) {
		return []ProjectSession{}, nil
	}
	if err != nil {
		return nil, err
	}

	var sessions []ProjectSession
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		path := filepath.Join(dirPath, e.Name())
		tokens, hasTokens, messages := scanSessionFile(path)
		sessions = append(sessions, ProjectSession{
			ID:           strings.TrimSuffix(e.Name(), filepath.Ext(e.Name())),
			Path:         path,
			Modified:     info.ModTime(),
			Size:         info.Size(),
			TotalTokens:  tokens,
			MessageCount: messages,
			HasTokenData: hasTokens,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	for i := range sessions {
		sessions[i].Index = i + 1
	}
	return sessions, nil
}

func projectSessionSummary(dirPath string) (count int, oldest time.Time) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0, time.Time{}
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(strings.ToLower(e.Name()), ".jsonl") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		count++
		if oldest.IsZero() || info.ModTime().Before(oldest) {
			oldest = info.ModTime()
		}
	}
	return count, oldest
}

// normalizePath lowercases and normalises separators so Windows paths are
// compared case-insensitively (d:/Foo and D:\foo → d:/foo).
func normalizePath(p string) string {
	return strings.ToLower(strings.ReplaceAll(p, "\\", "/"))
}

// encodePath converts an actual project path to the hashed directory name
// that Claude Code uses under ~/.claude/projects/.
func encodePath(path string) string {
	r := strings.NewReplacer(":", "-", "/", "-", "\\", "-")
	return strings.ToLower(r.Replace(path))
}

// projectStats reads a flat project directory: returns file size sum and most
// recent file mtime. No recursive walk — session files are always at top level.
func projectStats(dirPath string) (size int64, modified time.Time) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		size += info.Size()
		if info.ModTime().After(modified) {
			modified = info.ModTime()
		}
	}
	return
}

// deduplicateProjects merges duplicate project paths that differ only in
// case (Windows). Keeps the entry with the higher token total.
func deduplicateProjects(raw map[string]projectEntry) map[string]projectEntry {
	type best struct {
		path  string
		entry projectEntry
		total int64
	}
	seen := make(map[string]best, len(raw))
	for path, entry := range raw {
		norm := normalizePath(path)
		total := entry.total()
		if b, ok := seen[norm]; !ok || total > b.total {
			seen[norm] = best{path: path, entry: entry, total: total}
		}
	}
	out := make(map[string]projectEntry, len(seen))
	for _, b := range seen {
		out[b.path] = b.entry
	}
	return out
}

// scanSessions reads projects from claudeJSONPath (primary source of truth).
// Falls back to scanning projectsDir directly when claudeJSONPath is absent
// (e.g. custom --claude-dir for demos).
func scanSessions(claudeJSONPath, projectsDir string) ([]Session, error) {
	data, err := os.ReadFile(claudeJSONPath)
	if os.IsNotExist(err) {
		return scanFromDir(projectsDir)
	}
	if err != nil {
		return nil, err
	}

	var cfg struct {
		Projects map[string]projectEntry `json:"projects"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil || len(cfg.Projects) == 0 {
		return scanFromDir(projectsDir)
	}

	projects := deduplicateProjects(cfg.Projects)

	ch := make(chan Session, len(projects))
	var wg sync.WaitGroup

	for projectPath, entry := range projects {
		wg.Add(1)
		go func(projPath string, e projectEntry) {
			defer wg.Done()
			encoded := encodePath(projPath)
			dirPath := filepath.Join(projectsDir, encoded)
			size, modified := projectStats(dirPath)
			sessionCount, oldest := projectSessionSummary(dirPath)

			totalTokens := e.total()
			hasTokenData := e.hasAnyField()
			if !hasTokenData {
				totalTokens, hasTokenData = scanProjectTokens(dirPath)
			}
			ch <- Session{
				Name:         encoded,
				Path:         dirPath,
				ProjectPath:  projPath,
				Modified:     modified,
				Oldest:       oldest,
				Size:         size,
				TotalTokens:  totalTokens,
				SessionCount: sessionCount,
				HasTokenData: hasTokenData,
				HasData:      !modified.IsZero(),
			}
		}(projectPath, entry)
	}

	wg.Wait()
	close(ch)

	var sessions []Session
	for s := range ch {
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	for i := range sessions {
		sessions[i].Index = i + 1
	}

	return sessions, nil
}

// scanFromDir is the fallback: enumerate subdirectories of projectsDir directly.
// Token data is read from .jsonl session files since ~/.claude.json is absent.
func scanFromDir(projectsDir string) ([]Session, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	ch := make(chan Session, len(entries))
	var wg sync.WaitGroup

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wg.Add(1)
		go func(e os.DirEntry) {
			defer wg.Done()
			dirPath := filepath.Join(projectsDir, e.Name())
			size, modified := projectStats(dirPath)
			sessionCount, oldest := projectSessionSummary(dirPath)
			totalTokens, hasTokenData := scanProjectTokens(dirPath)
			ch <- Session{
				Name:         e.Name(),
				Path:         dirPath,
				Modified:     modified,
				Oldest:       oldest,
				Size:         size,
				TotalTokens:  totalTokens,
				SessionCount: sessionCount,
				HasTokenData: hasTokenData,
				HasData:      !modified.IsZero(),
			}
		}(entry)
	}

	wg.Wait()
	close(ch)

	var sessions []Session
	for s := range ch {
		sessions = append(sessions, s)
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})
	for i := range sessions {
		sessions[i].Index = i + 1
	}

	return sessions, nil
}

// DetectClaudeCLI returns the claude CLI version string, or empty string if not found.
func DetectClaudeCLI() string {
	path, err := exec.LookPath("claude")
	if err != nil {
		return ""
	}
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return "found"
	}
	return strings.TrimSpace(string(out))
}

// deleteSessionData removes only the Claude session-history directory for a
// project. It never invokes `claude project purge`.
func deleteSessionData(s Session, projectsDir string) error {
	return safeRemove(projectsDir, s.Path)
}

// purgeProject asks Claude Code to purge the project when possible, then falls
// back to removing its session-history directory if Claude leaves it behind.
func purgeProject(s Session, projectsDir string) error {
	if s.ProjectPath != "" {
		if _, err := exec.LookPath("claude"); err == nil {
			cmd := exec.Command("claude", "project", "purge", "-y", s.ProjectPath)
			_ = cmd.Run() // ignore error — check folder next
		}
	}
	// Folder already gone (claude handled it) — success.
	if _, err := os.Stat(s.Path); os.IsNotExist(err) {
		return nil
	}
	return safeRemove(projectsDir, s.Path)
}

// RunPurge executes a full purge for the given project snapshot.
// selected is a snapshot (caller must deep-copy before passing to avoid races).
// If all projects are selected and claude CLI is available, uses --all for efficiency.
func RunPurge(sessions []Session, selected map[int]bool, projectsDir string) (deleted, failed []string) {
	// Check if all sessions are selected
	allSelected := len(sessions) > 0
	for _, s := range sessions {
		if !selected[s.Index] {
			allSelected = false
			break
		}
	}

	if allSelected {
		if _, err := exec.LookPath("claude"); err == nil {
			cmd := exec.Command("claude", "project", "purge", "--all", "-y")
			if cmd.Run() == nil {
				// Verify each folder; clean up any that remain
				for _, s := range sessions {
					if _, statErr := os.Stat(s.Path); os.IsNotExist(statErr) {
						deleted = append(deleted, s.Name)
					} else {
						if err := safeRemove(projectsDir, s.Path); err != nil {
							failed = append(failed, s.Name)
						} else {
							deleted = append(deleted, s.Name)
						}
					}
				}
				return
			}
			// --all failed, fall through to per-project
		}
	}

	for _, s := range sessions {
		if !selected[s.Index] {
			continue
		}
		if err := purgeProject(s, projectsDir); err != nil {
			failed = append(failed, s.Name)
		} else {
			deleted = append(deleted, s.Name)
		}
	}
	return
}

// safeRemoveSessionFile removes one direct JSONL child of a project directory.
func safeRemoveSessionFile(projectDir, targetPath string) error {
	rel, err := filepath.Rel(filepath.Clean(projectDir), filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("invalid session path: %w", err)
	}
	if rel == "." || rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		strings.Contains(rel, string(filepath.Separator)) ||
		!strings.EqualFold(filepath.Ext(rel), ".jsonl") {
		return fmt.Errorf("refusing to delete file outside project session directory")
	}
	if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// forgetProject removes Claude-owned session data and the matching ~/.claude.json
// project entry. The user's source project directory is never touched.
func forgetProject(s Session, projectsDir, claudeJSONPath string) error {
	if s.Path != "" {
		if _, err := os.Stat(s.Path); err == nil {
			if err := safeRemove(projectsDir, s.Path); err != nil {
				return err
			}
	}
	}

	if s.ProjectPath == "" || claudeJSONPath == "" {
		return nil
	}
	data, err := os.ReadFile(claudeJSONPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}

	var root map[string]json.RawMessage
	if err := json.Unmarshal(data, &root); err != nil {
		return err
	}
	rawProjects, ok := root["projects"]
	if !ok {
		return nil
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(rawProjects, &projects); err != nil {
		return err
	}

	removed := false
	want := normalizePath(s.ProjectPath)
	for key := range projects {
		if normalizePath(key) == want {
			delete(projects, key)
			removed = true
		}
	}
	if !removed {
		return nil
	}

	updatedProjects, err := json.Marshal(projects)
	if err != nil {
		return err
	}
	root["projects"] = updatedProjects
	updated, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return err
	}
	updated = append(updated, '\n')

	mode := os.FileMode(0644)
	if info, statErr := os.Stat(claudeJSONPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmp := claudeJSONPath + ".claude-cleaner.tmp"
	if err := os.WriteFile(tmp, updated, mode); err != nil {
		return err
	}

	// Windows cannot reliably rename a file over an existing destination.
	// Move the original aside first and roll it back if replacement fails.
	backup := claudeJSONPath + ".claude-cleaner.bak"
	_ = os.Remove(backup)
	if err := os.Rename(claudeJSONPath, backup); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, claudeJSONPath); err != nil {
		_ = os.Rename(backup, claudeJSONPath)
		_ = os.Remove(tmp)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

func safeRemove(projectsDir, targetPath string) error {
	rel, err := filepath.Rel(filepath.Clean(projectsDir), filepath.Clean(targetPath))
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}
	if rel == "." ||
		rel == ".." ||
		strings.HasPrefix(rel, ".."+string(filepath.Separator)) ||
		strings.Contains(rel, string(filepath.Separator)) {
		return fmt.Errorf("refusing to delete path outside projects directory")
	}
	return os.RemoveAll(targetPath)
}

// ── Category cleanup ──────────────────────────────────────────────────────────

// Category represents a cleanable data directory or special cleanup operation.
type Category struct {
	Key       string
	Label     string
	Path      string
	Size      int64
	FileCount int
	Exists    bool
	Special   bool // JSON orphan cleanup / history trim — not a plain RemoveAll
}

var cleanableDirs = []struct {
	key   string
	label string
	dir   string
}{
	{"debug", "Debug logs", "debug"},
	{"file-history", "File history", "file-history"},
	{"telemetry", "Telemetry", "telemetry"},
	{"shell-snapshots", "Shell snapshots", "shell-snapshots"},
	{"transcripts", "Transcripts", "transcripts"},
	{"todos", "Todos", "todos"},
	{"plans", "Plans", "plans"},
	{"usage-data", "Usage data", "usage-data"},
	{"tasks", "Tasks", "tasks"},
	{"paste-cache", "Paste cache", "paste-cache"},
	// Only the disposable plugin cache is cleanable. Keep plugin state such as
	// installed_plugins.json, known_marketplaces.json, data, and marketplaces.
	{"plugins-cache", "Plugins cache", filepath.Join("plugins", "cache")},
}

// dirSizeCount returns total size and file count recursively so the preview
// matches what RemoveAll will actually reclaim.
func dirSizeCount(path string) (size int64, count int) {
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info == nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		count++
		return nil
	})
	return
}

// scanCategories returns cleanable categories inside claudeDir plus special entries.
func scanCategories(claudeDir string) []Category {
	var result []Category

	for _, cat := range cleanableDirs {
		path := filepath.Join(claudeDir, cat.dir)
		_, statErr := os.Stat(path)
		size, count := dirSizeCount(path)
		result = append(result, Category{
			Key:       cat.key,
			Label:     cat.label,
			Path:      path,
			Size:      size,
			FileCount: count,
			Exists:    statErr == nil,
		})
	}

	// Config backups: ~/.claude.json.backup* (sibling of claudeDir)
	parentDir := filepath.Dir(claudeDir)
	backups, _ := filepath.Glob(filepath.Join(parentDir, ".claude.json.backup*"))
	var backupSize int64
	for _, b := range backups {
		if info, err := os.Stat(b); err == nil {
			backupSize += info.Size()
		}
	}
	result = append(result, Category{
		Key:       "config-backups",
		Label:     "Config backups (.claude.json.backup*)",
		Path:      parentDir,
		Size:      backupSize,
		FileCount: len(backups),
		Exists:    len(backups) > 0,
	})

	// Orphan project entries in ~/.claude.json
	claudeJSONPath := filepath.Join(parentDir, ".claude.json")
	orphanCount, jsonExists := countOrphanEntries(claudeJSONPath)
	result = append(result, Category{
		Key:       "json-orphans",
		Label:     "Orphan project entries in ~/.claude.json",
		Path:      claudeJSONPath,
		Size:      0,
		FileCount: orphanCount,
		Exists:    jsonExists && orphanCount > 0,
		Special:   true,
	})

	// History trim: ~/.claude/history.jsonl
	histPath := filepath.Join(claudeDir, "history.jsonl")
	histSize := int64(0)
	histExists := false
	if info, err := os.Stat(histPath); err == nil {
		histSize = info.Size()
		histExists = histSize > 0
	}
	result = append(result, Category{
		Key:     "history-trim",
		Label:   "Trim history.jsonl (keep last 500 lines)",
		Path:    histPath,
		Size:    histSize,
		Exists:  histExists,
		Special: true,
	})

	return result
}

// countOrphanEntries returns the number of project paths in ~/.claude.json
// whose directories no longer exist, plus whether the file was readable.
func countOrphanEntries(claudeJSONPath string) (count int, exists bool) {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return 0, false
	}
	var cfg struct {
		Projects map[string]json.RawMessage `json:"projects"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return 0, true
	}
	for path := range cfg.Projects {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			count++
		}
	}
	return count, true
}

// cleanCategory executes the cleanup operation for a category.
func cleanCategory(cat Category, claudeDir string) error {
	switch cat.Key {
	case "json-orphans":
		parentDir := filepath.Dir(claudeDir)
		return cleanOrphanEntries(filepath.Join(parentDir, ".claude.json"))
	case "history-trim":
		return trimHistory(cat.Path, 500)
	case "config-backups":
		parentDir := filepath.Dir(claudeDir)
		backups, _ := filepath.Glob(filepath.Join(parentDir, ".claude.json.backup*"))
		for _, b := range backups {
			if err := os.Remove(b); err != nil && !os.IsNotExist(err) {
				return err
			}
		}
		return nil
	default:
		if cat.Path == "" || !cat.Exists {
			return nil
		}
		return os.RemoveAll(cat.Path)
	}
}

// cleanOrphanEntries removes project entries from ~/.claude.json where the
// project directory no longer exists.
func cleanOrphanEntries(claudeJSONPath string) error {
	return cleanOrphanEntriesExcept(claudeJSONPath, nil)
}

// cleanOrphanEntriesExcept preserves orphan entries whose normalized project
// paths are protected by Claude Cleaner.
func cleanOrphanEntriesExcept(claudeJSONPath string, protected map[string]bool) error {
	data, err := os.ReadFile(claudeJSONPath)
	if err != nil {
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	projectsRaw, ok := raw["projects"]
	if !ok {
		return nil
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(projectsRaw, &projects); err != nil {
		return err
	}
	changed := false
	for path := range projects {
		if protected != nil && protected[normalizePath(path)] {
			continue
		}
		if _, err := os.Stat(path); os.IsNotExist(err) {
			delete(projects, path)
			changed = true
		}
	}
	if !changed {
		return nil
	}
	newProjects, err := json.Marshal(projects)
	if err != nil {
		return err
	}
	raw["projects"] = newProjects
	newData, err := json.MarshalIndent(raw, "", "  ")
	if err != nil {
		return err
	}
	newData = append(newData, '\n')

	mode := os.FileMode(0644)
	if info, statErr := os.Stat(claudeJSONPath); statErr == nil {
		mode = info.Mode().Perm()
	}
	tmpPath := claudeJSONPath + ".claude-cleaner-orphans.tmp"
	if err := os.WriteFile(tmpPath, newData, mode); err != nil {
		return err
	}
	backup := claudeJSONPath + ".claude-cleaner-orphans.bak"
	_ = os.Remove(backup)
	if err := os.Rename(claudeJSONPath, backup); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	if err := os.Rename(tmpPath, claudeJSONPath); err != nil {
		_ = os.Rename(backup, claudeJSONPath)
		_ = os.Remove(tmpPath)
		return err
	}
	_ = os.Remove(backup)
	return nil
}

// trimHistory keeps only the last keepLines lines of histPath, writing atomically.
func trimHistory(histPath string, keepLines int) error {
	data, err := os.ReadFile(histPath)
	if err != nil {
		return err
	}
	content := strings.TrimRight(string(data), "\n")
	lines := strings.Split(content, "\n")
	if len(lines) <= keepLines {
		return nil
	}
	kept := lines[len(lines)-keepLines:]
	newContent := strings.Join(kept, "\n") + "\n"
	tmpPath := histPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0644); err != nil {
		return err
	}
	return os.Rename(tmpPath, histPath)
}

func formatSize(b int64) string {
	const (
		gb = 1 << 30
		mb = 1 << 20
		kb = 1 << 10
	)
	switch {
	case b >= gb:
		return fmt.Sprintf("%.1f GB", float64(b)/gb)
	case b >= mb:
		return fmt.Sprintf("%.1f MB", float64(b)/mb)
	case b >= kb:
		return fmt.Sprintf("%.1f KB", float64(b)/kb)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatTokens(n int64) string {
	const (
		e = 1_000_000_000_000_000_000 // exa  (int64 max ~9.2E)
		p = 1_000_000_000_000_000     // peta
		t = 1_000_000_000_000         // tera
		b = 1_000_000_000             // billion
		m = 1_000_000                 // million
		k = 1_000                     // kilo
	)
	switch {
	case n >= e:
		return fmt.Sprintf("%.1fE", float64(n)/e)
	case n >= p:
		return fmt.Sprintf("%.1fP", float64(n)/p)
	case n >= t:
		return fmt.Sprintf("%.1fT", float64(n)/t)
	case n >= b:
		return fmt.Sprintf("%.1fB", float64(n)/b)
	case n >= m:
		return fmt.Sprintf("%.1fM", float64(n)/m)
	case n >= k:
		return fmt.Sprintf("%.1fK", float64(n)/k)
	default:
		return fmt.Sprintf("%d", n)
	}
}

func humanTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	d := time.Since(t)
	switch {
	case d < 2*time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%d min ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d hr ago", int(d.Hours()))
	case d < 48*time.Hour:
		return "yesterday"
	case d < 7*24*time.Hour:
		return fmt.Sprintf("%d days ago", int(d.Hours()/24))
	default:
		return t.Format("2006-01-02")
	}
}

func truncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}
