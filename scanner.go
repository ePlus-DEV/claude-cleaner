package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type Session struct {
	Index    int
	Name     string
	Path     string
	Modified time.Time
	Size     int64
}

func scanSessions(projectsDir string) ([]Session, error) {
	entries, err := os.ReadDir(projectsDir)
	if err != nil {
		return nil, err
	}

	type result struct {
		s  Session
		ok bool
	}

	results := make([]result, len(entries))
	var wg sync.WaitGroup

	for i, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		wg.Add(1)
		go func(idx int, e fs.DirEntry) {
			defer wg.Done()
			fullPath := filepath.Join(projectsDir, e.Name())
			info, err := e.Info()
			if err != nil {
				return
			}
			size, _ := dirSize(fullPath)
			results[idx] = result{
				s: Session{
					Name:     e.Name(),
					Path:     fullPath,
					Modified: info.ModTime(),
					Size:     size,
				},
				ok: true,
			}
		}(i, entry)
	}

	wg.Wait()

	var sessions []Session
	for _, r := range results {
		if r.ok {
			sessions = append(sessions, r.s)
		}
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].Modified.After(sessions[j].Modified)
	})

	for i := range sessions {
		sessions[i].Index = i + 1
	}

	return sessions, nil
}

func dirSize(root string) (int64, error) {
	var total int64
	err := filepath.WalkDir(root, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total, err
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

func humanTime(t time.Time) string {
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
