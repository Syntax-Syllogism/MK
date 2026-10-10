package tui

import (
	"charm.land/lipgloss/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
	"testing"
)

func TestSearchLiveKeepClearAndChips(t *testing.T) {
	m := newTestModel(t)
	press(m, "/")
	press(m, "b")
	if m.filter.Query != "b" || len(m.tableTasks()) != 3 {
		t.Fatal("body search should be live")
	}
	press(m, "enter")
	if m.search != nil || m.filter.Query != "b" {
		t.Fatal("enter must retain query")
	}
	m.filter.Tags["green"] = true
	m.repairSelection()
	if m.currentFile() != "b.md" || len(m.tableTasks()) != 1 {
		t.Fatal("chips AND query; focus nearest visible section")
	}
	press(m, "esc")
	if m.filter.Query != "" || !m.filter.Tags["green"] {
		t.Fatal("escape clears only search")
	}
	press(m, "/")
	press(m, "z")
	press(m, "esc")
	if m.search != nil || m.filter.Query != "" {
		t.Fatal("escape cancels prompt")
	}
	m.filter.Query = "body"
	press(m, "F")
	if m.filter.Active() != 0 {
		t.Fatal("F must clear chips and query")
	}
}
func TestSearchNavigationAndOrder(t *testing.T) {
	m := newTestModel(t)
	m.filter.Query = "body"
	m.repairSelection()
	press(m, "n")
	if m.currentFile() != "c.md" {
		t.Fatal("next in reading order")
	}
	press(m, "n")
	if m.currentFile() != "b.md" {
		t.Fatal("next section")
	}
	press(m, "n")
	if m.currentFile() != "a.md" {
		t.Fatal("wrap")
	}
	press(m, "N")
	if m.currentFile() != "b.md" {
		t.Fatal("previous wraps")
	}
	press(m, "v")
	press(m, "n")
	if m.tableCur != "a.md" {
		t.Fatal("table next in section order")
	}
	m.snap.Tasks[0].Title = "zzalpha"
	m.snap.Tasks[1].Title = "alpha"
	m.snap.Tasks[2].Title = "alpha"
	m.filter.Query = "alpha"
	tasks := m.tableTasks()
	if tasks[0].File != "c.md" || tasks[1].File != "b.md" {
		t.Fatal("score first, section/order ties")
	}
	visible := m.visible("TODO")
	if visible[0].File != "a.md" {
		t.Fatal("board order must not rank search")
	}
}
func TestSearchHighlightUnicode(t *testing.T) {
	m := newTestModel(t)
	highlight := lipgloss.NewStyle().Bold(true).Underline(true).Render
	for _, tt := range []struct {
		query, title, want string
	}{
		{"x", "éax", "éa" + highlight("x")},
		{"éx", "éax", highlight("é") + "a" + highlight("x")},
		{"x", "界🙂ax.md", "界🙂a" + highlight("x") + ".md"},
		{"界x", "é界ax", "é" + highlight("界") + "a" + highlight("x")},
	} {
		t.Run(tt.query+"/"+tt.title, func(t *testing.T) {
			m.filter.Query = tt.query
			got := m.highlightTitle(tt.title)
			if ansi.Strip(got) != tt.title || got != tt.want || tt.want == tt.title {
				t.Fatalf("highlightTitle(%q) = %q, want %q", tt.title, got, tt.want)
			}
		})
	}
}
func TestGrabUnderQueryUsesHiddenCards(t *testing.T) {
	m := newTestModel(t)
	content := "needle"
	for _, file := range []string{"a.md", "b.md"} {
		if err := m.store.Apply(board.UpdateRequest{File: file, Content: &content}); err != nil {
			t.Fatal(err)
		}
	}
	writeTask(t, m.store.Dir(), "d.md", "DONE", "[]", "[]")
	reload(t, m)
	m.filter.Query = "needle"
	m.repairSelection()
	press(m, "space")
	press(m, "l")
	press(m, "G")
	press(m, "enter")
	task, _ := m.task("a.md")
	visible, _ := m.task("b.md")
	hidden, _ := m.task("d.md")
	if task.Status != "DONE" || task.Order == nil || visible.Order == nil || hidden.Order == nil || *task.Order <= *visible.Order || *task.Order >= *hidden.Order {
		t.Fatalf("drop must interpolate using hidden neighbour: task=%+v visible=%+v hidden=%+v", task, visible, hidden)
	}
	if m.filter.Query != "needle" {
		t.Fatal("grab must keep query")
	}
}

func TestSearchBodyCacheInvalidation(t *testing.T) {
	m := newTestModel(t)
	old := "needle"
	newContent := "absent"
	if err := m.store.Apply(board.UpdateRequest{File: "a.md", Content: &old}); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	m.filter.Query = "needle"
	if len(m.tableTasks()) != 1 {
		t.Fatal("body hit missing")
	}
	if err := m.store.Apply(board.UpdateRequest{File: "a.md", Content: &newContent}); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if len(m.tableTasks()) != 0 {
		t.Fatal("same-length body edit must invalidate cache")
	}
}
