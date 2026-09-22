package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type projectSessionsLoadedMsg struct {
	sessions []ProjectSession
	err      error
}

type sessionDeleteDoneMsg struct {
	deleted  []string
	failed   []string
	sessions []ProjectSession
	dryRun   bool
}

type forgetDoneMsg struct {
	forgotten []string
	failed    []string
	dryRun    bool
}

func projectIdentity(s Session) string {
	if s.ProjectPath != "" {
		return normalizePath(s.ProjectPath)
	}
	return normalizePath(s.Name)
}

func (m model) isProtected(s Session) bool {
	return m.protected[projectIdentity(s)]
}

func (m model) protectedList() []string {
	out := make([]string, 0, len(m.protected))
	for key, protected := range m.protected {
		if protected {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}

func (m model) toggleProtected(s Session) model {
	key := projectIdentity(s)
	if m.protected == nil {
		m.protected = make(map[string]bool)
	}
	if m.protected[key] {
		delete(m.protected, key)
	} else {
		m.protected[key] = true
		delete(m.selected, s.Index)
	}
	m.persistPrefs()
	return m
}

func (m model) openProjectDetail(s Session) (tea.Model, tea.Cmd) {
	m.state = stateProjectDetail
	m.detailProject = s
	m.projectSessions = nil
	m.detailSelected = make(map[int]bool)
	m.detailCursor = 0
	m.detailLoading = true
	m.detailNotice = ""

	projectDir := s.Path
	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sessions, err := scanProjectSessions(projectDir)
			return projectSessionsLoadedMsg{sessions: sessions, err: err}
		},
	)
}

func updateProjectStatsFromSessions(project Session, sessions []ProjectSession) Session {
	project.SessionCount = len(sessions)
	project.Size = 0
	project.TotalTokens = 0
	project.HasTokenData = false
	project.HasData = len(sessions) > 0
	project.Modified = project.Modified.Add(0)
	project.Oldest = project.Oldest.Add(0)

	if len(sessions) == 0 {
		project.Modified = project.Modified.Add(-project.Modified.Sub(project.Modified))
		project.Oldest = project.Oldest.Add(-project.Oldest.Sub(project.Oldest))
		return project
	}

	project.Modified = sessions[0].Modified
	project.Oldest = sessions[0].Modified
	for _, s := range sessions {
		project.Size += s.Size
		if s.HasTokenData {
			project.TotalTokens += s.TotalTokens
			project.HasTokenData = true
		}
		if s.Modified.After(project.Modified) {
			project.Modified = s.Modified
		}
		if s.Modified.Before(project.Oldest) {
			project.Oldest = s.Modified
		}
	}
	return project
}

func (m model) handleProjectDetailKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.projectSessions)
	locked := m.isProtected(m.detailProject)

	switch msg.String() {
	case "up", "k":
		if m.detailCursor > 0 {
			m.detailCursor--
		} else if n > 0 {
			m.detailCursor = n - 1
		}
	case "down", "j":
		if m.detailCursor < n-1 {
			m.detailCursor++
		} else if n > 0 {
			m.detailCursor = 0
		}
	case "g":
		m.detailCursor = 0
	case "G":
		if n > 0 {
			m.detailCursor = n - 1
		}
	case "l", "L":
		m = m.toggleProtected(m.detailProject)
		if m.isProtected(m.detailProject) {
			m.detailSelected = make(map[int]bool)
			m.detailNotice = "Project locked — destructive actions disabled."
		} else {
			m.detailNotice = "Project unlocked."
		}
	case " ":
		if !locked && n > 0 && m.detailCursor < n {
			idx := m.projectSessions[m.detailCursor].Index
			m.detailSelected[idx] = !m.detailSelected[idx]
		}
	case "a":
		if locked {
			m.detailNotice = "Project is locked. Unlock with l first."
			break
		}
		allOn := n > 0
		for _, s := range m.projectSessions {
			if !m.detailSelected[s.Index] {
				allOn = false
				break
			}
		}
		for _, s := range m.projectSessions {
			m.detailSelected[s.Index] = !allOn
		}
	case "n":
		m.detailSelected = make(map[int]bool)
	case "enter":
		if locked {
			m.detailNotice = "Project is locked. Unlock with l first."
			break
		}
		for _, selected := range m.detailSelected {
			if selected {
				m.state = stateSessionConfirm
				m.confirmIdx = 0
				return m, nil
			}
		}
	case "esc":
		m.state = stateList
		m.projectSessions = nil
		m.detailSelected = make(map[int]bool)
		return m.doRescan()
	}
	return m, nil
}

