package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
)

func sampleModel(t *testing.T) *Model {
	t.Helper()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	dir := t.TempDir()
	entries, err := os.ReadDir("../../test-work-items")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join("../../test-work-items", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, entry.Name()), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := board.Open(dir, board.Options{Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(store, Options{Version: "test"})
	m.snap.Dir = "/tasks"
	return m
}
func TestSampleScreens(t *testing.T) {
	for _, size := range []tea.WindowSizeMsg{{Width: 80, Height: 24}, {Width: 160, Height: 40}} {
		for _, view := range []string{"board", "table"} {
			t.Run(fmt.Sprintf("%s-%dx%d", view, size.Width, size.Height), func(t *testing.T) {
				m := sampleModel(t)
				m.Update(size)
				if view == "table" {
					press(m, "v")
				}
				lines := strings.Split(ansi.Strip(m.View().Content), "\n")
				for i := range lines {
					lines[i] = strings.TrimRight(lines[i], " ")
				}
				got := strings.Join(lines, "\n") + "\n"
				path := filepath.Join("testdata", t.Name()[strings.LastIndex(t.Name(), "/")+1:]+".golden")
				if os.Getenv("UPDATE_GOLDEN") == "1" {
					if err := os.MkdirAll("testdata", 0755); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(path, []byte(got), 0644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if got != string(want) {
					t.Fatalf("screen differs from %s", path)
				}
			})
		}
	}
}
func TestDiagnosticRenderingAndMissingDependency(t *testing.T) {
	m := newTestModel(t)
	writeTask(t, m.store.Dir(), "a.md", "TODO", "[]", "[missing, a]")
	reload(t, m)
	if !m.dependencyWarning(m.snap.Tasks[0]) || !strings.Contains(ansi.Strip(m.dependencyLine(m.snap.Tasks[0], -1)), "missing? a cycle") {
		t.Fatal("missing/cycle badges absent")
	}
	press(m, "enter")
	press(m, "tab")
	press(m, "enter")
	if m.toast != "unknown card: missing" || m.preview.task.File != "a.md" {
		t.Fatal("missing dependency must show toast")
	}
	press(m, "q")
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "bad.md"), []byte("---\norder: soon\n---\nBody remains."), 0600); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	m.prevMode = modeBoard
	m.mode = modePreview
	m.openPreview("bad.md")
	if !strings.Contains(ansi.Strip(m.previewHeader()), "PARSE ERROR:") || !strings.Contains(ansi.Strip(m.preview.rendered), "Body remains.") {
		t.Fatal("parse-error preview should retain body and show banner")
	}
}
func TestColumnScrolling(t *testing.T) {
	m := newTestModel(t)
	m.prefs.FiltersCollapsed = true
	for i := 0; i < 25; i++ {
		writeTask(t, m.store.Dir(), fmt.Sprintf("task-%02d.md", i), "TODO", "[]", "[]")
	}
	reload(t, m)
	screen := ansi.Strip(m.View().Content)
	if !strings.Contains(screen, "↓ ") {
		t.Fatal("missing bottom overflow")
	}
	press(m, "G")
	screen = ansi.Strip(m.View().Content)
	if !strings.Contains(screen, "task-24.md") || !strings.Contains(screen, "↑ ") {
		t.Fatal("selected last card must scroll into view")
	}
}

func TestCompactCardHasTwoContentLines(t *testing.T) {
	m := newTestModel(t)
	task := board.Task{File: "a-very-long-filename-that-needs-truncation.md", Tags: []string{"a-very-long-tag-that-needs-truncation"}}
	for _, selected := range []bool{false, true} {
		card := m.card(task, 26, selected)
		if lineCount(card) != 4 || ansi.StringWidth(strings.Split(ansi.Strip(card), "\n")[0]) != 26 {
			t.Fatalf("compact card must have two content lines inside its border: %q", ansi.Strip(card))
		}
	}
}
