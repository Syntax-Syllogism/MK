package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
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

func (m *Model) groupChips(row int) []string {
	values := m.facets()[row]
	selected := m.groups()[row]
	slices.SortFunc(values, func(a, b string) int {
		if selected[a] != selected[b] {
			if selected[a] {
				return -1
			}
			return 1
		}
		return strings.Compare(a, b)
	})
	if m.filterFind != nil && row == m.filterRow {
		query := strings.ToLower(m.filterFind.Value())
		values = slices.DeleteFunc(values, func(v string) bool { return !selected[v] && !strings.Contains(strings.ToLower(v), query) })
	}
	return values
}

func (m *Model) pruneFilters() {
	facets := m.facets()
	for row, group := range m.groups() {
		for value := range group {
			if !slices.Contains(facets[row], value) {
				delete(group, value)
			}
		}
		values := m.groupChips(row)
		if !slices.Contains(values, m.filterFocus[row]) {
			m.filterFocus[row] = ""
			if len(values) > 0 {
				m.filterFocus[row] = values[0]
			}
		}
		m.filterLayout(row) // Clamp offsets after facets disappear.
	}
	m.ensureFilterFocus()
}

func (m *Model) filterBudget(row int) int {
	budget := max(1, min(3, m.height/9))
	if m.mode == modeFilter && row == m.filterRow {
		budget = max(budget, m.height/3)
	}
	return budget
}

type chipPos struct{ index, x, w int }
type chipLine struct {
	prefix string
	chips  []chipPos
}

// Positions are terminal cells and are shared by rendering and hit testing.
func layoutGroup(prefix string, chips []string, width int) []chipLine {
	width = max(1, width)
	lines := []chipLine{{prefix: prefix}}
	x := lipgloss.Width(prefix)
	for i, chip := range chips {
		w := lipgloss.Width(chip) + 2
		if x > 0 && x+w > width && (len(lines[len(lines)-1].chips) > 0 || x >= width) {
			lines = append(lines, chipLine{})
			x = 0
		}
		lines[len(lines)-1].chips = append(lines[len(lines)-1].chips, chipPos{i, x, min(w, max(1, width-x))})
		x += min(w, max(1, width-x)) + 1
	}
	return lines
}

type filterGroupLayout struct {
	values              []string
	pinned, lines       []chipLine
	budget, markerWidth int
}

func (m *Model) filterLayout(row int) filterGroupLayout {
	values := m.groupChips(row)
	prefix := []string{"PROJECTS: ", "EPICS: ", "TAGS: "}[row]
	pinned := 0
	for pinned < len(values) && m.groups()[row][values[pinned]] {
		pinned++
	}
	g := filterGroupLayout{values: values, budget: m.filterBudget(row)}
	if pinned > 0 {
		g.pinned = layoutGroup(prefix, values[:pinned], m.width)
		limit := g.budget
		if pinned < len(values) || len(g.pinned) > limit {
			limit = max(0, limit-1)
		}
		// Leave a scrolling line when selections alone exceed the available space.
		g.pinned = g.pinned[:min(limit, len(g.pinned))]
		pinned = 0
		for _, line := range g.pinned {
			pinned += len(line.chips)
		}
		if len(g.pinned) > 0 {
			prefix = ""
		}
	}
	if pinned < len(values) || len(g.pinned) == 0 {
		g.lines = layoutGroup(prefix, values[pinned:], m.width)
	}
	g.budget = max(1, g.budget-len(g.pinned))
	if len(g.lines) > g.budget {
		// Both markers can occupy one line when the window is a single line tall.
		g.markerWidth = min(max(0, m.width-1), 2*(len(fmt.Sprint(len(values)))+1)+1)
		g.lines = layoutGroup(prefix, values[pinned:], m.width-g.markerWidth)
	}
	for i := range g.lines {
		for j := range g.lines[i].chips {
			g.lines[i].chips[j].index += pinned
		}
	}
	m.filterScroll[row] = max(0, min(m.filterScroll[row], max(0, len(g.lines)-g.budget)))
	return g
}

func chipLineIndex(lines []chipLine, index int) int {
	for y, line := range lines {
		for _, chip := range line.chips {
			if chip.index == index {
				return y
			}
		}
	}
	return -1
}

func (m *Model) ensureFilterFocus() {
	row := m.filterRow
	g := m.filterLayout(row)
	index := slices.Index(g.values, m.filterFocus[row])
	if index < 0 && len(g.values) > 0 {
		index = 0
		m.filterFocus[row] = g.values[0]
	}
	y := chipLineIndex(g.lines, index)
	if y < 0 {
		return
	}
	if y < m.filterScroll[row] {
		m.filterScroll[row] = y
	}
	if y >= m.filterScroll[row]+g.budget {
		m.filterScroll[row] = y - g.budget + 1
	}
}

