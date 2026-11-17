package tui

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func TestActionKey(t *testing.T) {
	for _, name := range []string{"enter", "shift+enter", "esc", "tab", "shift+tab", "backspace", "space", "delete", "ctrl+f", "ctrl+c", "H", ":", "/", "?", "!"} {
		if got := actionKey(name).String(); got != name {
			t.Errorf("%s: got %s", name, got)
		}
	}
}

func TestPalettePreviewFindDirections(t *testing.T) {
	for _, tc := range []struct {
		desc string
		key  string
		want int
	}{
		{"next occurrence", "enter", 1},
		{"previous occurrence", "shift+enter", 2},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			m := findBody(t, "needle needle needle\n")
			press(m, "ctrl+f")
			findQuery(m, "needle")
			if len(m.preview.find.matches) != 3 || m.preview.find.active != 0 {
				t.Fatal("expected three matches with the first selected")
			}
			if !strings.Contains(m.actionHelp(), tc.key+"  "+tc.desc) {
				t.Fatal("help does not describe the direction's key")
			}
			// Build this group's palette directly: ':' stays literal find input.
			m.startPalette()
			m.palette.input.SetValue(tc.desc)
			items := m.palette.matches()
			if len(items) == 0 || items[0].desc != tc.desc || items[0].press.String() != tc.key {
				t.Fatalf("wrong action for %q: %+v", tc.desc, items)
			}
			press(m, "enter")
			if m.palette != nil || m.preview.find.active != tc.want {
				t.Fatalf("expected palette closed and match %d; active=%d", tc.want, m.preview.find.active)
			}
		})
	}
}

func TestPaletteModesAndDispatch(t *testing.T) {
	for _, modeKey := range []string{"", "v", "enter", "f"} {
		t.Run(modeKey, func(t *testing.T) {
			m := newTestModel(t)
			if modeKey != "" {
				press(m, modeKey)
			}
			originalMode := m.mode
			press(m, ":")
			if m.palette == nil {
				t.Fatal("palette did not open")
			}
			for _, item := range m.palette.items {
				if slices.Contains([]string{"h", "j", "g", "ctrl+d", "pgdown"}, item.keys) {
					t.Fatalf("navigation in palette: %s", item.keys)
				}
			}
			press(m, "esc")
			if m.palette != nil || m.mode != originalMode {
				t.Fatal("cancel changed mode")
			}
			if modeKey == "f" {
				return
			}
			press(m, ":")
			for _, ch := range "tag" {
				press(m, string(ch))
			}
			items := m.palette.matches()
			if len(items) == 0 || items[0].desc != "tags" {
				t.Fatalf("tag filter: %+v", items)
			}
			press(m, "enter")
			if m.palette != nil || m.tags == nil {
				t.Fatal("palette did not dispatch tags")
			}
		})
	}
}

func TestPaletteInputAndHelp(t *testing.T) {
	m := newTestModel(t)
	press(m, ":")
	press(m, "j")
	press(m, "k")
	if m.palette.input.Value() != "jk" {
		t.Fatal("j/k must be text")
	}
	press(m, "?")
	if m.mode != modeHelp || !strings.Contains(m.View().Content, "Palette") {
		t.Fatal("palette help")
	}
	press(m, "esc")
	if m.palette.input.Value() != "jk" {
		t.Fatal("help lost query")
	}
	press(m, "esc")
	for _, entry := range []string{"/", "t", "enter"} {
		press(m, entry)
		if entry == "enter" {
			press(m, "ctrl+f")
		}
		press(m, ":")
		if m.palette != nil {
			t.Fatal("palette opened inside text input")
		}
		switch entry {
		case "/":
			if m.search.Value() != ":" {
				t.Fatal("search lost colon")
			}
			press(m, "esc")
		case "t":
			if m.tags.input.Value() != ":" {
				t.Fatal("tags lost colon")
			}
			press(m, "ctrl+c")
		case "enter":
			if m.preview.find.input.Value() != ":" {
				t.Fatal("find lost colon")
			}
			press(m, "esc")
			press(m, "q")
		}
	}
	press(m, "space")
	press(m, ":")
	if m.palette != nil || m.grab == nil {
		t.Fatal("palette opened during grab")
	}
}