func (m model) handleSessionConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.state = stateProjectDetail
		m.confirmIdx = 0
	case "left", "h", "tab":
		m.confirmIdx = 0
	case "right", "l":
		m.confirmIdx = 1
	case "y":
		m.confirmIdx = 1
		return m.doDeleteProjectSessions()
	case "enter":
		if m.confirmIdx == 1 {
			return m.doDeleteProjectSessions()
		}
		m.state = stateProjectDetail
		m.confirmIdx = 0
	}
	return m, nil
}

func (m model) doDeleteProjectSessions() (tea.Model, tea.Cmd) {
	selected := make(map[int]bool, len(m.detailSelected))
	for k, v := range m.detailSelected {
		selected[k] = v
	}
	sessions := append([]ProjectSession(nil), m.projectSessions...)
	projectDir := m.detailProject.Path
	dryRun := m.dryRun
	m.state = stateDeleting

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var deleted, failed []string
			for _, session := range sessions {
				if !selected[session.Index] {
					continue
				}
				if dryRun {
					deleted = append(deleted, session.ID)
					continue
				}
				if err := safeRemoveSessionFile(projectDir, session.Path); err != nil {
					failed = append(failed, session.ID)
				} else {
					deleted = append(deleted, session.ID)
				}
			}
			refreshed, err := scanProjectSessions(projectDir)
			if err != nil {
				failed = append(failed, "rescan")
				refreshed = sessions
			}
			return sessionDeleteDoneMsg{
				deleted: deleted, failed: failed, sessions: refreshed, dryRun: dryRun,
			}
		},
	)
}

func (m model) prepareForget(sessions []Session) (tea.Model, tea.Cmd) {
	var targets []Session
	for _, s := range sessions {
		if m.selected[s.Index] && !m.isProtected(s) {
			targets = append(targets, s)
		}
	}
	if len(targets) == 0 && m.cursor < len(sessions) {
		s := sessions[m.cursor]
		if m.isProtected(s) {
			return m, nil
		}
		targets = append(targets, s)
	}
	if len(targets) == 0 {
		return m, nil
	}

	m.forgetTargets = targets
	m.state = stateForgetConfirm
	m.confirmIdx = 0
	return m, nil
}

func (m model) handleForgetConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n":
		m.state = stateList
		m.confirmIdx = 0
		m.forgetTargets = nil
	case "left", "h", "tab":
		m.confirmIdx = 0
	case "right", "l":
		m.confirmIdx = 1
	case "y":
		m.confirmIdx = 1
		return m.doForgetProjects()
	case "enter":
		if m.confirmIdx == 1 {
			return m.doForgetProjects()
		}
		m.state = stateList
		m.confirmIdx = 0
		m.forgetTargets = nil
	}
	return m, nil
}

