package tui

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"strings"
)

type helpBinding struct {
	binding key.Binding
	groups  []string
}

func bind(keys []string, label, desc string, groups ...string) helpBinding {
	return helpBinding{key.NewBinding(key.WithKeys(keys...), key.WithHelp(label, desc)), groups}
}

var keymap = []helpBinding{
	bind([]string{"q"}, "q", "quit (leave preview)", "Global"),
	bind([]string{"ctrl+c"}, "ctrl+c", "quit; cancel grab/tags first", "Global"),
	bind([]string{"?"}, "?", "help", "Global"),
	bind([]string{"r"}, "r", "rescan", "Global"),
	bind([]string{"!"}, "!", "warnings", "Global"),
	bind([]string{"h", "l", "left", "right"}, "h/l ←/→", "section / chip", "Board", "Grab", "Filter panel"),
	bind([]string{"j", "k", "down", "up"}, "j/k ↓/↑", "card / row / scroll", "Board", "Grab", "Table", "Preview", "Filter panel", "Tag popup", "Overlay"),
	bind([]string{"g", "G"}, "g/G", "first / last", "Board", "Grab", "Table", "Preview"),
	bind([]string{"ctrl+d", "ctrl+u"}, "ctrl+d/u", "half page", "Board", "Table", "Preview", "Overlay"),
	bind([]string{"enter"}, "enter", "preview", "Board", "Table"),
	bind([]string{"enter"}, "enter", "drop", "Grab"),
	bind([]string{"enter"}, "enter", "follow dependency", "Preview"),
	bind([]string{"enter"}, "enter", "toggle chip", "Filter panel"),
	bind([]string{"enter"}, "enter", "add comma-separated tags", "Tag popup"),
	bind([]string{"enter"}, "enter", "keep query", "Search"),
	bind([]string{"esc"}, "esc", "clear search / close / cancel / save tags", "Board", "Grab", "Table", "Preview", "Filter panel", "Tag popup", "Search"),
	bind([]string{"v"}, "v", "board / table", "Board", "Table"),
	bind([]string{"z"}, "z", "compact / full cards", "Board", "Table"),
	bind([]string{"f"}, "f", "focus / leave filters", "Board", "Table", "Filter panel"),
	bind([]string{"F"}, "F", "clear chips and search", "Board", "Table", "Filter panel"),
	bind([]string{"ctrl+f"}, "ctrl+f", "collapse / expand filters", "Board", "Table", "Filter panel"),
	bind([]string{"tab", "shift+tab"}, "tab / shift+tab", "dependency / tag input-list", "Preview", "Tag popup"),
	bind([]string{"backspace"}, "backspace", "dependency back", "Preview"),
	bind([]string{"space"}, "space", "grab / page down / toggle chip", "Board", "Preview", "Filter panel", "Overlay"),
	bind([]string{"b"}, "b", "page up", "Preview", "Overlay"),
	bind([]string{"pgdown", "f"}, "pgdown / f", "page down", "Preview", "Overlay"),
	bind([]string{"pgup"}, "pgup", "page up", "Preview", "Overlay"),
	bind([]string{"d", "u"}, "d/u", "half page down/up", "Preview", "Overlay"),
	bind([]string{"h", "l", "left", "right"}, "h/l ←/→", "horizontal scroll", "Preview", "Overlay"),
	bind([]string{"H", "L"}, "H/L", "previous / next column bottom", "Board"),
	bind([]string{"J", "K"}, "J/K", "reorder above / below neighbour", "Board"),
	bind([]string{"t"}, "t", "tags", "Board", "Table", "Preview"),
	bind([]string{"e"}, "e", "$VISUAL / $EDITOR", "Board", "Table", "Preview"),
	bind([]string{"y"}, "y", "copy absolute path", "Board", "Table", "Preview"),
	bind([]string{"u"}, "u", "undo (up to 50 actions)", "Board", "Table"),
	bind([]string{"/"}, "/", "live search; table ranks title score", "Board", "Table"),
	bind([]string{"n", "N"}, "n/N", "next / previous match in section order", "Board", "Table"),
	bind([]string{"d", "x", "delete"}, "d / x / delete", "remove selected tag", "Tag popup"),
}

func allBindings() []key.Binding {
	var result []key.Binding
	for _, entry := range keymap {
		result = append(result, entry.binding)
	}
	return result
}

func groupBindings(group string) []key.Binding {
	var result []key.Binding
	for _, entry := range keymap {
		for _, g := range entry.groups {
			if g == group {
				result = append(result, entry.binding)
				break
			}
		}
	}
	return result
}
func (m *Model) helpGroup() string {
	if m.tags != nil {
		return "Tag popup"
	}
	if m.grab != nil {
		return "Grab"
	}
	if m.search != nil {
		return "Search"
	}
	current := m.mode
	if current == modeHelp {
		current = m.overlayMode
	}
	switch current {
	case modeTable:
		return "Table"
	case modePreview:
		return "Preview"
	case modeFilter:
		return "Filter panel"
	}
	return "Board"
}
func renderHelp(group string) string {
	lines := []string{group}
	for _, binding := range groupBindings(group) {
		h := binding.Help()
		lines = append(lines, h.Key+"  "+h.Desc)
	}
	return strings.Join(lines, "\n")
}
func (m *Model) actionHelp() string { return renderHelp(m.helpGroup()) + "\n\n" + renderHelp("Global") }
func (m *Model) shortHelp() string {
	group := m.helpGroup()
	if m.mode == modeHelp || m.mode == modeLogs {
		group = "Overlay"
	}
	keys := groupBindings(group)
	keys = keys[:min(3, len(keys))]
	keys = append(keys, groupBindings("Global")[2])
	return help.New().ShortHelpView(keys)
}
