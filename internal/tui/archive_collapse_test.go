package tui

import (
	"bytes"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestArchiveModesAndUndo(t *testing.T) {
	for _, view := range []string{"board", "table", "preview"} {
		t.Run(view, func(t *testing.T) {
			m := actionModel(t, map[string]string{"a.md": taskText("TODO", "order: 2\n", "[]"), "b.md": taskText("TODO", "order: 3\n", "[]")})
			before := readActionFile(t, m, "a.md")
			if view == "table" {
				press(m, "v")
			}
			if view == "preview" {
				press(m, "enter")
			}
			press(m, "A")
			if _, ok := m.task("a.md"); ok || m.mode == modePreview || m.currentFile() != "b.md" || !strings.Contains(m.toast, "u to undo") {
				t.Fatal("archive/selection failed", m.toast)
			}
			data, err := os.ReadFile(filepath.Join(m.store.Dir(), "archive", "a.md"))
			if err != nil || !bytes.Equal(data, before) {
				t.Fatal("archive changed bytes", err)
			}
			press(m, "u")
			if !bytes.Equal(before, readActionFile(t, m, "a.md")) || m.currentFile() != "a.md" || m.toast != "undid archive" {
				t.Fatal("undo failed")
			}
		})
	}
}

func TestArchiveRefusals(t *testing.T) {
	for _, kind := range []string{"conflict", "parse", "edited archive", "name taken", "disk conflict"} {
		t.Run(kind, func(t *testing.T) {
			body := taskText("TODO", "", "[]")
			if kind == "conflict" {
				body = "<<<<<<< ours\n=======\n>>>>>>> theirs"
			}
			if kind == "parse" {
				body = "---\norder: soon\n---\nBody"
			}
			m := actionModel(t, map[string]string{"a.md": body})
			if kind == "disk conflict" {
				if err := os.WriteFile(filepath.Join(m.store.Dir(), "a.md"), []byte("<<<<<<< ours"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			press(m, "A")
			if kind == "conflict" || kind == "parse" || kind == "disk conflict" {
				if !m.toastError || !strings.Contains(m.toast, "can't archive") || len(m.undo) != 0 {
					t.Fatal(m.toast)
				}
				return
			}
			path := filepath.Join(m.store.Dir(), "a.md")
			if kind == "edited archive" {
				path = filepath.Join(m.store.Dir(), "archive", "a.md")
			}
			if err := os.WriteFile(path, []byte("external"), 0600); err != nil {
				t.Fatal(err)
			}
			press(m, "u")
			if !m.toastError || !strings.Contains(m.toast, "changed or name taken") || len(m.undo) != 0 {
				t.Fatal(m.toast)
			}
		})
	}
}

func TestArchivedDependencyRenderingAndFollow(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": "---\nstatus: TODO\ndepends_on: [b]\n---\nA", "b.md": taskText("DONE", "", "[]")})
	m.selectBoard("b.md")
	press(m, "A")
	m.selectBoard("a.md")
	task, _ := m.task("a.md")
	if m.dependencyWarning(task) || !strings.Contains(ansi.Strip(m.dependencyLine(task, -1)), "b archived") {
		t.Fatal("wrong dependency warning")
	}
	press(m, "enter")
	if !strings.Contains(ansi.Strip(m.previewHeader()), "b archived") {
		t.Fatal(m.previewHeader())
	}
	press(m, "tab")
	press(m, "enter")
	if m.toast != "archived: b" || m.preview.task.File != "a.md" {
		t.Fatal(m.toast)
	}
}

func TestCollapseNavigationPersistenceAndGrab(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[shown]"), "b.md": taskText("IN PROGRESS", "order: 8\n", "[hidden]")})
	m.focusSec = 1
	press(m, "c")
	if !m.collapsed("IN PROGRESS") || m.focusSec != 2 || !slices.Contains(loadPrefs(m.prefsFile).Collapsed, "IN PROGRESS") {
		t.Fatal("collapse not persisted")
	}
	press(m, "h")
	if m.focusSec != 0 {
		t.Fatal("navigation did not skip strip")
	}
	press(m, "l")
	if m.focusSec != 2 {
		t.Fatal("right did not skip strip")
	}
	press(m, "c")
	press(m, "c")
	if m.toast != "can't collapse every column" {
		t.Fatal(m.toast)
	}
	m.filter.Tags["shown"] = true
	m.selectBoard("a.md")
	press(m, "space")
	press(m, "l")
	if m.grab.targetSec != "IN PROGRESS" {
		t.Fatal("grab skipped strip")
	}
	_ = m.View()
	for _, r := range m.layout.hits {
		if r.h.section == "IN PROGRESS" && r.h.kind == hitCard {
			t.Fatal("strip rendered a card")
		}
	}
	press(m, "enter")
	assertOrder(t, m, "a.md", 9)
	press(m, "u")
	press(m, "L")
	assertOrder(t, m, "a.md", 9)
	press(m, "v")
	before := slices.Clone(m.prefs.Collapsed)
	press(m, "c")
	if !slices.Equal(before, m.prefs.Collapsed) {
		t.Fatal("table collapsed column")
	}
}

func TestCollapsedLayout(t *testing.T) {
	flags := []bool{false, true, false, true, false, true}
	for _, width := range []int{80, 160} {
		for focus := range flags {
			first, count, col := layoutColumns(width, len(flags), focus, flags)
			if focus < first || focus >= first+count {
				t.Fatal("focus invisible")
			}
			used := count - 1
			for i := first; i < first+count; i++ {
				if flags[i] {
					used += 5
				} else {
					used += col
					if col < 26 {
						t.Fatal("expanded too small")
					}
				}
			}
			if used > width {
				t.Fatal("layout overflows")
			}
		}
	}
}

func TestCollapsedStripMouse(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "c")
	m.View()
	for _, r := range m.layout.hits {
		if r.h.kind != hitColumn || r.h.section != "TODO" {
			continue
		}
		m.updateMouse(tea.MouseWheelMsg{X: r.r.x, Y: r.r.y, Button: tea.MouseWheelDown})
		if m.scroll["TODO"] != 0 {
			t.Fatal("wheel scrolled strip")
		}
		m.updateMouse(tea.MouseClickMsg{X: r.r.x, Y: r.r.y, Button: tea.MouseLeft})
		if m.collapsed("TODO") || m.focusSec != 0 {
			t.Fatal("click did not expand strip")
		}
		return
	}
	t.Fatal("no strip hit region")
}

func TestOldPrefsAndUnknownCollapsedNames(t *testing.T) {
	path := filepath.Join(t.TempDir(), "prefs.json")
	if err := os.WriteFile(path, []byte(`{"view":"table","compact":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	p := loadPrefs(path)
	if p.View != "table" || p.Compact || len(p.Collapsed) != 0 {
		t.Fatal(p)
	}
	p.Collapsed = []string{"REMOVED"}
	savePrefs(path, p)
	if !slices.Equal(loadPrefs(path).Collapsed, p.Collapsed) {
		t.Fatal("unknown names lost")
	}
}

func TestArchivePreviewDependencySelectsNeighbour(t *testing.T) {
	m := actionModel(t, map[string]string{
		"a.md": "---\nstatus: TODO\ndepends_on: [c]\n---\nA",
		"b.md": taskText("DONE", "order: 1\n", "[]"),
		"c.md": taskText("DONE", "order: 2\n", "[]"),
		"d.md": taskText("DONE", "order: 3\n", "[]"),
	})
	press(m, "enter")
	press(m, "tab")
	press(m, "enter")
	if m.preview.task.File != "c.md" {
		t.Fatal("dependency not followed")
	}
	press(m, "A")
	if m.currentFile() != "d.md" || m.mode != modeBoard {
		t.Fatal("wrong neighbour", m.currentFile())
	}
}