func (m model) doForgetProjects() (tea.Model, tea.Cmd) {
	targets := append([]Session(nil), m.forgetTargets...)
	projectsDir := m.projectsDir
	claudeJSONPath := m.claudeJSONPath
	dryRun := m.dryRun
	m.state = stateDeleting

	return m, tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			var forgotten, failed []string
			for _, s := range targets {
				name := s.Name
				if s.ProjectPath != "" {
					name = filepath.Base(s.ProjectPath)
				}
				if dryRun {
					forgotten = append(forgotten, name)
					continue
				}
				if err := forgetProject(s, projectsDir, claudeJSONPath); err != nil {
					failed = append(failed, name)
				} else {
					forgotten = append(forgotten, name)
				}
			}
			return forgetDoneMsg{forgotten: forgotten, failed: failed, dryRun: dryRun}
		},
	)
}

func (m model) viewProjectDetail() string {
	var sb strings.Builder
	projectName := m.detailProject.Name
	if m.detailProject.ProjectPath != "" {
		projectName = filepath.Base(m.detailProject.ProjectPath)
	}

	lockLabel := ""
	if m.isProtected(m.detailProject) {
		lockLabel = "  " + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("LOCKED")
	}

	sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Render("Project Detail: "+projectName) + lockLabel + "\n")
	if m.detailProject.ProjectPath != "" {
		sb.WriteString("  " + dimStyle.Render(m.detailProject.ProjectPath) + "\n")
	}
	sb.WriteString("  " + dimStyle.Render(strings.Repeat("─", 78)) + "\n")

	if m.detailLoading {
		sb.WriteString("\n  " + m.spinner.View() + " Scanning conversations…\n")
		return sb.String()
	}

	var totalSize, totalTokens int64
	var anyTokens bool
	var oldest, latest string = "—", "—"
	if len(m.projectSessions) > 0 {
		oldestTime := m.projectSessions[0].Modified
		latestTime := m.projectSessions[0].Modified
		for _, s := range m.projectSessions {
			totalSize += s.Size
			if s.HasTokenData {
				totalTokens += s.TotalTokens
				anyTokens = true
			}
			if s.Modified.Before(oldestTime) {
				oldestTime = s.Modified
			}
			if s.Modified.After(latestTime) {
				latestTime = s.Modified
			}
		}
		oldest = oldestTime.Format("2006-01-02")
		latest = latestTime.Format("2006-01-02")
	}
	tok := "—"
	if anyTokens {
		tok = formatTokens(totalTokens)
	}
	sb.WriteString(fmt.Sprintf("  Sessions: %d  •  Size: %s  •  Tokens: %s  •  Oldest: %s  •  Latest: %s\n\n",
		len(m.projectSessions), formatSize(totalSize), tok, oldest, latest))

	if m.detailNotice != "" {
		sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrCyan).Render(m.detailNotice) + "\n\n")
	}

	if len(m.projectSessions) == 0 {
		sb.WriteString("  " + dimStyle.Render("No conversation JSONL files found.") + "\n")
		sb.WriteString(helpStyle.Render("esc back  l lock/unlock"))
		return sb.String()
	}

	const (
		idW = 30
		timeW = 16
		msgW = 9
		tokW = 10
	)
	sb.WriteString(dimStyle.Render(fmt.Sprintf("        %-*s  %-*s  %-*s  %-*s  %s",
		idW, "Session", timeW, "Modified", msgW, "Messages", tokW, "Tokens", "Size")) + "\n")
	sb.WriteString(dimStyle.Render("  " + strings.Repeat("─", 88)) + "\n")

	rowW := m.width
	if rowW < 100 {
		rowW = 100
	}
	for i, session := range m.projectSessions {
		isCursor := m.detailCursor == i
		isSelected := m.detailSelected[session.Index]
		bg := clrBg
		rowStyle := rowNormalStyle
		if isCursor {
			bg = clrCursor
			rowStyle = rowCursorStyle
		} else if isSelected {
			bg = clrSelection
			rowStyle = rowSelectedStyle
		}
		cur := lipgloss.NewStyle().Background(bg).Render("  ")
		if isCursor {
			cur = lipgloss.NewStyle().Foreground(clrPurple).Background(bg).Bold(true).Render("▶ ")
		}
		check := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Render("[ ]")
		if isSelected {
			check = lipgloss.NewStyle().Foreground(clrGreen).Background(bg).Bold(true).Render("[✓]")
		}
		id := lipgloss.NewStyle().Foreground(clrFg).Background(bg).Width(idW).Render(truncate(session.ID, idW))
		mod := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Width(timeW).Render(session.Modified.Format("2006-01-02 15:04"))
		msgs := lipgloss.NewStyle().Foreground(clrComment).Background(bg).Width(msgW).Render(fmt.Sprintf("%d", session.MessageCount))
		tokens := "—"
		if session.HasTokenData {
			tokens = formatTokens(session.TotalTokens)
		}
		tokCell := lipgloss.NewStyle().Foreground(clrPurple).Background(bg).Width(tokW).Render(tokens)
		size := lipgloss.NewStyle().Foreground(clrCyan).Background(bg).Render(formatSize(session.Size))
		sb.WriteString(rowStyle.Width(rowW).Render(cur+check+" "+id+"  "+mod+"  "+msgs+"  "+tokCell+"  "+size) + "\n")
	}

	footer := "↑/↓ navigate  space select  a all  n none  enter delete selected  l lock/unlock  esc back"
	if m.isProtected(m.detailProject) {
		footer = "↑/↓ navigate  l unlock project  esc back    destructive actions are disabled while locked"
	}
	sb.WriteString(helpStyle.Render(footer))
	return sb.String()
}

