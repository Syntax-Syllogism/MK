package tui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
)

func (m *Model) tableFields(t board.Task) []string {
	deps := fmt.Sprint(len(t.DependsOn))
	if m.dependencyWarning(t) {
		deps += "!"
	}
	return []string{m.highlightTitle(t.File), empty(t.Project), empty(t.Epic), empty(t.Status), empty(strings.Join(t.Tags, ",")), deps}
}
func (m *Model) tableView() string {
	tasks := m.tableTasks()
	headers := []string{"FILE", "PROJECT", "EPIC", "STATUS", "TAGS", "DEPS"}
	widths := []int{4, 7, 4, 6, 4, 4}
	for _, t := range tasks {
		for i, field := range m.tableFields(t) {
			widths[i] = max(widths[i], lipgloss.Width(field))
		}
	}
	// Reserve at least 40% for FILE, sharing the remainder across metadata.
	available := max(6, m.width-5)
	fileWidth := max(1, available*2/5)
	budget := max(5, available-fileWidth)
	for i := 1; i < 6; i++ {
		widths[i] = min(widths[i], max(1, budget/(6-i)))
		budget -= widths[i]
	}
	widths[0] = max(1, available)
	for i := 1; i < 6; i++ {
		widths[0] -= widths[i]
	}
	widths[0] = max(fileWidth, widths[0])
	row := func(fields []string) string {
		parts := make([]string, len(fields))
		for i, f := range fields {
			parts[i] = lipgloss.NewStyle().Width(widths[i]).Render(ansi.Truncate(f, widths[i], "…"))
		}
		return strings.Join(parts, " ")
	}
	lines := []string{accent.Bold(true).Render(row(headers))}
	h := max(1, m.contentHeight()-1)
	index := max(0, fileIndex(tasks, m.tableCur))
	m.tableScroll = max(0, min(m.tableScroll, index))
	if index >= m.tableScroll+h {
		m.tableScroll = index - h + 1
	}
	for i := m.tableScroll; i < len(tasks) && len(lines) <= h; i++ {
		y := 2 + i - m.tableScroll
		if filters := m.filterView(); filters != "" {
			y += lineCount(filters)
		}
		m.addHit(rect{0, y, m.width, 1}, hit{kind: hitCard, file: tasks[i].File})
		line := row(m.tableFields(tasks[i]))
		if tasks[i].File == m.tableCur {
			line = lipgloss.NewStyle().Reverse(true).Render(line)
		}
		lines = append(lines, line)
	}
	if len(tasks) == 0 {
		lines = append(lines, dim.Render("(no visible tasks)"))
	}
	return strings.Join(lines, "\n")
}