func (m *Model) toggleFilter(row int, value string) {
	group := m.groups()[row]
	group[value] = !group[value]
	m.filterFocus[row] = value
	m.repairSelection()
	if row == m.filterRow {
		m.ensureFilterFocus()
	}
}

func (m *Model) updateFilters(k string) {
	row := m.filterRow
	values := m.groupChips(row)
	index := max(0, slices.Index(values, m.filterFocus[row]))
	switch k {
	case "esc":
		m.mode = m.prevMode
		return
	case "h", "left":
		index--
	case "l", "right":
		index++
	case "g":
		index = 0
	case "G":
		index = len(values) - 1
	case "j", "down", "k", "up":
		step := 1
		if k == "k" || k == "up" {
			step = -1
		}
		m.filterFind = nil
		m.filterRow = max(0, min(2, row+step))
		m.ensureFilterFocus()
		return
	case "ctrl+d", "pgdown", "ctrl+u", "pgup":
		g := m.filterLayout(row)
		y := chipLineIndex(g.lines, index)
		step := g.budget
		if k == "ctrl+u" || k == "pgup" {
			step = -step
		}
		if len(g.lines) > 0 {
			target := max(0, min(len(g.lines)-1, max(0, y)+step))
			if len(g.lines[target].chips) > 0 {
				index = g.lines[target].chips[0].index
			}
		}
	case "space", " ", "enter":
		if len(values) > 0 {
			m.toggleFilter(row, values[index])
		}
		return
	}
	if len(values) > 0 {
		m.filterFocus[row] = values[max(0, min(index, len(values)-1))]
	}
	m.ensureFilterFocus()
}

func (m *Model) filterView() string { return m.renderFilters(false) }
func (m *Model) renderFilters(record bool) string {
	if m.prefs.FiltersCollapsed {
		return ""
	}
	var lines []string
	for row := 0; row < 3; row++ {
		g := m.filterLayout(row)
		start := m.filterScroll[row]
		end := min(len(g.lines), start+g.budget)
		shown := append(slices.Clone(g.pinned), g.lines[start:end]...)
		above, below := 0, 0
		for _, line := range g.lines[:start] {
			above += len(line.chips)
		}
		for _, line := range g.lines[end:] {
			below += len(line.chips)
		}
		if record {
			m.addHit(rect{0, 1 + len(lines), m.width, len(shown)}, hit{kind: hitFilterGroup, row: row})
		}
		for y, line := range shown {
			lineWidth := m.width
			if y >= len(g.pinned) {
				lineWidth -= g.markerWidth
			}
			text := line.prefix
			for _, pos := range line.chips {
				value := g.values[pos.index]
				style := lipgloss.NewStyle()
				if m.groups()[row][value] {
					style = style.Reverse(true)
				}
				if m.mode == modeFilter && row == m.filterRow && value == m.filterFocus[row] {
					style = style.Underline(true).Bold(true)
				}
				text += strings.Repeat(" ", max(0, pos.x-lipgloss.Width(text)))
				text += fitScreen(style.Render("["+value+"]"), pos.w, 1)
				if pos.x+pos.w < lineWidth {
					text += " "
				}
				if record {
					m.addHit(rect{pos.x, 1 + len(lines), pos.w, 1}, hit{kind: hitChip, row: row, chip: pos.index})
				}
			}
			markers := ""
			if y == len(g.pinned) && above > 0 {
				markers = fmt.Sprintf("↑%d", above)
			}
			if y == len(shown)-1 && below > 0 {
				if markers != "" {
					markers += " "
				}
				markers += fmt.Sprintf("↓%d", below)
			}
			if markers != "" {
				text += strings.Repeat(" ", max(0, m.width-lipgloss.Width(text)-lipgloss.Width(markers))) + dim.Render(markers)
			}
			lines = append(lines, fitScreen(text, m.width, 1))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Model) startFilterFind() tea.Cmd {
	input := textinput.New()
	input.Prompt = "Find chips: "
	input.SetWidth(max(1, m.width-14))
	m.filterFind = &input
	return input.Focus()
}

func (m *Model) updateFilterFind(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "esc":
		m.filterFind = nil
		m.ensureFilterFocus()
		return nil
	case "space", " ", "enter", "h", "l", "left", "right", "j", "k", "up", "down", "pgdown", "pgup", "ctrl+d", "ctrl+u":
		m.updateFilters(msg.String())
		return nil
	}
	return m.updateFilterFindInput(msg)
}

func (m *Model) updateFilterFindInput(msg tea.Msg) tea.Cmd {
	old := m.filterFind.Value()
	var cmd tea.Cmd
	*m.filterFind, cmd = m.filterFind.Update(msg)
	if m.filterFind.Value() != old {
		m.ensureFilterFocus()
	}
	return cmd
}