func (m model) viewSessionConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n  " + dangerStyle.Render("⚠  Delete selected conversation sessions?") + "\n\n")
	for _, session := range m.projectSessions {
		if !m.detailSelected[session.Index] {
			continue
		}
		tokens := "—"
		if session.HasTokenData {
			tokens = formatTokens(session.TotalTokens) + " tok"
		}
		sb.WriteString(fmt.Sprintf("    %s  %-32s  %-12s  %s\n",
			checkOnStyle.Render("✓"), truncate(session.ID, 32), tokens, formatSize(session.Size)))
	}
	sb.WriteString("\n  " + dimStyle.Render("Only selected JSONL conversation files will be deleted. Source code is NOT affected.") + "\n\n")
	no := dimStyle.Render("[ N ]  No, cancel")
	yes := dimStyle.Render("[ Y ]  Yes, delete")
	if m.confirmIdx == 0 {
		no = lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render("[ N ]  No, cancel")
	} else {
		yes = lipgloss.NewStyle().Foreground(clrRed).Bold(true).Render("[ Y ]  Yes, delete")
	}
	sb.WriteString("  " + no + "      " + yes + "\n")
	return sb.String()
}

func (m model) viewForgetConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n  " + dangerStyle.Render("⚠  Forget Claude project metadata?") + "\n\n")
	for _, project := range m.forgetTargets {
		name := project.Name
		if project.ProjectPath != "" {
			name = filepath.Base(project.ProjectPath)
		}
		sb.WriteString("    " + checkOnStyle.Render("✓") + "  " + name + "\n")
		if project.ProjectPath != "" {
			sb.WriteString("       " + dimStyle.Render(project.ProjectPath) + "\n")
		}
	}
	sb.WriteString("\n  Removes:\n")
	sb.WriteString("    • Claude session history under ~/.claude/projects\n")
	sb.WriteString("    • Matching project entries from ~/.claude.json\n")
	sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(clrGreen).Bold(true).Render("Source project files will NOT be deleted.") + "\n\n")

	no := dimStyle.Render("[ N ]  No, cancel")
	yes := dimStyle.Render("[ Y ]  Yes, forget")
	if m.confirmIdx == 0 {
		no = lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render("[ N ]  No, cancel")
	} else {
		yes = lipgloss.NewStyle().Foreground(clrRed).Bold(true).Render("[ Y ]  Yes, forget")
	}
	sb.WriteString("  " + no + "      " + yes + "\n")
	return sb.String()
}
