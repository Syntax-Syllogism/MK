package tui

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/viewport"
	"github.com/charmbracelet/x/ansi"
)

type helpBinding struct {
	binding key.Binding
	groups  []string
	palette bool
	short   map[string]shortHint
}

func bind(keys []string, label, desc string, groups ...string) helpBinding {
	return helpBinding{binding: key.NewBinding(key.WithKeys(keys...), key.WithHelp(label, desc)), groups: groups, palette: true}
}

type shortHint struct {
	priority int
	desc     string
}

func (b helpBinding) hint(group string, priority int, desc string) helpBinding {
	if b.short == nil {
		b.short = map[string]shortHint{}
	}
	b.short[group] = shortHint{priority, desc}
	return b
}

func nav(keys []string, label, desc string, groups ...string) helpBinding {
	b := bind(keys, label, desc, groups...)
	b.palette = false
	return b
}

// pagerKeys removes pager aliases that conflict with task actions.
func pagerKeys() viewport.KeyMap {
	k := viewport.DefaultKeyMap()
	k.PageDown.SetKeys("pgdown", "space")
	k.PageUp.SetKeys("pgup", "b")
	k.HalfPageUp.SetKeys("ctrl+u")
	k.HalfPageDown.SetKeys("ctrl+d")
	return k
}

