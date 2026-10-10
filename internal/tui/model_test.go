package tui

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	writeTask(t, dir, "a.md", "TODO", "[blue]", "[b]")
	writeTask(t, dir, "b.md", "DONE", "[green]", "[]")
	writeTask(t, dir, "c.md", "TODO", "[blue]", "[]")
	store, err := board.Open(dir, board.Options{Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	return NewModel(store, Options{Version: "test"})
}
func writeTask(t *testing.T, dir, file, status, tags, deps string) {
	t.Helper()
	data := "---\nstatus: " + status + "\nproject: mk\ntags: " + tags + "\ndepends_on: " + deps + "\n---\n# " + file + "\n\nBody **bold**.\n"
	if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}
func press(m *Model, k string) {
	msg := tea.KeyPressMsg{Text: k, Code: []rune(k)[0]}
	switch k {
	case "enter":
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		msg = tea.KeyPressMsg{Code: tea.KeyEscape}
	case "tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab}
	case "shift+tab":
		msg = tea.KeyPressMsg{Code: tea.KeyTab, Mod: tea.ModShift}
	case "backspace":
		msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
	case "space":
		msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
	case "left":
		msg = tea.KeyPressMsg{Code: tea.KeyLeft}
	case "right":
		msg = tea.KeyPressMsg{Code: tea.KeyRight}
	case "up":
		msg = tea.KeyPressMsg{Code: tea.KeyUp}
	case "down":
		msg = tea.KeyPressMsg{Code: tea.KeyDown}
	case "pgup":
		msg = tea.KeyPressMsg{Code: tea.KeyPgUp}
	case "pgdown":
		msg = tea.KeyPressMsg{Code: tea.KeyPgDown}
	case "delete":
		msg = tea.KeyPressMsg{Code: tea.KeyDelete}
	default:
		if strings.HasPrefix(k, "ctrl+") {
			msg = tea.KeyPressMsg{Code: []rune(strings.TrimPrefix(k, "ctrl+"))[0], Mod: tea.ModCtrl}
		}
	}
	_, cmd := m.Update(msg)
	if k == "r" && cmd != nil {
		m.Update(cmd())
	}
}
func reload(t *testing.T, m *Model) {
	t.Helper()
	if err := m.store.Scan(); err != nil {
		t.Fatal(err)
	}
	m.Update(reloadMsg{})
}
func TestNavigationAndReload(t *testing.T) {
	m := newTestModel(t)
	press(m, "h")
	press(m, "k")
	if m.currentFile() != "a.md" || m.focusSec != 0 {
		t.Fatal("navigation must clamp")
	}
	press(m, "G")
	if m.currentFile() != "c.md" {
		t.Fatal("last card")
	}
	press(m, "j")
	if m.currentFile() != "c.md" {
		t.Fatal("bottom must clamp")
	}
	writeTask(t, m.store.Dir(), "a.md", "TODO", "[blue]", "[b]")
	reload(t, m)
	if m.currentFile() != "c.md" {
		t.Fatal("same section selection lost")
	}
	writeTask(t, m.store.Dir(), "c.md", "DONE", "[blue]", "[]")
	reload(t, m)
	if m.focusSec != 0 || m.currentFile() != "a.md" {
		t.Fatal("moved selection must use clamped previous index in current section")
	}
	if err := os.Remove(filepath.Join(m.store.Dir(), "a.md")); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if m.currentFile() != "" {
		t.Fatal("empty section selection should clear")
	}
	for i := 0; i < 20; i++ {
		press(m, "l")
	}
	if m.focusSec != len(m.sections)-1 {
		t.Fatal("section must clamp")
	}
}
func TestFilterReloadAndClear(t *testing.T) {
	m := newTestModel(t)
	m.filter.Tags["green"] = true
	if err := os.Remove(filepath.Join(m.store.Dir(), "b.md")); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if m.filter.Active() != 0 {
		t.Fatal("vanished facet still active")
	}
	press(m, "f")
	press(m, "space")
	if m.filter.Active() != 1 {
		t.Fatal("filter chip did not toggle")
	}
	press(m, "f")
	press(m, "F")
	if m.filter.Active() != 0 {
		t.Fatal("F did not clear")
	}
}
func TestPreviewDependencyNavigation(t *testing.T) {
	m := newTestModel(t)
	press(m, "enter")
	if strings.Contains(ansi.Strip(m.preview.rendered), "**bold**") || !strings.Contains(ansi.Strip(m.preview.rendered), "Body bold.") {
		t.Fatal("preview must render Markdown with Glamour")
	}
	press(m, "tab")
	press(m, "enter")
	if m.preview.task.File != "b.md" {
		t.Fatal("dep jump failed")
	}
	press(m, "backspace")
	if m.preview.task.File != "a.md" {
		t.Fatal("dep back failed")
	}
	press(m, "tab")
	press(m, "enter")
	m.filter.Tags["blue"] = true
	press(m, "q")
	if m.filter.Active() != 0 || m.currentFile() != "b.md" {
		t.Fatal("last viewed dependency must be selected and visible")
	}
	if !strings.Contains(m.toast, "filters cleared") {
		t.Fatal("missing clear notification")
	}
}
func TestPreviewScrollTopAndBottom(t *testing.T) {
	for _, tc := range []struct {
		name    string
		content string
		long    bool
	}{
		{name: "long", content: strings.Repeat("Preview line.\n\n", 100), long: true},
		{name: "short", content: "Short preview."},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newTestModel(t)
			m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
			m.snap.Tasks[0].Content = tc.content
			m.openPreview(m.snap.Tasks[0].File)
			m.mode = modePreview
			for i := 0; i < 2; i++ {
				press(m, "G")
				if !m.preview.viewport.AtBottom() {
					t.Fatal("G must scroll to the bottom")
				}
				if tc.long && m.preview.viewport.YOffset() == 0 {
					t.Fatal("long preview must scroll away from the top")
				}
				if !tc.long && m.preview.viewport.YOffset() != 0 {
					t.Fatal("short preview must stay at offset zero")
				}
			}
			for i := 0; i < 2; i++ {
				press(m, "g")
				if m.preview.viewport.YOffset() != 0 {
					t.Fatal("g must scroll to the top")
				}
			}
		})
	}
}

