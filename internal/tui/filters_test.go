package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

func manyFilterModel(t *testing.T, width, height int) *Model {
	t.Helper()
	m := newTestModel(t)
	tags := make([]string, 320)
	for i := range tags {
		tags[i] = fmt.Sprintf("t%03d", i)
	}
	writeTask(t, m.store.Dir(), "many.md", "TODO", "["+strings.Join(tags, ", ")+"]", "[]")
	reload(t, m)
	m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	press(m, "f")
	press(m, "j")
	press(m, "j")
	return m
}

func TestFilterReachabilityAndGroupMemory(t *testing.T) {
	for _, size := range [][2]int{{80, 24}, {160, 40}} {
		t.Run(fmt.Sprint(size), func(t *testing.T) {
			m := manyFilterModel(t, size[0], size[1])
			values := m.groupChips(2)
			for _, value := range values {
				if m.filterFocus[2] != value {
					t.Fatalf("focus = %q, want %q", m.filterFocus[2], value)
				}
				if !strings.Contains(ansi.Strip(m.filterView()), "["+value+"]") {
					t.Fatalf("focused %s is hidden", value)
				}
				press(m, "l")
			}
			focus, scroll := m.filterFocus[2], m.filterScroll[2]
			press(m, "k")
			press(m, "j")
			if m.filterFocus[2] != focus || m.filterScroll[2] != scroll {
				t.Fatal("group position lost")
			}
			press(m, "g")
			if m.filterFocus[2] != values[0] || m.filterScroll[2] != 0 {
				t.Fatal("first chip")
			}
			press(m, "pgdown")
			if m.filterScroll[2] == 0 {
				t.Fatal("page down did not scroll")
			}
			press(m, "pgup")
			if m.filterScroll[2] != 0 {
				t.Fatal("page up did not return")
			}
			press(m, "G")
			if m.filterFocus[2] != values[len(values)-1] {
				t.Fatal("last chip")
			}
			focusedHeight := m.contentHeight()
			press(m, "f")
			if m.contentHeight() <= focusedHeight {
				t.Fatal("board must grow when panel loses focus")
			}
		})
	}
}

func TestFilterMarkersCountHiddenChips(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	press(m, "G")
	for _, key := range []string{"g", "pgdown", "G"} {
		press(m, key)
		m.View()
		shown := map[string]bool{}
		for _, r := range m.layout.hits {
			if r.h.kind == hitChip && r.h.row == 2 {
				shown[m.groupChips(2)[r.h.chip]] = true
			}
		}
		values := m.groupChips(2)
		first, last := len(values), -1
		for i, value := range values {
			if shown[value] {
				first = min(first, i)
				last = max(last, i)
			}
		}
		view := ansi.Strip(m.filterView())
		if first > 0 && !strings.Contains(view, fmt.Sprintf("↑%d", first)) {
			t.Fatalf("incorrect above count: %s", view)
		}
		if last < len(values)-1 && !strings.Contains(view, fmt.Sprintf("↓%d", len(values)-last-1)) {
			t.Fatalf("incorrect below count: %s", view)
		}
		if len(shown)+first+len(values)-last-1 != len(values) {
			t.Fatal("chip or hit lost")
		}
		for _, line := range strings.Split(view, "\n") {
			if ansi.StringWidth(line) > m.width {
				t.Fatal("line exceeds width")
			}
		}
	}
	small := newTestModel(t)
	if view := small.filterView(); strings.ContainsAny(view, "↑↓") {
		t.Fatal("markers when chips fit")
	}
}

func TestFilterPinningAndFocus(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	press(m, "G")
	value := m.filterFocus[2]
	press(m, "space")
	if m.groupChips(2)[0] != value || m.filterFocus[2] != value {
		t.Fatal("toggle must pin and retain focus")
	}
	press(m, "right")
	press(m, "G")
	if !strings.Contains(ansi.Strip(m.filterView()), "["+value+"]") {
		t.Fatal("selected chip scrolled away")
	}
	press(m, "g")
	press(m, "enter")
	values := m.groupChips(2)
	if values[len(values)-1] != value || m.filterFocus[2] != value {
		t.Fatal("untoggle must restore sort and focus")
	}
	if !strings.Contains(ansi.Strip(m.filterView()), "["+value+"]") {
		t.Fatal("untoggled focused chip hidden")
	}
}

