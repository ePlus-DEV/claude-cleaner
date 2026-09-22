package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestProjectStats(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "one.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	size, mod := projectStats(dir)
	if size == 0 {
		t.Error("size should be > 0")
	}
	if mod.IsZero() {
		t.Error("modified should not be zero")
	}
}

func TestProjectEntryTotal(t *testing.T) {
	i := int64(100)
	o := int64(50)
	cc := int64(20)
	cr := int64(10)
	e := projectEntry{
		LastTotalInputTokens:              &i,
		LastTotalOutputTokens:             &o,
		LastTotalCacheCreationInputTokens: &cc,
		LastTotalCacheReadInputTokens:     &cr,
	}
	if got := e.total(); got != 180 {
		t.Errorf("total want 180, got %d", got)
	}
	if !e.hasAnyField() {
		t.Error("hasAnyField should be true")
	}

	empty := projectEntry{}
	if empty.hasAnyField() {
		t.Error("empty entry hasAnyField should be false")
	}
	if empty.total() != 0 {
		t.Error("empty entry total should be 0")
	}
}

func TestDeduplicateProjects(t *testing.T) {
	i1 := int64(1000)
	i2 := int64(500)
	raw := map[string]projectEntry{
		"d:/laragon/www/g-front": {LastTotalInputTokens: &i1},
		"D:/laragon/www/g-front": {LastTotalInputTokens: &i2}, // duplicate, lower tokens
		"/home/user/app":         {},
	}
	deduped := deduplicateProjects(raw)

	if len(deduped) != 2 {
		t.Errorf("expected 2 after dedup, got %d", len(deduped))
	}
	// The higher-token entry (i1=1000) should win
	for path, entry := range deduped {
		norm := normalizePath(path)
		if norm == "d:/laragon/www/g-front" {
			if entry.total() != 1000 {
				t.Errorf("expected winning entry to have total 1000, got %d", entry.total())
			}
		}
	}
}

func TestNormalizePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"d:/laragon/www/g-front", "d:/laragon/www/g-front"},
		{"D:/laragon/www/g-front", "d:/laragon/www/g-front"},
		{"D:\\laragon\\www\\g-front", "d:/laragon/www/g-front"},
	}
	for _, c := range cases {
		if got := normalizePath(c.in); got != c.want {
			t.Errorf("normalizePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestSafeRemove(t *testing.T) {
	root := t.TempDir()

	child := filepath.Join(root, "session-1")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, child); err != nil {
		t.Errorf("expected no error for direct child, got: %v", err)
	}

	if err := safeRemove(root, root); err == nil {
		t.Error("expected error when target equals projectsDir")
	}

	if err := safeRemove(root, filepath.Dir(root)); err == nil {
		t.Error("expected error for path above projectsDir")
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, nested); err == nil {
		t.Error("expected error for nested path")
	}
}

func TestScanSessionsFromDir(t *testing.T) {
	projectsDir := t.TempDir()
	names := []string{"proj-a", "proj-b", "proj-c"}
	// Each session has real usage data so we can verify token reading.
	usageLine := `{"type":"assistant","message":{"usage":{"input_tokens":1000,"output_tokens":500,"cache_creation_input_tokens":200,"cache_read_input_tokens":300}}}` + "\n"
	for _, name := range names {
		dir := filepath.Join(projectsDir, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(usageLine), 0644); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := scanSessions("/nonexistent/.claude.json", projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != len(names) {
		t.Errorf("expected %d sessions, got %d", len(names), len(sessions))
	}
	for i, s := range sessions {
		if s.Index != i+1 {
			t.Errorf("session[%d].Index = %d, want %d", i, s.Index, i+1)
		}
		if s.Size == 0 {
			t.Errorf("session[%d].Size should be > 0", i)
		}
		if !s.HasTokenData {
			t.Errorf("session[%d] should have token data from .jsonl", i)
		}
		// 1000 + 500 + 200 + 300 = 2000
		if s.TotalTokens != 2000 {
			t.Errorf("session[%d] tokens want 2000, got %d", i, s.TotalTokens)
		}
	}
}

func TestScanSessionsFromClaudeJSON(t *testing.T) {
	projectsDir := t.TempDir()

	i := int64(108600)
	o := int64(50000)

	projects := map[string]string{
		"/home/user/my-app":     encodePath("/home/user/my-app"),
		"/home/user/other-proj": encodePath("/home/user/other-proj"),
	}
	for _, encoded := range projects {
		dir := filepath.Join(projectsDir, encoded)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	claudeJSON := `{"projects":{"/home/user/my-app":{"lastTotalInputTokens":108600,"lastTotalOutputTokens":50000},"/home/user/other-proj":{}}}`
	jsonPath := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(jsonPath, []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanSessions(jsonPath, projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions, got %d", len(sessions))
	}
	for _, s := range sessions {
		if s.ProjectPath == "" {
			t.Errorf("session %q should have ProjectPath set", s.Name)
		}
		if s.ProjectPath == "/home/user/my-app" {
			if !s.HasTokenData {
				t.Error("my-app should have token data")
			}
			want := i + o
			if s.TotalTokens != want {
				t.Errorf("my-app total tokens want %d, got %d", want, s.TotalTokens)
			}
		}
		if s.ProjectPath == "/home/user/other-proj" {
			if s.HasTokenData {
				t.Error("other-proj should not have token data (no fields in JSON and no usage in .jsonl)")
			}
		}
	}
}

func TestScanProjectTokens(t *testing.T) {
	dir := t.TempDir()
	// Two assistant messages with usage; one user message (ignored); one assistant with no usage (ignored).
	lines := "" +
		`{"type":"user","message":{"content":"hello"}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":200,"cache_read_input_tokens":30}}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":10,"output_tokens":5,"cache_creation_input_tokens":0,"cache_read_input_tokens":0}}}` + "\n" +
		`{"type":"assistant","message":{}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(lines), 0644); err != nil {
		t.Fatal(err)
	}

	total, hasData := scanProjectTokens(dir)
	if !hasData {
		t.Error("hasData should be true")
	}
	// (100+50+200+30) + (10+5+0+0) = 395
	want := int64(395)
	if total != want {
		t.Errorf("total want %d, got %d", want, total)
	}
}

func TestScanProjectTokensLargeJSONLLine(t *testing.T) {
	dir := t.TempDir()
	line := `{"type":"assistant","padding":"` +
		strings.Repeat("x", 2*1024*1024) +
		`","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":20,"cache_read_input_tokens":10}}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "large.jsonl"), []byte(line), 0644); err != nil {
		t.Fatal(err)
	}

	total, hasData := scanProjectTokens(dir)
	if !hasData {
		t.Fatal("large JSONL line should still produce token data")
	}
	if total != 180 {
		t.Fatalf("large JSONL line tokens want 180, got %d", total)
	}
}

func TestScanProjectTokensEmptyDir(t *testing.T) {
	dir := t.TempDir()
	total, hasData := scanProjectTokens(dir)
	if hasData {
		t.Error("empty dir: hasData should be false")
	}
	if total != 0 {
		t.Errorf("empty dir: total should be 0, got %d", total)
	}
}

func TestScanProjectTokensNoUsage(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "s.jsonl"), []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	total, hasData := scanProjectTokens(dir)
	if hasData {
		t.Error("no usage lines: hasData should be false")
	}
	if total != 0 {
		t.Errorf("total should be 0, got %d", total)
	}
}

func TestScanSessionsClaudeJSONFallsBackToJSONL(t *testing.T) {
	projectsDir := t.TempDir()

	// Project exists in claude.json but has no token fields — .jsonl has usage.
	projPath := "/home/user/no-fields-proj"
	encoded := encodePath(projPath)
	dir := filepath.Join(projectsDir, encoded)
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	jsonl := `{"type":"assistant","message":{"usage":{"input_tokens":500,"output_tokens":250,"cache_creation_input_tokens":100,"cache_read_input_tokens":50}}}` + "\n"
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte(jsonl), 0644); err != nil {
		t.Fatal(err)
	}

	claudeJSON := `{"projects":{"/home/user/no-fields-proj":{}}}`
	jsonPath := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(jsonPath, []byte(claudeJSON), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanSessions(jsonPath, projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	s := sessions[0]
	if !s.HasTokenData {
		t.Error("should have token data from .jsonl fallback")
	}
	// 500 + 250 + 100 + 50 = 900
	if s.TotalTokens != 900 {
		t.Errorf("total tokens want 900, got %d", s.TotalTokens)
	}
}

func TestFormatTokens(t *testing.T) {
	cases := []struct {
		n    int64
		want string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1.0K"},
		{108600, "108.6K"},
		{1_000_000, "1.0M"},
		{10_700_000, "10.7M"},
		{1_000_000_000, "1.0B"},
		{2_500_000_000, "2.5B"},
		{1_000_000_000_000, "1.0T"},
		{1_000_000_000_000_000, "1.0P"},
		{1_000_000_000_000_000_000, "1.0E"},
	}
	for _, c := range cases {
		if got := formatTokens(c.n); got != c.want {
			t.Errorf("formatTokens(%d) = %q, want %q", c.n, got, c.want)
		}
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
	}
	for _, tt := range tests {
		if got := formatSize(tt.bytes); got != tt.want {
			t.Errorf("formatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestHumanTime(t *testing.T) {
	if got := humanTime(time.Now().Add(-30 * time.Second)); got != "just now" {
		t.Errorf("got %q, want 'just now'", got)
	}
	if got := humanTime(time.Now().Add(-25 * time.Hour)); got != "yesterday" {
		t.Errorf("got %q, want 'yesterday'", got)
	}
	if got := humanTime(time.Time{}); got != "—" {
		t.Errorf("zero time want '—', got %q", got)
	}
}

func TestEncodePath(t *testing.T) {
	cases := []struct{ in, want string }{
		{"d:/laragon/www/dev/claude-session-cleaner", "d--laragon-www-dev-claude-session-cleaner"},
		{"/Users/foo/myproject", "-users-foo-myproject"},
		{"D:\\laragon\\www\\myapp", "d--laragon-www-myapp"},
	}
	for _, c := range cases {
		if got := encodePath(c.in); got != c.want {
			t.Errorf("encodePath(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
	if got := truncate("hello world", 8); got != "hello w…" {
		t.Errorf("got %q, want 'hello w…'", got)
	}
}

// --- malformed / empty JSON ---

func TestScanSessionsMalformedJSONFallsBack(t *testing.T) {
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "proj-x")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	jsonPath := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(jsonPath, []byte("{not valid json"), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanSessions(jsonPath, projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("malformed JSON should fallback to dir scan, got %d sessions", len(sessions))
	}
}

func TestScanSessionsEmptyProjectsFallsBack(t *testing.T) {
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "proj-y")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "f.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	jsonPath := filepath.Join(t.TempDir(), ".claude.json")
	if err := os.WriteFile(jsonPath, []byte(`{"projects":{}}`), 0644); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanSessions(jsonPath, projectsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Errorf("empty projects map should fallback to dir scan, got %d sessions", len(sessions))
	}
}

// --- projectStats skips subdirectories ---

func TestProjectStatsSkipsSubdirs(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "subdir", "nested.jsonl"), []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	// No files at top level — only subdir
	size, mod := projectStats(dir)
	if size != 0 {
		t.Errorf("projectStats should skip subdirs, got size %d", size)
	}
	if !mod.IsZero() {
		t.Error("projectStats should skip subdirs, got non-zero mtime")
	}
}

func TestDirSizeCountRecursive(t *testing.T) {
	dir := t.TempDir()
	nested := filepath.Join(dir, "nested", "deeper")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "top.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "child.txt"), []byte("1234567"), 0644); err != nil {
		t.Fatal(err)
	}

	size, count := dirSizeCount(dir)
	if size != 12 {
		t.Fatalf("recursive size want 12, got %d", size)
	}
	if count != 2 {
		t.Fatalf("recursive count want 2, got %d", count)
	}
}

func TestScanCategoriesTargetsOnlyPluginCache(t *testing.T) {
	claudeDir := t.TempDir()
	cacheDir := filepath.Join(claudeDir, "plugins", "cache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheDir, "cached.bin"), []byte("cache"), 0644); err != nil {
		t.Fatal(err)
	}
	statePath := filepath.Join(claudeDir, "plugins", "installed_plugins.json")
	if err := os.WriteFile(statePath, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	var pluginCat *Category
	cats := scanCategories(claudeDir)
	for i := range cats {
		if cats[i].Key == "plugins-cache" {
			pluginCat = &cats[i]
			break
		}
	}
	if pluginCat == nil {
		t.Fatal("plugins-cache category not found")
	}
	if pluginCat.Path != cacheDir {
		t.Fatalf("plugin cleanup path want %q, got %q", cacheDir, pluginCat.Path)
	}
	if err := cleanCategory(*pluginCat, claudeDir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(cacheDir); !os.IsNotExist(err) {
		t.Fatal("plugin cache should be removed")
	}
	if _, err := os.Stat(statePath); err != nil {
		t.Fatalf("plugin state must be preserved: %v", err)
	}
}

func TestProjectStatsEmptyDir(t *testing.T) {
	dir := t.TempDir()
	size, mod := projectStats(dir)
	if size != 0 {
		t.Errorf("empty dir size want 0, got %d", size)
	}
	if !mod.IsZero() {
		t.Error("empty dir mtime should be zero")
	}
}

func TestProjectStatsNonexistentDir(t *testing.T) {
	size, mod := projectStats("/nonexistent/path/xyz")
	if size != 0 || !mod.IsZero() {
		t.Error("nonexistent dir should return zero size and zero mtime")
	}
}

// --- RunPurge partial vs all ---

func TestRunPurgePartialSelection(t *testing.T) {
	projectsDir := t.TempDir()

	makeSession := func(idx int, name string) Session {
		dir := filepath.Join(projectsDir, name)
		_ = os.Mkdir(dir, 0755)
		_ = os.WriteFile(filepath.Join(dir, "f.jsonl"), []byte("x"), 0644)
		return Session{Index: idx, Name: name, Path: dir}
	}

	sessions := []Session{
		makeSession(1, "proj-a"),
		makeSession(2, "proj-b"),
		makeSession(3, "proj-c"),
	}
	// Only select proj-a and proj-c
	selected := map[int]bool{1: true, 3: true}

	deleted, failed := RunPurge(sessions, selected, projectsDir)

	if len(failed) != 0 {
		t.Errorf("no failures expected, got %v", failed)
	}
	if len(deleted) != 2 {
		t.Errorf("want 2 deleted, got %d: %v", len(deleted), deleted)
	}
	// proj-b should still exist
	if _, err := os.Stat(filepath.Join(projectsDir, "proj-b")); os.IsNotExist(err) {
		t.Error("proj-b should NOT be deleted (not selected)")
	}
	// proj-a and proj-c should be gone
	if _, err := os.Stat(filepath.Join(projectsDir, "proj-a")); !os.IsNotExist(err) {
		t.Error("proj-a should be deleted")
	}
}

// --- purgeProject with no ProjectPath ---

func TestPurgeProjectNoProjectPath(t *testing.T) {
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "orphan")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	s := Session{Index: 1, Name: "orphan", Path: dir, ProjectPath: ""}
	if err := purgeProject(s, projectsDir); err != nil {
		t.Errorf("purgeProject with no ProjectPath should still remove dir: %v", err)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Error("dir should be removed")
	}
}

func TestPurgeProjectAlreadyGone(t *testing.T) {
	projectsDir := t.TempDir()
	s := Session{Index: 1, Name: "gone", Path: filepath.Join(projectsDir, "gone"), ProjectPath: ""}
	if err := purgeProject(s, projectsDir); err != nil {
		t.Errorf("purgeProject on nonexistent path should succeed: %v", err)
	}
}


func TestScanProjectSessions(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "aaa.jsonl")
	second := filepath.Join(dir, "bbb.jsonl")

	firstData := "" +
		`{"type":"user","message":{"content":"hello"}}` + "\n" +
		`{"type":"assistant","message":{"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":20,"cache_read_input_tokens":10}}}` + "\n"
	secondData := `{"type":"assistant","message":{"usage":{"input_tokens":10,"output_tokens":5}}}` + "\n"

	if err := os.WriteFile(first, []byte(firstData), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(second, []byte(secondData), 0644); err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	if err := os.Chtimes(first, now.Add(-time.Hour), now.Add(-time.Hour)); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(second, now, now); err != nil {
		t.Fatal(err)
	}

	sessions, err := scanProjectSessions(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 2 {
		t.Fatalf("want 2 sessions, got %d", len(sessions))
	}
	if sessions[0].ID != "bbb" {
		t.Fatalf("newest session should be first, got %q", sessions[0].ID)
	}
	if sessions[1].TotalTokens != 180 {
		t.Fatalf("aaa tokens want 180, got %d", sessions[1].TotalTokens)
	}
	if sessions[1].MessageCount != 2 {
		t.Fatalf("aaa messages want 2, got %d", sessions[1].MessageCount)
	}
}

func TestSafeRemoveSessionFile(t *testing.T) {
	projectDir := t.TempDir()
	direct := filepath.Join(projectDir, "session.jsonl")
	if err := os.WriteFile(direct, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := safeRemoveSessionFile(projectDir, direct); err != nil {
		t.Fatalf("direct JSONL should be removable: %v", err)
	}
	if _, err := os.Stat(direct); !os.IsNotExist(err) {
		t.Fatal("direct JSONL should be gone")
	}

	nested := filepath.Join(projectDir, "nested", "session.jsonl")
	if err := os.MkdirAll(filepath.Dir(nested), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nested, []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := safeRemoveSessionFile(projectDir, nested); err == nil {
		t.Fatal("nested session path should be rejected")
	}

	notJSONL := filepath.Join(projectDir, "notes.txt")
	if err := os.WriteFile(notJSONL, []byte("x"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := safeRemoveSessionFile(projectDir, notJSONL); err == nil {
		t.Fatal("non-JSONL file should be rejected")
	}
}

func TestForgetProjectRemovesClaudeDataOnly(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, ".claude", "projects")
	if err := os.MkdirAll(projectsDir, 0755); err != nil {
		t.Fatal(err)
	}

	sourceDir := filepath.Join(root, "source", "my-project")
	if err := os.MkdirAll(sourceDir, 0755); err != nil {
		t.Fatal(err)
	}
	sourceFile := filepath.Join(sourceDir, "keep.txt")
	if err := os.WriteFile(sourceFile, []byte("keep"), 0644); err != nil {
		t.Fatal(err)
	}

	sessionDir := filepath.Join(projectsDir, encodePath(sourceDir))
	if err := os.MkdirAll(sessionDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sessionDir, "session.jsonl"), []byte("{}"), 0644); err != nil {
		t.Fatal(err)
	}

	otherProject := filepath.Join(root, "source", "other")
	config := map[string]any{
		"theme": "dark",
		"projects": map[string]any{
			sourceDir:    map[string]any{"lastTotalInputTokens": 1},
			otherProject: map[string]any{},
		},
	}
	data, _ := json.Marshal(config)
	jsonPath := filepath.Join(root, ".claude.json")
	if err := os.WriteFile(jsonPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	project := Session{
		Name:        encodePath(sourceDir),
		Path:        sessionDir,
		ProjectPath: sourceDir,
	}
	if err := forgetProject(project, projectsDir, jsonPath); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(sessionDir); !os.IsNotExist(err) {
		t.Fatal("Claude session directory should be removed")
	}
	if _, err := os.Stat(sourceFile); err != nil {
		t.Fatalf("source project must remain untouched: %v", err)
	}

	updated, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatal(err)
	}
	var rootJSON map[string]json.RawMessage
	if err := json.Unmarshal(updated, &rootJSON); err != nil {
		t.Fatal(err)
	}
	var projects map[string]json.RawMessage
	if err := json.Unmarshal(rootJSON["projects"], &projects); err != nil {
		t.Fatal(err)
	}
	if _, ok := projects[sourceDir]; ok {
		t.Fatal("forgotten project entry should be removed")
	}
	if _, ok := projects[otherProject]; !ok {
		t.Fatal("unrelated project entry should be preserved")
	}
	if _, ok := rootJSON["theme"]; !ok {
		t.Fatal("unrelated root config should be preserved")
	}
}