func TestPreviewReloadRemovedAndConflict(t *testing.T) {
	m := newTestModel(t)
	press(m, "enter")
	old := m.preview.task.Content
	if err := os.Remove(filepath.Join(m.store.Dir(), "a.md")); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if !m.preview.removed || m.preview.task.Content != old || !strings.Contains(m.previewHeader(), "file removed") {
		t.Fatal("removed preview must retain content")
	}
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "conflict.md"), []byte("<<<<<<< ours\n=======\n>>>>>>> theirs"), 0600); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	m.openPreview("conflict.md")
	if m.preview.rendered != m.preview.task.Content {
		t.Fatal("conflict must be raw")
	}
}
func TestPreferencesKeys(t *testing.T) {
	m := newTestModel(t)
	press(m, "v")
	press(m, "z")
	press(m, "ctrl+f")
	if m.mode != modeTable || m.prefs.View != "table" || m.prefs.Compact || !m.prefs.FiltersCollapsed {
		t.Fatalf("prefs: %+v", m.prefs)
	}
	if got := loadPrefs(m.prefsFile); got != m.prefs {
		t.Fatal("prefs not persisted")
	}
}
func TestScreenBounds(t *testing.T) {
	m := newTestModel(t)
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 160, Height: 40}, {Width: 20, Height: 10}} {
		m.Update(size)
		for _, k := range []string{"z", "v", "v", "enter", "?", "esc", "q", "!", "esc"} {
			press(m, k)
			screen := ansi.Strip(m.View().Content)
			lines := strings.Split(screen, "\n")
			if len(lines) != size.Height {
				t.Fatalf("%s: height %d != %d", k, len(lines), size.Height)
			}
			for _, line := range lines {
				if ansi.StringWidth(line) > size.Width {
					t.Fatalf("%s: line exceeds width %d: %q", k, size.Width, line)
				}
			}
		}
	}
}
func TestReadOnlyKeysDoNotWriteTasks(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	entries, err := os.ReadDir("../../test-work-items")
	if err != nil {
		t.Fatal(err)
	}
	type original struct {
		bytes []byte
		mtime time.Time
	}
	before := map[string]original{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		data, err := os.ReadFile(filepath.Join("../../test-work-items", name))
		if err != nil {
			t.Fatal(err)
		}
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if strings.HasSuffix(name, ".md") {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			before[name] = original{data, info.ModTime()}
		}
	}
	store, err := board.Open(dir, board.Options{Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	seen := map[string]bool{}
	for _, binding := range allBindings() {
		for _, k := range binding.Keys() {
			if !seen[k] {
				keys = append(keys, k)
				seen[k] = true
			}
		}
	}
	for _, mode := range []string{"board", "table", "preview", "filters", "search", "help"} {
		for _, k := range keys {
			// Quick moves deliberately write tasks on the board. Editor handoff
			// can write externally in board/table/preview; it is text in search.
			if mode == "board" && (k == "H" || k == "L" || k == "J" || k == "K") {
				continue
			}
			if k == "e" && (mode == "board" || mode == "table" || mode == "preview") {
				continue
			}
			t.Run(mode+"/"+k, func(t *testing.T) {
				// Isolate keys so grab/tag entry cannot turn the next key into a
				// drop/save, and undo starts with no intentionally mutating history.
				m := NewModel(store, Options{})
				// Prior cases may persist a table preference; start each case
				// on the board before entering its requested mode.
				m.mode = modeBoard
				m.filter.Query = "body"
				m.repairSelection()
				switch mode {
				case "table":
					press(m, "v")
				case "preview":
					press(m, "enter")
				case "filters":
					press(m, "f")
				case "search":
					press(m, "/")
				case "help":
					press(m, "?")
				}
				press(m, k)
				for name, want := range before {
					path := filepath.Join(dir, name)
					data, err := os.ReadFile(path)
					if err != nil {
						t.Fatal(err)
					}
					info, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					if !bytes.Equal(data, want.bytes) || !info.ModTime().Equal(want.mtime) {
						t.Errorf("task changed: %s", name)
					}
				}
			})
		}
	}
	m := NewModel(store, Options{})
	m.mode = modeBoard
	for _, keys := range [][]string{{"f", "space", "enter", "F", "f"}, {"enter", "tab", "enter", "backspace", "q"}, {"!", "j", "esc"}, {"?", "j", "esc"}, {"r"}} {
		for _, k := range keys {
			press(m, k)
		}
	}
	for name, want := range before {
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, want.bytes) || !info.ModTime().Equal(want.mtime) {
			t.Errorf("task changed: %s", name)
		}
	}
}