var keymap = []helpBinding{
	bind([]string{"A"}, "A", "archive (u undoes)", "Board", "Table", "Preview").hint("Board", 11, "archive").hint("Table", 9, "archive").hint("Preview", 10, "archive"),
	bind([]string{"c"}, "c", "collapse / expand column", "Board").hint("Board", 12, "collapse"),
	bind([]string{"/", "ctrl+f"}, "/ ctrl+f", "find chips in group", "Filter panel").hint("Filter panel", 2, "find"),
	nav([]string{"g", "G"}, "g/G", "first / last chip", "Filter panel"),
	nav([]string{"ctrl+d", "ctrl+u", "pgdown", "pgup"}, "ctrl+d/u pgdown/up", "page chips", "Filter panel", "Chip find"),
	nav([]string{"h", "l", "left", "right"}, "h/l ←/→", "previous / next chip", "Chip find"),
	nav([]string{"j", "k", "down", "up"}, "j/k ↓/↑", "change group and close find", "Chip find"),
	nav([]string{"type"}, "type", "narrow chips by substring", "Chip find"),
	bind([]string{"space", "enter"}, "space / enter", "toggle chip; keep finding", "Chip find").hint("Chip find", 1, "toggle"),
	bind([]string{"esc"}, "esc", "clear and close find", "Chip find").hint("Chip find", 2, "clear"),
	bind([]string{"ctrl+f", "/"}, "ctrl+f /", "find in body", "Preview").hint("Preview", 1, "find"),
	bind([]string{"enter"}, "enter", "next occurrence", "Preview find").hint("Preview find", 1, "next"),
	bind([]string{"shift+enter"}, "shift+enter", "previous occurrence", "Preview find").hint("Preview find", 2, "previous"),
	bind([]string{"esc"}, "esc", "clear and close find", "Preview find").hint("Preview find", 3, "close"),
	bind([]string{"ctrl+f"}, "ctrl+f", "focus find; type literal text", "Preview find"),
	bind([]string{"q"}, "q", "quit (leave preview)", "Global").hint("Preview", 9, "close"),
	bind([]string{"ctrl+c"}, "ctrl+c", "quit; cancel grab/tags first", "Global"),
	bind([]string{"?"}, "?", "help", "Global"),
	bind([]string{"r"}, "r", "rescan", "Global"),
	bind([]string{"!"}, "!", "warnings", "Global"),
	nav([]string{"h", "l", "left", "right"}, "h/l ←/→", "section / chip", "Board", "Grab", "Filter panel").hint("Grab", 1, "column"),
	nav([]string{"j", "k", "down", "up"}, "j/k ↓/↑", "card / row / scroll", "Board", "Grab", "Table", "Preview", "Filter panel", "Tag popup", "Overlay").hint("Grab", 2, "position").hint("Filter panel", 2, "group").hint("Overlay", 1, "scroll"),
	nav([]string{"g", "G"}, "g/G", "first / last", "Board", "Grab", "Table", "Preview"),
	nav([]string{"ctrl+d", "ctrl+u"}, "ctrl+d/u", "half page", "Board", "Table", "Preview", "Overlay"),
	bind([]string{"enter"}, "enter", "preview", "Board", "Table").hint("Board", 3, "preview").hint("Table", 2, "preview"),
	bind([]string{"enter"}, "enter", "drop", "Grab").hint("Grab", 3, "drop"),
	bind([]string{"enter"}, "enter", "follow dependency", "Preview").hint("Preview", 3, "follow"),
	bind([]string{"enter"}, "enter", "toggle chip", "Filter panel"),
	bind([]string{"enter"}, "enter", "add comma-separated tags", "Tag popup").hint("Tag popup", 1, "add"),
	bind([]string{"enter"}, "enter", "keep query", "Search").hint("Search", 1, "keep"),
	bind([]string{"esc"}, "esc", "clear search", "Board", "Table"),
	bind([]string{"esc"}, "esc", "cancel", "Grab").hint("Grab", 4, "cancel"),
	bind([]string{"esc"}, "esc", "dependency back / leave preview", "Preview"),
	bind([]string{"esc"}, "esc", "leave filters", "Filter panel").hint("Filter panel", 5, "leave"),
	bind([]string{"esc"}, "esc", "save tags", "Tag popup").hint("Tag popup", 4, "save"),
	bind([]string{"esc"}, "esc", "clear query and close", "Search").hint("Search", 2, "clear"),
	bind([]string{"v"}, "v", "board / table", "Board", "Table").hint("Board", 9, "view").hint("Table", 8, "view"),
	bind([]string{"z"}, "z", "compact / full cards", "Board", "Table").hint("Board", 10, "density"),
	bind([]string{"f"}, "f", "focus / leave filters", "Board", "Table", "Filter panel").hint("Board", 7, "filters").hint("Table", 6, "filters"),
	bind([]string{"F"}, "F", "clear chips and search", "Board", "Table", "Filter panel").hint("Filter panel", 3, "clear"),
	bind([]string{"p"}, "p", "collapse / expand filters", "Board", "Table", "Filter panel").hint("Filter panel", 4, "collapse"),
	bind([]string{"tab"}, "tab", "next dependency", "Preview").hint("Preview", 2, "dependency"),
	bind([]string{"shift+tab"}, "shift+tab", "previous dependency", "Preview"),
	bind([]string{"tab", "shift+tab"}, "tab / shift+tab", "switch tag input/list", "Tag popup").hint("Tag popup", 2, "list"),
	bind([]string{"backspace"}, "backspace", "dependency back", "Preview").hint("Preview", 4, "back"),
	bind([]string{"space"}, "space", "grab", "Board").hint("Board", 1, "grab"),
	nav([]string{"space"}, "space", "page down", "Preview", "Overlay"),
	bind([]string{"space"}, "space", "toggle chip", "Filter panel").hint("Filter panel", 1, "toggle"),
	nav([]string{"b"}, "b", "page up", "Preview", "Overlay"),
	nav([]string{"pgdown"}, "pgdown", "page down", "Preview", "Overlay"),
	nav([]string{"pgup"}, "pgup", "page up", "Preview", "Overlay"),
	nav([]string{"h", "l", "left", "right"}, "h/l ←/→", "horizontal scroll", "Preview", "Overlay"),
	bind([]string{"H"}, "H", "move to previous column bottom", "Board"),
	bind([]string{"L"}, "L", "move to next column bottom", "Board"),
	bind([]string{"J"}, "J", "reorder below neighbour", "Board"),
	bind([]string{"K"}, "K", "reorder above neighbour", "Board"),
	bind([]string{"t"}, "t", "tags", "Board", "Table", "Preview").hint("Board", 4, "tags").hint("Table", 3, "tags").hint("Preview", 5, "tags"),
	bind([]string{"e"}, "e", "$VISUAL / $EDITOR", "Board", "Table", "Preview").hint("Board", 5, "edit").hint("Table", 4, "edit").hint("Preview", 6, "edit"),
	bind([]string{"y"}, "y", "copy absolute path", "Board", "Table", "Preview"),
	bind([]string{"u"}, "u", "undo (up to 50 actions)", "Board", "Table", "Preview").hint("Board", 6, "undo").hint("Table", 5, "undo").hint("Preview", 7, "undo"),
	bind([]string{"/", "ctrl+f"}, "/ ctrl+f", "live search; table ranks title score", "Board", "Table").hint("Board", 2, "search").hint("Table", 1, "search"),
	bind([]string{"n"}, "n", "next match in section order", "Board", "Table"),
	bind([]string{"N"}, "N", "previous match in section order", "Board", "Table"),
	bind([]string{"d", "x", "delete"}, "d / x / delete", "remove selected tag", "Tag popup").hint("Tag popup", 3, "remove"),
	bind([]string{":"}, ":", "command palette", "Board", "Table", "Preview", "Filter panel").hint("Board", 8, "palette").hint("Table", 7, "palette").hint("Preview", 8, "palette"),
	nav([]string{"up", "down", "ctrl+n", "ctrl+p"}, "↑/↓ ctrl+n/p", "select action", "Palette").hint("Palette", 1, "select"),
	bind([]string{"enter"}, "enter", "run action", "Palette").hint("Palette", 2, "run"),
	bind([]string{"esc"}, "esc", "close", "Palette", "Overlay").hint("Palette", 3, "close").hint("Overlay", 2, "close"),
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
	if m.filterFind != nil {
		return "Chip find"
	}
	if m.palette != nil {
		return "Palette"
	}
	if m.preview.find != nil {
		return "Preview find"
	}
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
func (m *Model) undoHint() string {
	if len(m.undo) == 0 || m.mode == modeHelp || m.mode == modeLogs {
		return ""
	}
	switch m.helpGroup() {
	case "Board", "Table", "Preview":
		return fmt.Sprintf("  u undo (%d)", len(m.undo))
	}
	return ""
}

func shortLabel(entry helpBinding) string {
	switch entry.binding.Keys()[0] {
	case "h":
		return "h/l"
	case "j":
		return "j/k"
	}
	return entry.binding.Keys()[0]
}

func (m *Model) shortHelp() string {
	if m.width < 60 {
		return ""
	}
	group := m.helpGroup()
	if m.mode == modeHelp || m.mode == modeLogs {
		group = "Overlay"
	}
	var entries []helpBinding
	for _, entry := range keymap {
		if _, ok := entry.short[group]; ok {
			if entry.binding.Keys()[0] == "u" && m.undoHint() != "" {
				continue
			}
			entries = append(entries, entry)
		}
	}
	slices.SortStableFunc(entries, func(a, b helpBinding) int { return a.short[group].priority - b.short[group].priority })
	var keys []key.Binding
	for _, entry := range entries {
		keys = append(keys, key.NewBinding(key.WithKeys(entry.binding.Keys()...), key.WithHelp(shortLabel(entry), entry.short[group].desc)))
	}
	helpKey := groupBindings("Global")[2]
	renderer := help.New()
	renderer.ShortSeparator = " · "
	result := renderer.ShortHelpView([]key.Binding{helpKey})
	for n := 1; n <= len(keys); n++ {
		candidate := renderer.ShortHelpView(append(slices.Clone(keys[:n]), helpKey))
		if ansi.StringWidth(candidate)+ansi.StringWidth(m.undoHint()) > m.width {
			break
		}
		result = candidate
	}
	return result
}
