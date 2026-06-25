package main

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Dracula palette
var (
	clrPurple  = lipgloss.Color("#BD93F9")
	clrCyan    = lipgloss.Color("#8BE9FD")
	clrGreen   = lipgloss.Color("#50FA7B")
	clrRed     = lipgloss.Color("#FF5555")
	clrFg      = lipgloss.Color("#F8F8F2")
	clrComment = lipgloss.Color("#6272A4")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(clrPurple).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(clrPurple).
			Padding(0, 2)

	dimStyle      = lipgloss.NewStyle().Foreground(clrComment)
	nameStyle     = lipgloss.NewStyle().Foreground(clrFg)
	timeStyle     = lipgloss.NewStyle().Foreground(clrComment)
	sizeStyle     = lipgloss.NewStyle().Foreground(clrCyan)
	checkOnStyle  = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	checkOffStyle = lipgloss.NewStyle().Foreground(clrComment)
	cursorStyle   = lipgloss.NewStyle().Foreground(clrPurple).Bold(true)
	dangerStyle   = lipgloss.NewStyle().Foreground(clrRed).Bold(true)
	successStyle  = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)
	countStyle    = lipgloss.NewStyle().Foreground(clrGreen).Bold(true)

	helpStyle = lipgloss.NewStyle().
			Foreground(clrComment).
			BorderTop(true).
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(clrComment).
			MarginTop(1).
			PaddingTop(1)
)

type appState int

const (
	stateLoading appState = iota
	stateList
	stateConfirm
	stateDeleting
	stateDone
)

type sessionsLoadedMsg struct {
	sessions []Session
	err      error
}

type deleteDoneMsg struct {
	deleted []string
	failed  []string
}

type model struct {
	state       appState
	claudeDir   string
	projectsDir string
	sessions    []Session
	selected    map[int]bool
	cursor      int
	spinner     spinner.Model
	input       textinput.Model
	deleted     []string
	failed      []string
	width       int
}

func newModel(claudeDir, projectsDir string) model {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = lipgloss.NewStyle().Foreground(clrPurple)

	ti := textinput.New()
	ti.Placeholder = "DELETE"
	ti.CharLimit = 10
	ti.Width = 20

	return model{
		state:       stateLoading,
		claudeDir:   claudeDir,
		projectsDir: projectsDir,
		selected:    make(map[int]bool),
		spinner:     sp,
		input:       ti,
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		func() tea.Msg {
			sessions, err := scanSessions(m.projectsDir)
			return sessionsLoadedMsg{sessions, err}
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		return m, nil

	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.state == stateConfirm {
			return m.handleConfirmKey(msg)
		}
		return m.handleListKey(msg)

	case sessionsLoadedMsg:
		if msg.err != nil {
			return m, tea.Quit
		}
		m.sessions = msg.sessions
		m.state = stateList
		return m, nil

	case deleteDoneMsg:
		m.deleted = msg.deleted
		m.failed = msg.failed
		m.state = stateDone
		return m, nil

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Forward non-key messages to textinput while in confirm state
	if m.state == stateConfirm {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}

	return m, nil
}

func (m model) handleListKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.sessions)

	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
		}

	case "down", "j":
		if m.cursor < n-1 {
			m.cursor++
		}

	case " ":
		if n > 0 {
			idx := m.sessions[m.cursor].Index
			m.selected[idx] = !m.selected[idx]
		}

	case "a":
		anyOn := false
		for _, v := range m.selected {
			if v {
				anyOn = true
				break
			}
		}
		for _, s := range m.sessions {
			m.selected[s.Index] = !anyOn
		}

	case "enter":
		count := 0
		for _, v := range m.selected {
			if v {
				count++
			}
		}
		if count > 0 {
			m.state = stateConfirm
			m.input.Focus()
			return m, textinput.Blink
		}
	}

	return m, nil
}

