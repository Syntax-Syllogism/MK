package tui

import (
	"charm.land/bubbles/v2/key"

	tea "charm.land/bubbletea/v2"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func findHit(t *testing.T, m *Model, kind hitKind, file string) region {
	t.Helper()
	m.View()
	for _, r := range m.layout.hits {
		if r.h.kind == kind && (file == "" || r.h.file == file || r.h.section == file) {
			return r
		}
	}
	t.Fatalf("no hit: %v %s", kind, file)
	return region{}
}
func click(m *Model, r region) {
	m.Update(tea.MouseClickMsg{X: r.r.x, Y: r.r.y, Button: tea.MouseLeft})
}
func TestMouseSelectOpenAndChips(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 160, Height: 40}} {
		t.Run(fmt.Sprint(size.Width), func(t *testing.T) {
			m := newTestModel(t)
			m.Update(size)
			r := findHit(t, m, hitCard, "c.md")
			click(m, r)
			if m.currentFile() != "c.md" || m.mode != modeBoard {
				t.Fatal("first click selects")
			}
			click(m, findHit(t, m, hitCard, "c.md"))
			if m.mode != modePreview {
				t.Fatal("second click opens")
			}
			press(m, "q")
			r = findHit(t, m, hitChip, "")
			click(m, r)
			if m.filter.Active() != 1 {
				t.Fatal("chip toggle")
			}
			click(m, findHit(t, m, hitChip, ""))
			if m.filter.Active() != 0 {
				t.Fatal("chip untoggle")
			}
		})
	}
}
func TestMouseWheelColumnAndTable(t *testing.T) {
	m := newTestModel(t)
	for i := 0; i < 30; i++ {
		writeTask(t, m.store.Dir(), fmt.Sprintf("done-%02d.md", i), "DONE", "[]", "[]")
	}
	reload(t, m)
	r := findHit(t, m, hitColumn, "DONE")
	m.Update(tea.MouseWheelMsg{X: r.r.x, Y: r.r.y + 1, Button: tea.MouseWheelDown})
	if m.scroll["DONE"] != 1 || m.scroll["TODO"] != 0 {
		t.Fatal("wheel must scroll hovered column")
	}
	selected := m.selected["DONE"]
	m.View()
	if m.scroll["DONE"] != 1 || m.selected["DONE"] != selected {
		t.Fatal("render must retain wheel position")
	}
	press(m, "v")
	m.View()
	m.Update(tea.MouseWheelMsg{X: 1, Y: 10, Button: tea.MouseWheelDown})
	if m.tableScroll != 3 {
		t.Fatal("table scroll three lines")
	}
}
func TestNoMouseAndIgnoredEvents(t *testing.T) {
	m := newTestModel(t)
	r := findHit(t, m, hitCard, "c.md")
	m = NewModel(m.store, Options{NoMouse: true})
	if m.View().MouseMode != tea.MouseModeNone {
		t.Fatal("no-mouse must disable capture")
	}
	click(m, r)
	if m.currentFile() != "a.md" {
		t.Fatal("no-mouse ignores input")
	}
	m.noMouse = false
	if m.View().MouseMode != tea.MouseModeCellMotion {
		t.Fatal("default mouse capture")
	}
	m.Update(tea.MouseReleaseMsg{X: r.r.x, Y: r.r.y, Button: tea.MouseLeft})
	m.Update(tea.MouseClickMsg{X: r.r.x, Y: r.r.y, Button: tea.MouseRight})
	press(m, "/")
	click(m, r)
	if m.currentFile() != "a.md" {
		t.Fatal("release/right click/search must ignore mouse")
	}
}
func TestMousePreviewDependency(t *testing.T) {
	m := newTestModel(t)
	press(m, "enter")
	click(m, findHit(t, m, hitDependency, ""))
	if m.preview.task.File != "b.md" {
		t.Fatal("dependency click should jump")
	}
}
func TestHelpBindingCoverage(t *testing.T) {
	groups := []string{"Board", "Grab", "Table", "Preview", "Preview find", "Filter panel", "Chip find", "Tag popup", "Search", "Global", "Palette", "Overlay"}
	for _, binding := range allBindings() {
		found := false
		for _, group := range groups {
			for _, b := range groupBindings(group) {
				if b.Help() == binding.Help() {
					found = true
				}
			}
		}
		if !found || binding.Help().Key == "" || binding.Help().Desc == "" {
			t.Fatalf("undocumented binding: %v", binding.Keys())
		}
	}
	m := newTestModel(t)
	press(m, "space")
	press(m, "?")
	if m.helpGroup() != "Grab" {
		t.Fatal("grab mode help")
	}
	press(m, "esc")
	press(m, "esc")
	press(m, "v")
	press(m, "?")
	if m.helpGroup() != "Table" {
		t.Fatal("table mode help")
	}
}

func TestMouseSampleLayoutAcrossSizes(t *testing.T) {
	var file string
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 160, Height: 40}} {
		m := sampleModel(t)
		m.Update(size)
		m.View()
		if file == "" {
			for _, r := range m.layout.hits {
				if r.h.kind == hitCard && r.h.file != m.currentFile() {
					file = r.h.file
					break
				}
			}
		}
		click(m, findHit(t, m, hitCard, file))
		if m.currentFile() != file || m.mode != modeBoard {
			t.Fatalf("%dx%d: sample click failed", size.Width, size.Height)
		}
	}
}
func TestHelpSuspendsTextEntryAndGrab(t *testing.T) {
	m := newTestModel(t)
	press(m, "/")
	press(m, "a")
	press(m, "?")
	if m.helpGroup() != "Search" || m.mode != modeHelp {
		t.Fatal("search help group")
	}
	press(m, "esc")
	press(m, "enter")
	if m.filter.Query != "a" || m.search != nil {
		t.Fatal("search help return lost query")
	}
	press(m, "F")
	press(m, "t")
	press(m, "?")
	if m.helpGroup() != "Tag popup" || !strings.Contains(m.View().Content, "Tag popup") {
		t.Fatal("tag help must display over popup")
	}
	press(m, "esc")
	press(m, "ctrl+c")
	if m.tags != nil {
		t.Fatal("tag help return cancellation")
	}
	press(m, "space")
	press(m, "?")
	press(m, "ctrl+c")
	if m.grab != nil || m.mode == modeHelp {
		t.Fatal("cancel grab through help")
	}
}

func TestViewportBindingsAppearInHelp(t *testing.T) {
	defaults := reflect.ValueOf(pagerKeys())
	for _, group := range []string{"Preview", "Overlay"} {
		var documented []string
		for _, binding := range groupBindings(group) {
			documented = append(documented, binding.Keys()...)
		}
		for i := 0; i < defaults.NumField(); i++ {
			for _, k := range defaults.Field(i).Interface().(key.Binding).Keys() {
				if !slices.Contains(documented, k) {
					t.Fatalf("%s: undocumented viewport key %s", group, k)
				}
			}
		}
	}
}
