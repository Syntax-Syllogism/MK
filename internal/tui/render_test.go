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

func TestCardHeading(t *testing.T) {
	m := newTestModel(t)
	for _, compact := range []bool{true, false} {
		m.prefs.Compact = compact
		for _, selected := range []bool{false, true} {
			for _, width := range []int{26, 60} {
				task := board.Task{File: "task.md", Tags: []string{"tag"}}
				before := m.card(task, width, selected)
				task.Heading = "Readable heading"
				card := m.card(task, width, selected)
				if lineCount(card) != lineCount(before)+1 {
					t.Fatal("heading adds exactly one line")
				}
				if compact && lineCount(before) != 4 {
					t.Fatal("heading-free compact card keeps two content lines")
				}
				lines := strings.Split(ansi.Strip(card), "\n")
				for i, line := range lines {
					if ansi.StringWidth(line) != width {
						t.Fatalf("wrong card width: %q", line)
					}
					if strings.Contains(line, task.File) && !strings.Contains(lines[i+1], task.Heading) {
						t.Fatal("heading must follow filename")
					}
				}
				task.Heading = strings.Repeat("Long heading ", 10)
				if lineCount(m.card(task, width, selected)) != lineCount(card) {
					t.Fatal("heading must truncate, never wrap")
				}
			}
		}
	}
}

func TestMixedCardHeightsScrollAndMouse(t *testing.T) {
	m := newTestModel(t)
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m.prefs.FiltersCollapsed = true
	tasks := []board.Task{
		{File: "first.md", Heading: "First heading"},
		{File: "second.md"},
		{File: "third.md", Heading: "Third heading"},
		{File: "fourth.md"},
		{File: "fifth.md", Heading: "Fifth heading"},
	}
	for i := range tasks {
		tasks[i].Status = "TODO"
		order := float64(i)
		tasks[i].Order = &order
	}
	m.snap.Tasks = tasks
	m.selected["TODO"] = tasks[0].File
	m.View()
	var third region
	for _, r := range m.layout.hits {
		if r.h.file == "third.md" {
			third = r
		}
	}
	if third.h.file == "" {
		t.Fatal("third card must be visible")
	}
	// Click the last content line, which moves down when earlier cards have headings.
	m.Update(tea.MouseClickMsg{X: third.r.x + 1, Y: third.r.y + third.r.h - 2, Button: tea.MouseLeft})
	if m.currentFile() != "third.md" {
		t.Fatal("mixed-height click must select third card")
	}
	for _, compact := range []bool{true, false} {
		m.prefs.Compact = compact
		for _, task := range tasks {
			m.selected["TODO"] = task.File
			m.View()
			found := false
			for _, r := range m.layout.hits {
				if r.h.kind == hitCard && r.h.file == task.File {
					found = true
					if r.r.h != lineCount(m.card(task, r.r.w, true)) {
						t.Fatalf("selected card clipped: %s", task.File)
					}
				}
			}
			if !found {
				t.Fatalf("selected card hidden: %s", task.File)
			}
		}
	}
}

func TestTableHeadingColumn(t *testing.T) {
	m := newTestModel(t)
	m.snap.Tasks[0].Heading = "Readable heading"
	fields := m.tableFields(m.snap.Tasks[0])
	if ansi.Strip(fields[0]) != "a.md" || ansi.Strip(fields[1]) != "Readable heading" {
		t.Fatalf("fields: %v", fields)
	}
	for _, width := range []int{80, 160} {
		m.Update(tea.WindowSizeMsg{Width: width, Height: 24})
		header := strings.Split(ansi.Strip(m.tableView()), "\n")[0]
		if !strings.HasPrefix(header, "FILE") || strings.Index(header, "TITLE") < (width*2+4)/5 {
			t.Fatalf("FILE floor lost: %q", header)
		}
		if ansi.StringWidth(header) != width {
			t.Fatalf("table width: %q", header)
		}
	}
}
