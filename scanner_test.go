package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDirSize(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "nested")
	if err := os.Mkdir(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "one.txt"), []byte("12345"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(nested, "two.txt"), []byte("1234567"), 0644); err != nil {
		t.Fatal(err)
	}

	size, err := dirSize(root)
	if err != nil {
		t.Fatal(err)
	}
	if size != 12 {
		t.Errorf("expected 12, got %d", size)
	}
}

func TestSafeRemove(t *testing.T) {
	root := t.TempDir()

	// Direct child — should succeed
	child := filepath.Join(root, "session-1")
	if err := os.Mkdir(child, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, child); err != nil {
		t.Errorf("expected no error for direct child, got: %v", err)
	}

	// Same path as projectsDir — refuse
	if err := safeRemove(root, root); err == nil {
		t.Error("expected error when target equals projectsDir")
	}

	// Parent directory — refuse
	if err := safeRemove(root, filepath.Dir(root)); err == nil {
		t.Error("expected error for path above projectsDir")
	}

	// Nested (grandchild) path — refuse
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0755); err != nil {
		t.Fatal(err)
	}
	if err := safeRemove(root, nested); err == nil {
		t.Error("expected error for nested path")
	}
}

func TestScanSessions(t *testing.T) {
	root := t.TempDir()
	names := []string{"proj-a", "proj-b", "proj-c"}
	for _, name := range names {
		dir := filepath.Join(root, name)
		if err := os.Mkdir(dir, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "session.jsonl"), []byte("{}"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	sessions, err := scanSessions(root)
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
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Errorf("got %q, want 'hello'", got)
	}
	if got := truncate("hello world", 8); got != "hello w…" {
		t.Errorf("got %q, want 'hello w…'", got)
	}
}
