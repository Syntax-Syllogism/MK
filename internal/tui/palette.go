package tui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/sahilm/fuzzy"
)

type paletteState struct {
	input  textinput.Model
	items  []paletteItem
	cursor int
}
type paletteItem struct {
	keys, desc string
	press      tea.KeyPressMsg
}

// actionKey constructs the same messages as keyboard input for action dispatch.
func actionKey(name string) tea.KeyPressMsg {
	msg := tea.KeyPressMsg{}
	if strings.HasPrefix(name, "ctrl+") {
		msg.Mod = tea.ModCtrl
		name = strings.TrimPrefix(name, "ctrl+")
	} else if strings.HasPrefix(name, "shift+") {
		msg.Mod = tea.ModShift
		name = strings.TrimPrefix(name, "shift+")
	}
	switch name {
	case "enter":
		msg.Code = tea.KeyEnter
	case "esc":
		msg.Code = tea.KeyEscape
	case "tab":
		msg.Code = tea.KeyTab
	case "backspace":
		msg.Code = tea.KeyBackspace
	case "space":
		msg.Code = tea.KeySpace
		msg.Text = " "
	case "delete":
		msg.Code = tea.KeyDelete
	default:
		msg.Code = []rune(name)[0]
		if msg.Mod == 0 {
			msg.Text = name
		}
	}
	return msg
}

func (m *Model) startPalette() tea.Cmd {
	group := m.helpGroup()
	state := &paletteState{input: textinput.New()}
	state.input.Prompt = ": "
	state.input.SetWidth(max(1, min(66, m.width-8)))
	for _, entry := range keymap {
		if !entry.palette || (!slices.Contains(entry.groups, group) && !slices.Contains(entry.groups, "Global")) {
			continue
		}
		name := entry.binding.Keys()[0]
		state.items = append(state.items, paletteItem{keys: entry.binding.Help().Key, desc: entry.binding.Help().Desc, press: actionKey(name)})
	}
	m.palette = state
	return state.input.Focus()
}

func (p *paletteState) matches() []paletteItem {
	if p.input.Value() == "" {
		return p.items
	}
	labels := make([]string, len(p.items))
	for i, item := range p.items {
		labels[i] = item.desc + " " + item.keys
	}
	var items []paletteItem
	for _, match := range fuzzy.Find(p.input.Value(), labels) {
		items = append(items, p.items[match.Index])
	}
	return items
}

func (m *Model) updatePalette(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	p := m.palette
	items := p.matches()
	switch msg.String() {
	case "esc":
		m.palette = nil
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "up", "ctrl+p":
		p.cursor = max(0, p.cursor-1)
	case "down", "ctrl+n":
		p.cursor = max(0, min(len(items)-1, p.cursor+1))
	case "enter":
		if len(items) > 0 {
			item := items[min(p.cursor, len(items)-1)]
			m.palette = nil
			return m.Update(item.press)
		}
	default:
		old := p.input.Value()
		var cmd tea.Cmd
		p.input, cmd = p.input.Update(msg)
		if old != p.input.Value() {
			p.cursor = 0
		}
		return m, cmd
	}
	return m, nil
}

func (m *Model) paletteView() string {
	p := m.palette
	w := max(1, min(70, m.width-4))
	p.input.SetWidth(max(1, w-2))
	lines := []string{"COMMAND PALETTE", p.input.View()}
	items := p.matches()
	rows := max(0, min(12, m.height-8))
	start := max(0, p.cursor-rows+1)
	for i := start; i < min(len(items), start+rows); i++ {
		item := items[i]
		prefix := "  "
		if i == p.cursor {
			prefix = "› "
		}
		keyWidth := ansi.StringWidth(item.keys)
		desc := ansi.Truncate(prefix+item.desc, max(1, w-keyWidth-1), "…")
		gap := strings.Repeat(" ", max(1, w-ansi.StringWidth(desc)-keyWidth))
		lines = append(lines, desc+gap+dim.Render(item.keys))
	}
	if len(items) == 0 {
		lines = append(lines, dim.Render("No matching actions"))
	}
	return strings.Join(lines, "\n")
}
