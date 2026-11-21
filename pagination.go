package main

import "github.com/charmbracelet/lipgloss"

// listPageSize estimates how many project rows fit below the header/footer.
func (m *model) listPageSize() int {
	size := m.height - lipgloss.Height(m.renderHeader()) - 9
	if size < 3 {
		size = 3
	}
	return size
}

// detailPageSize estimates visible rows on the Project Detail screen.
func (m *model) detailPageSize() int {
	size := m.height - lipgloss.Height(m.renderHeader()) - 14
	if size < 3 {
		size = 3
	}
	return size
}

func (m *model) clampListOffset(n int) {
	size := m.listPageSize()
	if n <= size {
		m.listOffset = 0
		return
	}
	if m.listOffset < 0 {
		m.listOffset = 0
	}
	if m.listOffset > n-size {
		m.listOffset = n - size
	}
}

func (m *model) ensureListCursorVisible(n int) {
	size := m.listPageSize()
	if n <= size {
		m.listOffset = 0
		return
	}
	if m.cursor < m.listOffset {
		m.listOffset = m.cursor
	}
	if m.cursor >= m.listOffset+size {
		m.listOffset = m.cursor - size + 1
	}
	m.clampListOffset(n)
}

func (m *model) clampDetailOffset(n int) {
	size := m.detailPageSize()
	if n <= size {
		m.detailOffset = 0
		return
	}
	if m.detailOffset < 0 {
		m.detailOffset = 0
	}
	if m.detailOffset > n-size {
		m.detailOffset = n - size
	}
}

func (m *model) ensureDetailCursorVisible(n int) {
	size := m.detailPageSize()
	if n <= size {
		m.detailOffset = 0
		return
	}
	if m.detailCursor < m.detailOffset {
		m.detailOffset = m.detailCursor
	}
	if m.detailCursor >= m.detailOffset+size {
		m.detailOffset = m.detailCursor - size + 1
	}
	m.clampDetailOffset(n)
}