func TestFilterFindInputAndHelp(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	m.filter.Tags["t319"] = true
	press(m, "/")
	for _, k := range []string{"T", "0", "1"} {
		press(m, k)
	}
	values := m.groupChips(2)
	if len(values) != 11 || values[0] != "t319" || !slices.Contains(values, "t010") {
		t.Fatalf("unexpected narrowed values: %v", values)
	}
	if m.helpGroup() != "Chip find" || !strings.Contains(m.actionHelp(), "toggle chip") {
		t.Fatal("chip find help missing")
	}
	press(m, "?")
	press(m, "esc")
	if m.mode != modeFilter || m.filterFind.Value() != "T01" {
		t.Fatal("help did not restore find")
	}
	press(m, "right")
	value := m.filterFocus[2]
	press(m, "space")
	if !m.filter.Tags[value] || m.filterFind == nil {
		t.Fatal("space must toggle and keep find")
	}
	press(m, "enter")
	if m.filter.Tags[value] || m.filterFocus[2] != value {
		t.Fatal("enter must untoggle and retain focus")
	}
	for _, k := range []string{"q", "f", "F", "r", "!", ":"} {
		press(m, k)
	}
	if m.filterFind.Value() != "T01qfFr!:" || m.mode != modeFilter || m.palette != nil || !m.filter.Tags["t319"] {
		t.Fatal("global keys escaped chip find")
	}
	if !strings.Contains(ansi.Strip(m.View().Content), "Find chips:") {
		t.Fatal("find prompt missing")
	}
	press(m, "esc")
	if m.filterFind != nil || len(m.groupChips(2)) != 322 {
		t.Fatal("escape must clear narrowing")
	}
	press(m, "ctrl+f")
	press(m, "0")
	press(m, "k")
	if m.filterFind != nil || m.filterRow != 1 {
		t.Fatal("group change must clear find")
	}
	press(m, "/")
	_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	if cmd == nil {
		t.Fatal("ctrl+c must quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("ctrl+c did not quit")
	}
}

func TestFilterMouseScrollAndHits(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	m.View()
	var group region
	for _, r := range m.layout.hits {
		if r.h.kind == hitFilterGroup && r.h.row == 2 {
			group = r
		}
	}
	original := m.filterFocus[2]
	m.Update(tea.MouseWheelMsg{X: group.r.x, Y: group.r.y, Button: tea.MouseWheelDown})
	m.View()
	if m.filterScroll[2] != 1 || m.filterFocus[2] != original {
		t.Fatal("wheel must scroll without changing focus")
	}
	var target region
	for _, r := range m.layout.hits {
		if r.h.kind == hitChip && r.h.row == 2 {
			target = r
			break
		}
	}
	value := m.groupChips(2)[target.h.chip]
	line := strings.Split(ansi.Strip(m.View().Content), "\n")[target.r.y]
	if !strings.HasPrefix(line[target.r.x:], "["+value+"]") {
		t.Fatal("hit does not match drawn chip")
	}
	click(m, target)
	if !m.filter.Tags[value] || m.filterFocus[2] != value {
		t.Fatal("scrolled chip click toggled wrong value")
	}
	press(m, "/")
	scroll := m.filterScroll[2]
	m.Update(tea.MouseWheelMsg{X: group.r.x, Y: group.r.y, Button: tea.MouseWheelDown})
	click(m, target)
	if m.filterScroll[2] != scroll || !m.filter.Tags[value] {
		t.Fatal("mouse must be ignored during find")
	}
	press(m, "esc")
	press(m, "right")
	if !strings.Contains(ansi.Strip(m.filterView()), "["+m.filterFocus[2]+"]") {
		t.Fatal("keyboard must reveal focus after wheel")
	}
}

func TestFilterReloadClampsFocusAndScroll(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	press(m, "G")
	press(m, "space")
	press(m, "right")
	press(m, "G")
	if err := os.Remove(filepath.Join(m.store.Dir(), "many.md")); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if m.filterScroll[2] != 0 || m.filter.Tags["t319"] || !slices.Contains(m.groupChips(2), m.filterFocus[2]) {
		t.Fatal("reload did not prune focus, selection and scroll")
	}
	m.View()
}

func TestFilterLayoutLongAndUnicodeChips(t *testing.T) {
	for _, width := range []int{1, 8, 20} {
		lines := layoutGroup("TAGS: ", []string{"日本語", "a-very-long-tag", "x"}, width)
		count := 0
		for _, line := range lines {
			for _, pos := range line.chips {
				count++
				if pos.x < 0 || pos.x+pos.w > width {
					t.Fatalf("invalid chip cells: %+v at width %d", pos, width)
				}
			}
		}
		if count != 3 {
			t.Fatal("chip lost during layout")
		}
	}
}

func TestFilterManySelectionsStayWithinBudget(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	values := m.groupChips(2)
	for _, value := range values {
		m.filter.Tags[value] = true
	}
	for _, value := range values {
		m.filterFocus[2] = value
		m.ensureFilterFocus()
		if !strings.Contains(ansi.Strip(m.View().Content), "["+value+"]") {
			t.Fatalf("selected %s is hidden", value)
		}
		g := m.filterLayout(2)
		if len(g.pinned)+min(g.budget, len(g.lines)) > m.filterBudget(2) {
			t.Fatal("selections exceed group budget")
		}
	}
}

func TestFilterFindNoMatchesResizeAndPaste(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	press(m, "/")
	m.Update(tea.PasteMsg{Content: "t319"})
	if values := m.groupChips(2); len(values) != 1 || values[0] != "t319" {
		t.Fatalf("paste did not narrow: %v", values)
	}
	m.Update(tea.WindowSizeMsg{Width: 160, Height: 40})
	if !strings.Contains(ansi.Strip(m.View().Content), "[t319]") {
		t.Fatal("resize hid match")
	}
	press(m, "x")
	for _, k := range []string{"space", "enter", "right", "pgdown", "pgup"} {
		press(m, k)
	}
	if len(m.groupChips(2)) != 0 || m.filter.Active() != 0 {
		t.Fatal("empty find must not toggle stale focus")
	}
	press(m, "esc")
	if len(m.groupChips(2)) != 322 {
		t.Fatal("find did not clear")
	}
}

func TestFilterClearKeepsFocusedChipVisible(t *testing.T) {
	m := manyFilterModel(t, 80, 24)
	press(m, "G")
	value := m.filterFocus[2]
	press(m, "space")
	press(m, "F")
	if m.filter.Active() != 0 || m.filterFocus[2] != value || !strings.Contains(ansi.Strip(m.filterView()), "["+value+"]") {
		t.Fatal("clearing pinned selections must reveal the focused chip in its alphabetical position")
	}
}
