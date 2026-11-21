package main

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m model) openSessionPreview(s ProjectSession) (tea.Model, tea.Cmd) {
	m.previewSession = s
	m.previewMessages = readSessionPreview(s.Path, 80)
	m.previewOffset = 0
	m.state = stateSessionPreview
	return m, nil
}

func (m model) closePreview() (tea.Model, tea.Cmd) {
	m.state = stateProjectDetail
	m.previewMessages = nil
	m.previewOffset = 0
	return m, nil
}

func (m model) handlePreviewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.previewMessages)
	switch msg.String() {
	case "esc", "v", "backspace":
		return m.closePreview()
	case "up", "k":
		if m.previewOffset > 0 {
			m.previewOffset--
		} else if n > 0 {
			m.previewOffset = n - 1
		}
	case "down", "j":
		if m.previewOffset < n-1 {
			m.previewOffset++
		} else {
			m.previewOffset = 0
		}
	case "pgup":
		m.previewOffset -= 5
		if m.previewOffset < 0 {
			m.previewOffset = 0
		}
	case "pgdn":
		m.previewOffset += 5
		if n > 0 && m.previewOffset > n-1 {
			m.previewOffset = n - 1
		}
	case "g":
		m.previewOffset = 0
	case "G":
		if n > 0 {
			m.previewOffset = n - 1
		}
	}
	return m, nil
}

func (m model) viewSessionPreview() string {
	var sb strings.Builder
	s := m.previewSession
	title := s.Title
	if title == "" {
		title = s.ID
	}

	sb.WriteString("\n  " + lipgloss.NewStyle().Foreground(clrPurple).Bold(true).Render("Conversation preview") + "\n")
	sb.WriteString("  " + lipgloss.NewStyle().Foreground(clrFg).Bold(true).Render(truncate(title, 76)) + "\n")
	sb.WriteString("  " + dimStyle.Render("ID: "+s.ID+"  •  "+formatSize(s.Size)) + "\n")
	sb.WriteString("  " + dimStyle.Render(strings.Repeat("─", 78)) + "\n")

	n := len(m.previewMessages)
	if n == 0 {
		sb.WriteString("\n  " + dimStyle.Render("No readable text found in this conversation.") + "\n")
	} else {
		if m.previewOffset >= n {
			m.previewOffset = n - 1
		}
		if m.previewOffset < 0 {
			m.previewOffset = 0
		}
		used := lipgloss.Height(sb.String()) + lipgloss.Height(m.renderHeader())
		maxLines := m.height - used - 2
		if maxLines < 6 {
			maxLines = 6
		}
		textWidth := m.width - 6
		if textWidth < 60 {
			textWidth = 60
		}

		linesUsed := 0
		last := m.previewOffset
		for i := m.previewOffset; i < n && linesUsed < maxLines; i++ {
			msg := m.previewMessages[i]
			role, color := "YOU", clrGreen
			if msg.Role == "assistant" {
				role, color = "CLAUDE", clrPurple
			}
			header := lipgloss.NewStyle().Foreground(color).Bold(true).Render(role)
			if msg.Time != "" {
				header += "  " + dimStyle.Render(msg.Time)
			}
			body := lipgloss.NewStyle().Foreground(clrFg).Width(textWidth).Render(msg.Text)
			sb.WriteString("  " + header + "\n")
			sb.WriteString("  " + body + "\n\n")
			linesUsed += lipgloss.Height(body) + 3
			last = i
		}
		sb.WriteString(helpStyle.Render(fmt.Sprintf("  [%d-%d/%d]", m.previewOffset+1, last+1, n)) + "\n")
	}

	sb.WriteString(helpStyle.Render("\n  ↑/↓/PgUp/PgDn scroll  g/G top/bottom  esc/v/q back"))
	return sb.String()
}
