package tui

import (
	"slices"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
)

func (m *Model) facets() [][]string {
	p, e, t := board.Facets(m.snap.Tasks)
	return [][]string{p, e, t}
}
func (m *Model) groups() []map[string]bool {
	return []map[string]bool{m.filter.Projects, m.filter.Epics, m.filter.Tags}
}
func (m *Model) pruneFilters() {
	facets := m.facets()
	for row, group := range m.groups() {
		for value := range group {
			if !slices.Contains(facets[row], value) {
				delete(group, value)
			}
		}
	}
	m.filterRow = max(0, min(2, m.filterRow))
	m.filterChip = max(0, min(m.filterChip, len(facets[m.filterRow])-1))
}
func (m *Model) updateFilters(k string) {
	rows := m.facets()
	switch k {
	case "esc":
		m.mode = m.prevMode
	case "h", "left":
		m.filterChip--
	case "l", "right":
		m.filterChip++
	case "j", "down":
		m.filterRow++
		m.filterChip = 0
	case "k", "up":
		m.filterRow--
		m.filterChip = 0
	}
	m.filterRow = max(0, min(2, m.filterRow))
	m.filterChip = max(0, min(m.filterChip, len(rows[m.filterRow])-1))
	if (k == "space" || k == "enter") && len(rows[m.filterRow]) > 0 {
		value := rows[m.filterRow][m.filterChip]
		group := m.groups()[m.filterRow]
		group[value] = !group[value]
		m.repairSelection()
	}
}
func (m *Model) filterView() string { return m.renderFilters(false) }
func (m *Model) renderFilters(record bool) string {
	if m.prefs.FiltersCollapsed {
		return ""
	}
	var lines []string
	names := []string{"PROJECTS", "EPICS", "TAGS"}
	rowHeight := max(1, min(3, m.height/9))
	for row, values := range m.facets() {
		prefix := names[row] + ": "
		start := 0
		if m.mode == modeFilter && row == m.filterRow {
			width := lipgloss.Width(prefix)
			for i := 0; i <= m.filterChip && i < len(values); i++ {
				width += lipgloss.Width(values[i]) + 3
			}
			if width > m.width*rowHeight {
				start = m.filterChip
			}
		}
		line := prefix
		x := lipgloss.Width(prefix)
		y := 0
		for i := start; i < len(values); i++ {
			value := values[i]
			chip := "[" + value + "]"
			w := lipgloss.Width(chip)
			if x+w > m.width && x > lipgloss.Width(prefix) {
				lines = append(lines, line)
				line = ""
				x = 0
				y++
			}
			if y >= rowHeight {
				break
			}
			style := lipgloss.NewStyle()
			if m.groups()[row][value] {
				style = style.Reverse(true)
			}
			if m.mode == modeFilter && row == m.filterRow && i == m.filterChip {
				style = style.Underline(true).Bold(true)
			}
			shown := fitScreen(style.Render(chip), max(1, m.width-x), 1)
			if record {
				m.addHit(rect{x, 1 + len(lines), min(w, m.width-x), 1}, hit{kind: hitChip, row: row, chip: i})
			}
			line += shown
			if x+w < m.width {
				line += " "
			}
			x += w + 1
		}
		if y < rowHeight {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