func TestPaletteSelectionMouseAndEmpty(t *testing.T) {
	m := newTestModel(t)
	press(m, ":")
	press(m, "ctrl+n")
	if m.palette.cursor != 1 {
		t.Fatal("next")
	}
	press(m, "ctrl+p")
	if m.palette.cursor != 0 {
		t.Fatal("previous")
	}
	selected := m.currentFile()
	m.Update(tea.MouseWheelMsg{Button: tea.MouseWheelDown})
	if m.currentFile() != selected || m.mode != modeBoard {
		t.Fatal("mouse escaped palette")
	}
	m.palette.input.SetValue("zzzzzzzzz")
	press(m, "enter")
	if m.palette == nil || !strings.Contains(m.paletteView(), "No matching actions") {
		t.Fatal("empty results")
	}
	_, cmd := m.Update(actionKey("ctrl+c"))
	if cmd == nil {
		t.Fatal("ctrl+c must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("not quit")
	}
}

func TestCuratedHelpWidths(t *testing.T) {
	m := newTestModel(t)
	previous := 0
	for _, width := range []int{60, 80, 120, 200} {
		m.width = width
		line := ansi.Strip(m.shortHelp())
		if !strings.HasPrefix(line, "space grab · / search") || !strings.HasSuffix(line, "? help") {
			t.Fatalf("width %d: %q", width, line)
		}
		if ansi.StringWidth(line) > width || len(line) < previous {
			t.Fatalf("width %d: does not fit/grow", width)
		}
		previous = len(line)
	}
	press(m, "t")
	press(m, "newtag")
	press(m, "enter")
	press(m, "esc")
	m.toastUntil = time.Time{}
	for _, width := range []int{60, 80, 120, 200} {
		m.width = width
		line := ansi.Strip(m.shortHelp() + m.undoHint())
		if ansi.StringWidth(line) > width || strings.Count(line, "u undo") != 1 || !strings.Contains(line, "u undo (1)") {
			t.Fatalf("undo hint: %q", line)
		}
	}
	m.width = 59
	if m.shortHelp() != "" {
		t.Fatal("narrow hint")
	}
}

func TestPagerKeysAndPreviewUndo(t *testing.T) {
	pager := reflect.ValueOf(pagerKeys())
	for i := 0; i < pager.NumField(); i++ {
		for _, k := range pager.Field(i).Interface().(key.Binding).Keys() {
			if slices.Contains([]string{"d", "u", "f"}, k) {
				t.Fatalf("conflicting pager key %s", k)
			}
		}
	}
	m := newTestModel(t)
	press(m, "enter")
	original := m.previewHeader()
	press(m, "t")
	press(m, "newtag")
	press(m, "enter")
	press(m, "esc")
	if !strings.Contains(m.previewHeader(), "newtag") {
		t.Fatal("tag not applied")
	}
	press(m, "u")
	if m.previewHeader() != original || len(m.undo) != 0 {
		t.Fatal("preview undo did not refresh")
	}
	for _, viewport := range []bool{false, true} {
		m.mode = modePreview
		if viewport {
			m.mode = modeHelp
			m.overlayMode = modePreview
		}
		vp := &m.preview.viewport
		if viewport {
			vp = &m.overlay
		}
		vp.SetHeight(10)
		vp.SetContent(strings.Repeat("line\n", 100))
		vp.SetYOffset(40)
		for _, k := range []string{"d", "u", "f"} {
			press(m, k)
			if vp.YOffset() != 40 {
				t.Fatalf("%s scrolled", k)
			}
		}
		for _, k := range []string{"ctrl+d", "space", "pgdown"} {
			before := vp.YOffset()
			press(m, k)
			if vp.YOffset() <= before {
				t.Fatalf("%s did not scroll down", k)
			}
		}
		for _, k := range []string{"ctrl+u", "b", "pgup"} {
			before := vp.YOffset()
			press(m, k)
			if vp.YOffset() >= before {
				t.Fatalf("%s did not scroll up", k)
			}
		}
	}
}

func TestPanelAndFindKeys(t *testing.T) {
	for _, modeKey := range []string{"", "v"} {
		m := newTestModel(t)
		if modeKey != "" {
			press(m, modeKey)
		}
		press(m, "p")
		if !m.prefs.FiltersCollapsed || !loadPrefs(m.prefsFile).FiltersCollapsed {
			t.Fatal("panel preference not saved")
		}
		press(m, "ctrl+f")
		if m.search == nil || !m.prefs.FiltersCollapsed {
			t.Fatal("ctrl+f should search")
		}
		press(m, "esc")
		press(m, "f")
		press(m, "ctrl+f")
		if m.search != nil || m.mode != modeFilter || m.filterFind == nil {
			t.Fatal("filter ctrl+f should find chips")
		}
		press(m, "p")
		if m.filterFind.Value() != "p" {
			t.Fatal("collapse key must be text during chip find")
		}
		press(m, "esc")
		press(m, "p")
		if m.mode == modeFilter || !loadPrefs(m.prefsFile).FiltersCollapsed {
			t.Fatal("collapse must leave filter panel")
		}
	}
}

func TestCuratedHelpByMode(t *testing.T) {
	for _, tc := range []struct {
		keys   []string
		prefix string
	}{
		{[]string{"v"}, "/ search · enter preview"},
		{[]string{"enter"}, "ctrl+f find · tab dependency"},
		{[]string{"f"}, "space toggle · / find · j/k group"},
		{[]string{"space"}, "h/l column · j/k position"},
		{[]string{"/"}, "enter keep · esc clear"},
		{[]string{"t"}, "enter add · tab list"},
		{[]string{"enter", "ctrl+f"}, "enter next · shift+enter previous · esc close"},
		{[]string{"?"}, "j/k scroll · esc close"},
		{[]string{":"}, "up select · enter run"},
	} {
		m := newTestModel(t)
		m.width = 200
		for _, k := range tc.keys {
			press(m, k)
		}
		if got := ansi.Strip(m.shortHelp()); !strings.HasPrefix(got, tc.prefix) {
			t.Errorf("%v: %q", tc.keys, got)
		}
		if tc.keys[0] == "space" && !strings.Contains(ansi.Strip(m.View().Content), "h/l column") {
			t.Fatal("grab status hides curated hints")
		}
	}
}

func TestPaletteKeymapCoverage(t *testing.T) {
	for _, group := range []string{"Board", "Table", "Preview", "Filter panel"} {
		var keys []string
		for _, binding := range groupBindings(group) {
			keys = append(keys, binding.Keys()...)
		}
		for _, required := range []string{":"} {
			if !slices.Contains(keys, required) {
				t.Errorf("%s missing %s", group, required)
			}
		}
		if group == "Board" || group == "Table" {
			for _, required := range []string{"p", "ctrl+f"} {
				if !slices.Contains(keys, required) {
					t.Errorf("%s missing %s", group, required)
				}
			}
		}
	}
	m := newTestModel(t)
	press(m, ":")
	for _, item := range m.palette.items {
		if item.press.String() == "" {
			t.Errorf("no dispatch key for %s", item.desc)
		}
	}
	for range len(m.palette.items) + 3 {
		press(m, "down")
	}
	if m.palette.cursor != len(m.palette.items)-1 {
		t.Fatal("selection should clamp at end")
	}
	view := ansi.Strip(m.paletteView())
	last := m.palette.items[len(m.palette.items)-1]
	if !strings.Contains(view, last.desc) {
		t.Fatal("selection must scroll into view")
	}
	if strings.Count(view, "\n") > 13 {
		t.Fatal("more than 12 action rows")
	}
	for _, size := range []tea.WindowSizeMsg{{Width: 20, Height: 8}, {Width: 80, Height: 24}} {
		m.Update(size)
		lines := strings.Split(ansi.Strip(m.View().Content), "\n")
		if len(lines) != size.Height {
			t.Fatal("palette screen height")
		}
		for _, line := range lines {
			if ansi.StringWidth(line) > size.Width {
				t.Fatal("palette screen width")
			}
		}
	}
}