func (m model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		m.state = stateList
		m.input.SetValue("")
		m.input.Blur()
		return m, nil

	case tea.KeyEnter:
		if m.input.Value() == "DELETE" {
			m.state = stateDeleting
			m.input.SetValue("")
			return m, tea.Batch(
				m.spinner.Tick,
				func() tea.Msg {
					var deleted, failed []string
					for _, s := range m.sessions {
						if !m.selected[s.Index] {
							continue
						}
						if err := safeRemove(m.projectsDir, s.Path); err != nil {
							failed = append(failed, s.Name)
						} else {
							deleted = append(deleted, s.Name)
						}
					}
					return deleteDoneMsg{deleted, failed}
				},
			)
		}
		m.input.SetValue("")
		return m, nil
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m model) View() string {
	header := titleStyle.Render("  Claude Session Cleaner  ·  ePlus.DEV  ") + "\n" +
		dimStyle.Render("  "+m.claudeDir) + "\n"

	var body string
	switch m.state {
	case stateLoading:
		body = "\n  " + m.spinner.View() + " Scanning sessions…\n"
	case stateList:
		body = m.viewList()
	case stateConfirm:
		body = m.viewConfirm()
	case stateDeleting:
		body = "\n  " + m.spinner.View() + " Deleting…\n"
	case stateDone:
		body = m.viewDone()
	}

	return header + body
}

func (m model) viewList() string {
	if len(m.sessions) == 0 {
		return "\n  " + dimStyle.Render("No Claude project sessions found.") + "\n" +
			"\n  " + dimStyle.Render("q quit")
	}

	const (
		nameW = 44
		timeW = 14
		sizeW = 10
	)

	var sb strings.Builder
	sb.WriteString("\n")

	// Header row
	sb.WriteString(dimStyle.Render(fmt.Sprintf("       %-*s  %-*s  %s",
		nameW, "Name",
		timeW, "Last modified",
		"Size",
	)) + "\n")
	sb.WriteString(dimStyle.Render("  "+strings.Repeat("─", nameW+timeW+sizeW+12)) + "\n")

	for _, s := range m.sessions {
		cur := "  "
		if m.cursor == s.Index-1 {
			cur = cursorStyle.Render("▶ ")
		}

		check := checkOffStyle.Render("[ ]")
		if m.selected[s.Index] {
			check = checkOnStyle.Render("[✓]")
		}

		name := nameStyle.Width(nameW).Render(truncate(s.Name, nameW))
		t := timeStyle.Width(timeW).Render(humanTime(s.Modified))
		sz := sizeStyle.Render(formatSize(s.Size))

		sb.WriteString(cur + check + "  " + name + "  " + t + "  " + sz + "\n")
	}

	selected := 0
	for _, v := range m.selected {
		if v {
			selected++
		}
	}

	footer := fmt.Sprintf(
		"↑/↓ navigate  space toggle  a select all  enter confirm  q quit    %s selected",
		countStyle.Render(fmt.Sprintf("%d", selected)),
	)
	sb.WriteString(helpStyle.Render(footer))

	return sb.String()
}

func (m model) viewConfirm() string {
	var sb strings.Builder
	sb.WriteString("\n")
	sb.WriteString("  " + dangerStyle.Render("⚠  Will delete:") + "\n\n")

	var total int64
	for _, s := range m.sessions {
		if !m.selected[s.Index] {
			continue
		}
		total += s.Size
		sb.WriteString(fmt.Sprintf("    %s  %s  %s\n",
			checkOnStyle.Render("✓"),
			nameStyle.Render(truncate(s.Name, 44)),
			sizeStyle.Render(formatSize(s.Size)),
		))
	}

	sb.WriteString(fmt.Sprintf("\n  Total: %s\n\n", sizeStyle.Render(formatSize(total))))
	sb.WriteString("  " + dimStyle.Render("Deletes session history only. Source code is NOT affected.") + "\n\n")
	sb.WriteString("  " + m.input.View() + "\n\n")
	sb.WriteString("  " + dimStyle.Render("enter confirm  esc back"))

	return sb.String()
}

func (m model) viewDone() string {
	var sb strings.Builder
	sb.WriteString("\n")

	for _, name := range m.deleted {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", successStyle.Render("✓"), name))
	}
	for _, name := range m.failed {
		sb.WriteString(fmt.Sprintf("  %s  %s\n", dangerStyle.Render("✗"), name))
	}

	if len(m.deleted) > 0 {
		sb.WriteString(fmt.Sprintf("\n  %s\n",
			successStyle.Render(fmt.Sprintf("%d session(s) deleted", len(m.deleted))),
		))
	}
	sb.WriteString("\n  " + dimStyle.Render("press enter or q to exit"))

	return sb.String()
}
