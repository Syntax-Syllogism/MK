package tui

import (
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/sahilm/fuzzy"
)

func (m *Model) startSearch() tea.Cmd {
	input := textinput.New()
	input.Prompt = "/ "
	input.SetWidth(max(1, m.width-3))
	input.SetValue(m.filter.Query)
	m.search = &input
	return input.Focus()
}
func (m *Model) updateSearch(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "enter":
		m.search = nil
		return nil
	case "esc":
		m.search = nil
		m.filter.Query = ""
		m.repairSelection()
		return nil
	case "ctrl+c":
		return tea.Quit
	}
	return m.updateSearchInput(msg)
}
func (m *Model) nextMatch(back bool) {
	if strings.TrimSpace(m.filter.Query) == "" {
		return
	}
	var files []string
	for _, s := range m.sections {
		for _, t := range m.visible(s.Name) {
			files = append(files, t.File)
		}
	}
	if len(files) == 0 {
		return
	}
	index := 0
	for i, file := range files {
		if file == m.currentFile() {
			index = i
			break
		}
	}
	step := 1
	if back {
		step = -1
	}
	file := files[(index+step+len(files))%len(files)]
	m.selectBoard(file)
	m.tableCur = file
}
func (m *Model) highlightTitle(title string) string {
	return m.highlightText(title, strings.TrimSuffix(title, ".md"))
}

func (m *Model) highlightHeading(heading string) string {
	return m.highlightText(heading, heading)
}

func (m *Model) highlightText(text, candidate string) string {
	indexes := map[int]bool{}
	for _, term := range strings.Fields(m.filter.Query) {
		if hits := fuzzy.Find(term, []string{candidate}); len(hits) > 0 {
			for _, i := range hits[0].MatchedIndexes {
				indexes[i] = true
			}
		}
	}
	var out strings.Builder
	for i, r := range text {
		text := string(r)
		if indexes[i] {
			text = lipgloss.NewStyle().Bold(true).Underline(true).Render(text)
		}
		out.WriteString(text)
	}
	return out.String()
}

func (m *Model) updateSearchInput(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	*m.search, cmd = m.search.Update(msg)
	if query := m.search.Value(); query != m.filter.Query {
		m.filter.Query = query
		m.repairSelection()
	}
	return cmd
}
