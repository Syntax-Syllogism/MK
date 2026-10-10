package tui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
)

var accent = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(6))
var dim = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(8))
var danger = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(1))
var warning = lipgloss.NewStyle().Foreground(lipgloss.ANSIColor(3))

func empty(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
func epic(t board.Task) string {
	if t.Status == board.StatusConflict {
		return "!!! GIT CONFLICT"
	}
	if t.ParseError != "" {
		return "!!! PARSE ERROR"
	}
	if t.Epic == "" {
		return board.NoEpic
	}
	return t.Epic
}
func lineCount(s string) int         { return lipgloss.Height(s) }
func lipWrap(s string, w int) string { return lipgloss.Wrap(s, max(1, w), " ") }
func fitScreen(s string, w, h int) string {
	lines := strings.Split(s, "\n")
	lines = lines[:min(len(lines), max(0, h))]
	for i, line := range lines {
		lines[i] = ansi.Truncate(line, max(1, w), "…")
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
func (m *Model) contentHeight() int {
	h := 0
	if filters := m.filterView(); filters != "" {
		h = lineCount(filters)
	}
	return max(1, m.height-2-h)
}
func (m *Model) dependencyWarning(t board.Task) bool {
	if m.deps.Cycles[t.File] {
		return true
	}
	for _, slug := range t.DependsOn {
		if m.deps.Missing(slug) {
			return true
		}
	}
	return false
}
func (m *Model) dependencyLine(t board.Task, cursor int) string {
	var deps []string
	for i, slug := range t.DependsOn {
		s := slug
		if m.deps.Missing(slug) {
			s = warning.Render(slug + "?")
		}
		if i == cursor {
			s = lipgloss.NewStyle().Reverse(true).Render(s)
		}
		deps = append(deps, s)
	}
	if m.deps.Cycles[t.File] {
		deps = append(deps, danger.Render("cycle"))
	}
	return empty(strings.Join(deps, " "))
}
func (m *Model) card(t board.Task, w int, selected bool) string {
	inner := max(1, w-2)
	title := accent.Bold(true).Render(ansi.Truncate(m.highlightTitle(t.File), inner, "…"))
	tags := strings.Join(t.Tags, ", ")
	var lines []string
	if m.prefs.Compact {
		badge := ""
		if len(t.DependsOn) > 0 {
			badge = fmt.Sprintf("⛓%d ", len(t.DependsOn))
			if m.dependencyWarning(t) {
				badge = danger.Render(badge)
			}
		}
		lines = []string{title, ansi.Truncate(badge+dim.Render(tags), inner, "…")}
	} else {
		ep := dim.Render(epic(t))
		if t.ParseError != "" || t.Status == board.StatusConflict {
			ep = danger.Render(epic(t))
		}
		wrapped := strings.Split(lipWrap(m.highlightTitle(t.File), inner), "\n")
		wrapped = wrapped[:min(2, len(wrapped))]
		lines = []string{dim.Render(strings.ToUpper(empty(t.Project))), ep, accent.Bold(true).Render(strings.Join(wrapped, "\n")), dim.Render(ansi.Truncate(tags, inner, "…")), ansi.Truncate(m.dependencyLine(t, -1), inner, "…")}
	}
	style := lipgloss.NewStyle().Width(w).Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.ANSIColor(8))
	if selected {
		style = style.Border(lipgloss.ThickBorder()).BorderForeground(lipgloss.ANSIColor(6))
	}
	if t.Status == board.StatusConflict {
		style = style.BorderForeground(lipgloss.ANSIColor(1))
	}
	if m.grab != nil && m.grab.file == t.File {
		if selected {
			style = style.Border(lipgloss.DoubleBorder()).BorderForeground(lipgloss.ANSIColor(6))
		} else {
			style = style.Faint(true)
		}
	}
	if w < 3 {
		return fitScreen(strings.Join(lines, "\n"), w, len(lines))
	}
	return style.Render(strings.Join(lines, "\n"))
}
func (m *Model) boardView() string {
	first, count, width := layoutColumns(m.width, len(m.sections), m.focusSec)
	height := m.contentHeight()
	var columns []string
	x := 0
	y := 1
	if filters := m.filterView(); filters != "" {
		y += lineCount(filters)
	}
	for i := first; i < first+count; i++ {
		s := m.sections[i]
		w := width
		if i == first {
			w += max(0, m.width-(count*width+count-1))
		}
		tasks := m.visible(s.Name)
		total := 0
		for _, t := range m.snap.Tasks {
			if board.Section(t, m.snap.Columns) == s.Name {
				total++
			}
		}
		label := fmt.Sprintf("%s %d", s.Name, len(tasks))
		if m.filter.Active() > 0 {
			label = fmt.Sprintf("%s %d/%d", s.Name, len(tasks), total)
		}
		if !s.Droppable {
			label += " ⊘"
		}
		if i == first && first > 0 {
			label = fmt.Sprintf("‹ %d ", first) + label
		}
		if i == first+count-1 && i+1 < len(m.sections) {
			label += fmt.Sprintf(" %d ›", len(m.sections)-i-1)
		}
		header := accent
		if s.Name == board.StatusConflict {
			header = danger
		}
		if s.Name == board.StatusOther {
			header = warning
		}
		if i == m.focusSec {
			header = header.Bold(true)
		}
		m.renderX = x
		m.renderY = y + 1
		m.addHit(rect{x, y, w, height}, hit{kind: hitColumn, section: s.Name})
		body := m.columnCards(s.Name, tasks, w, max(0, height-1))
		columns = append(columns, lipgloss.NewStyle().Width(w).Render(fitScreen(header.Render(label)+"\n"+body, w, height)))
		x += w + 1
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, joinGutters(columns)...)
}
func joinGutters(columns []string) []string {
	var result []string
	for i, c := range columns {
		if i > 0 {
			result = append(result, " ")
		}
		result = append(result, c)
	}
	return result
}
func (m *Model) columnCards(section string, tasks []board.Task, w, h int) string {
	if len(tasks) == 0 && (m.grab == nil || m.grab.targetSec != section) {
		return dim.Render("(empty)")
	}
	selected := max(0, fileIndex(tasks, m.selected[section]))
	ghostIndex := -1
	if m.grab != nil && m.grab.targetSec == section {
		// Keep the origin in place while inserting the ghost among the other cards.
		insertion := m.grab.targetIdx
		origin := fileIndex(tasks, m.grab.file)
		if origin >= 0 && origin <= insertion {
			insertion++
		}
		insertion = min(insertion, len(tasks))
		if task, ok := m.task(m.grab.file); ok {
			tasks = append(tasks[:insertion:insertion], append([]board.Task{task}, tasks[insertion:]...)...)
			selected = insertion
			ghostIndex = insertion
		}
	}
	start := max(0, min(m.scroll[section], selected))
	// Advance only as far as needed to keep the selected card visible.
	for start < selected {
		used := 0
		if start > 0 {
			used++
		}
		for i := start; i <= selected; i++ {
			used += lineCount(m.card(tasks[i], w, (m.grab == nil && i == selected) || i == ghostIndex))
		}
		if selected < len(tasks)-1 {
			used++
		}
		if used <= h {
			break
		}
		start++
	}
	m.scroll[section] = start
	var lines []string
	used := 0
	if start > 0 {
		lines = append(lines, fmt.Sprintf("↑ %d more", start))
		used++
	}
	end := start
	for end < len(tasks) {
		card := m.card(tasks[end], w, (m.grab == nil && end == selected) || end == ghostIndex)
		size := lineCount(card)
		reserve := 0
		if end+1 < len(tasks) {
			reserve = 1
		}
		if used+size+reserve > h && end > start {
			break
		}
		shownHeight := min(size, max(0, h-used-reserve))
		m.addHit(rect{m.renderX, m.renderY + used, w, shownHeight}, hit{kind: hitCard, section: section, file: tasks[end].File})
		lines = append(lines, fitScreen(card, w, shownHeight))
		used += min(size, max(0, h-used-reserve))
		end++
		if used >= h {
			break
		}
	}
	if end < len(tasks) {
		lines = append(lines, fmt.Sprintf("↓ %d more", len(tasks)-end))
	}
	return strings.Join(lines, "\n")
}
func (m *Model) warningText() string {
	entries, _ := m.logs.snapshot()
	if len(entries) == 0 {
		return "WARNINGS\nNo recent warnings."
	}
	return "WARNINGS\n" + strings.Join(entries, "\n\n")
}
func (m *Model) View() tea.View {
	m.layout = layout{}
	viewName := m.prefs.View
	right := fmt.Sprintf("%s  filters (%d)  %s", viewName, m.filter.Active(), m.version)
	if m.filter.Query != "" {
		right += fmt.Sprintf(" search: %q", m.filter.Query)
	}
	leftWidth := max(1, m.width-lipgloss.Width(right)-2)
	left := lipgloss.NewStyle().Width(leftWidth).Render(ansi.Truncate("MK // "+m.snap.Dir, leftWidth, "…"))
	top := left + "  " + right
	if m.mode == modeBoard || m.mode == modeTable || m.mode == modeFilter {
		m.addHit(rect{leftWidth + 2 + lipgloss.Width(viewName) + 2, 0, 11, 1}, hit{kind: hitFilters})
	}
	status := m.shortHelp()
	if m.width < 60 {
		status = ""
	}
	if len(m.undo) > 0 {
		status += fmt.Sprintf("  u undo (%d)", len(m.undo))
	}
	if m.grab != nil {
		status = fmt.Sprintf("MOVING %s → %s #%d   enter drop · esc cancel", m.grab.file, m.grab.targetSec, m.grab.targetIdx+1)
	}
	if time.Now().Before(m.toastUntil) {
		status = warning.Render(m.toast)
		if m.toastError {
			status = danger.Render(m.toast)
		}
	}
	var body string
	switch m.mode {
	case modePreview:
		header := m.renderPreviewHeader(true)
		header = fitScreen(header, m.width, max(0, min(lineCount(header), m.height-3)))
		body = header + "\n" + m.preview.viewport.View()
	case modeHelp, modeLogs:
		body = m.overlay.View()
	default:
		var content string
		if m.mode == modeTable || (m.mode == modeFilter && m.prevMode == modeTable) {
			content = m.tableView()
		} else {
			content = m.boardView()
		}
		body = content
		if filters := m.renderFilters(true); filters != "" {
			body = filters + "\n" + content
		}
	}
	if m.tags != nil && m.mode != modeHelp && m.mode != modeLogs {
		popup := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1).Render(m.tagsView())
		body = lipgloss.Place(m.width, max(0, m.height-2), lipgloss.Center, lipgloss.Center, popup)
	}
	if m.search != nil && m.mode != modeHelp && m.mode != modeLogs {
		status = m.search.View()
	}
	screen := fitScreen(top, m.width, 1) + "\n" + fitScreen(body, m.width, max(0, m.height-2)) + "\n" + fitScreen(status, m.width, 1)
	v := tea.NewView(fitScreen(screen, m.width, m.height))
	v.AltScreen = true
	v.MouseMode = m.mouseMode()
	return v
}
