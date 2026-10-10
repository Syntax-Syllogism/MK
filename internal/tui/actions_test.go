package tui

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
)

func actionModel(t *testing.T, files map[string]string) *Model {
	t.Helper()
	// Start with a scratch copy of the project's QA fixtures, then replace tasks
	// with small scenarios whose full and visible orders can be asserted exactly.
	m := sampleModel(t)
	dir := m.store.Dir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".md") {
			if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil {
				t.Fatal(err)
			}
		}
	}
	for file, data := range files {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(data), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, ".kanban.yml"), []byte("columns: [TODO, IN PROGRESS, DONE]\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := m.store.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	m.toastUntil = m.toastUntil.Add(-10000000000)
	m.mode = modeBoard
	m.selectBoard("a.md")
	return m
}
func taskText(status, order, tags string) string {
	return "---\n# retain on undo\nstatus: " + status + "\n" + order + "tags: " + tags + "\nproject: mk\n---\n\nBody **unchanged**.\n"
}
func readActionFile(t *testing.T, m *Model, file string) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(m.store.Dir(), file))
	if err != nil {
		t.Fatal(err)
	}
	return data
}
func assertOrder(t *testing.T, m *Model, file string, value float64) {
	t.Helper()
	task, ok := m.task(file)
	if !ok || task.Order == nil || *task.Order != value {
		t.Fatalf("%s: order %v, want %v", file, task.Order, value)
	}
}
func TestActionsGrabMoveAndUndo(t *testing.T) {
	m := actionModel(t, map[string]string{
		"a.md": taskText("TODO", "", "[blue]"),
		"b.md": taskText("IN PROGRESS", "order: 8\n", "[]"),
	})
	before := readActionFile(t, m, "a.md")
	press(m, "space")
	press(m, "l")
	press(m, "G")
	press(m, "enter")
	task, _ := m.task("a.md")
	if task.Status != "IN PROGRESS" || m.currentFile() != "a.md" || m.grab != nil || len(m.undo) != 1 {
		t.Fatal("move/focus/undo failed")
	}
	assertOrder(t, m, "a.md", 9)
	press(m, "u")
	if !bytes.Equal(before, readActionFile(t, m, "a.md")) {
		t.Fatal("undo must restore exact bytes including absent order and comments")
	}
}
func TestActionsGrabCancelAndNoop(t *testing.T) {
	for _, finish := range []string{"esc", "enter", "ctrl+c"} {
		t.Run(finish, func(t *testing.T) {
			m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
			before := readActionFile(t, m, "a.md")
			press(m, "space")
			press(m, "q")
			if m.grab == nil {
				t.Fatal("q cancelled grab")
			}
			press(m, "h")
			if m.grab.targetSec != "TODO" {
				t.Fatal("left must clamp")
			}
			press(m, finish)
			if m.grab != nil || len(m.undo) != 0 || !bytes.Equal(before, readActionFile(t, m, "a.md")) {
				t.Fatal("cancel/no-op wrote")
			}
		})
	}
}
func TestActionsFilteredGrabAndQuickReorder(t *testing.T) {
	for _, grab := range []bool{false, true} {
		t.Run(map[bool]string{true: "grab", false: "quick"}[grab], func(t *testing.T) {
			m := actionModel(t, map[string]string{
				"a.md": taskText("TODO", "order: 1\n", "[shown]"),
				"b.md": taskText("TODO", "order: 2\n", "[hidden]"),
				"c.md": taskText("TODO", "order: 3\n", "[shown]"),
				"d.md": taskText("TODO", "order: 5\n", "[hidden]"),
			})
			m.filter.Tags["shown"] = true
			m.repairSelection()
			if grab {
				press(m, "space")
				press(m, "j")
				press(m, "enter")
			} else {
				press(m, "J")
			}
			assertOrder(t, m, "a.md", 4)
			assertOrder(t, m, "b.md", 2)
			assertOrder(t, m, "d.md", 5)
		})
	}
}
func TestActionsGrabBetweenNeighbours(t *testing.T) {
	m := actionModel(t, map[string]string{
		"a.md": taskText("TODO", "order: 1\n", "[]"),
		"b.md": taskText("TODO", "order: 2\n", "[]"),
		"c.md": taskText("TODO", "order: 4\n", "[]"),
		"d.md": taskText("TODO", "order: 8\n", "[]"),
	})
	press(m, "space")
	press(m, "j")
	press(m, "j")
	press(m, "enter")
	assertOrder(t, m, "a.md", 6)
}
func TestActionsUnorderedTailUndo(t *testing.T) {
	m := actionModel(t, map[string]string{
		"a.md": taskText("TODO", "", "[]"),
		"b.md": taskText("IN PROGRESS", "", "[]"),
		"c.md": taskText("IN PROGRESS", "", "[]"),
	})
	before := map[string][]byte{}
	for _, file := range []string{"a.md", "b.md", "c.md"} {
		before[file] = readActionFile(t, m, file)
	}
	press(m, "L")
	assertOrder(t, m, "b.md", 1)
	assertOrder(t, m, "c.md", 2)
	assertOrder(t, m, "a.md", 3)
	if len(m.undo) != 1 || len(m.undo[0].files) != 3 {
		t.Fatal("must group tail into one undo")
	}
	for _, file := range m.undo[0].files {
		if file.afterHash != sha256.Sum256(readActionFile(t, m, file.file)) {
			t.Fatalf("%s undo guard does not match written bytes", file.file)
		}
	}
	press(m, "u")
	for file, original := range before {
		if !bytes.Equal(original, readActionFile(t, m, file)) {
			t.Fatalf("%s not restored", file)
		}
	}
}
func TestActionsQuickBoundsAndOther(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("DONE", "order: 3\n", "[]"), "other.md": taskText("BLOCKED", "", "[]")})
	before := readActionFile(t, m, "a.md")
	press(m, "L")
	press(m, "J")
	if !bytes.Equal(before, readActionFile(t, m, "a.md")) || len(m.undo) > 0 {
		t.Fatal("bounds wrote")
	}
	m.selectBoard("other.md")
	press(m, "L")
	task, _ := m.task("other.md")
	if task.Status != "TODO" {
		t.Fatal("OTHER L must enter first column")
	}
	press(m, "u")
	m.selectBoard("other.md")
	press(m, "H")
	task, _ = m.task("other.md")
	if task.Status != "DONE" {
		t.Fatal("OTHER H must enter last column")
	}
}
func TestActionsDiagnosticGuardsAndReload(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]"), "bad.md": "---\norder: soon\n---\nBody", "conflict.md": "<<<<<<< ours\n=======\n>>>>>>> theirs"})
	for _, file := range []string{"bad.md", "conflict.md"} {
		m.selectBoard(file)
		press(m, "space")
		if m.grab != nil || !strings.Contains(m.toast, "press e to fix it") {
			t.Fatal("missing diagnostic grab guard")
		}
		press(m, "t")
		if m.tags != nil {
			t.Fatal("diagnostic tag popup opened")
		}
		if m.editFile(file) == nil {
			t.Fatal("editor must allow diagnostics")
		}
	}
	m.selectBoard("a.md")
	press(m, "space")
	press(m, "l")
	press(m, "l")
	if m.grab.targetSec != "DONE" {
		t.Fatal("grab did not skip diagnostic sections")
	}
	if err := os.Remove(filepath.Join(m.store.Dir(), "a.md")); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if m.grab != nil || !strings.Contains(m.toast, "move cancelled") {
		t.Fatal("reload must cancel")
	}
}
func TestActionsTags(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[blue]")})
	original := readActionFile(t, m, "a.md")
	press(m, "t")
	m.tags.input.SetValue("  Foo , blue, bar")
	press(m, "enter")
	if !slices.Equal(m.tags.tags, []string{"blue", "foo", "bar"}) || m.toast != "already tagged" {
		t.Fatal("normalization/dedupe")
	}
	press(m, "esc")
	task, _ := m.task("a.md")
	if !slices.Equal(task.Tags, []string{"blue", "foo", "bar"}) {
		t.Fatal("tags not saved")
	}
	press(m, "u")
	if !bytes.Equal(original, readActionFile(t, m, "a.md")) {
		t.Fatal("tag undo")
	}
	press(m, "t")
	press(m, "tab")
	press(m, "d")
	press(m, "esc")
	data := readActionFile(t, m, "a.md")
	if !strings.Contains(string(data), "tags: []") || strings.SplitN(string(data), "---", 3)[2] != strings.SplitN(string(original), "---", 3)[2] {
		t.Fatal("empty tags/body preservation")
	}
	press(m, "t")
	m.tags.input.SetValue("cancel")
	press(m, "enter")
	press(m, "ctrl+c")
	if !bytes.Equal(data, readActionFile(t, m, "a.md")) {
		t.Fatal("tag cancel wrote")
	}
}
func TestActionsTagsExternalEdit(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "t")
	m.tags.input.SetValue("foo")
	press(m, "enter")
	external := append(readActionFile(t, m, "a.md"), []byte("external\n")...)
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "a.md"), external, 0600); err != nil {
		t.Fatal(err)
	}
	press(m, "esc")
	if !bytes.Equal(external, readActionFile(t, m, "a.md")) || !strings.Contains(m.toast, "changed on disk") {
		t.Fatal("stale popup overwrote external edit")
	}
}
func TestActionsUndoExternalAndCap(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "L")
	external := append(readActionFile(t, m, "a.md"), []byte("external\n")...)
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "a.md"), external, 0600); err != nil {
		t.Fatal(err)
	}
	press(m, "u")
	if len(m.undo) != 0 || !strings.Contains(m.toast, "changed since") || !bytes.Equal(external, readActionFile(t, m, "a.md")) {
		t.Fatal("unsafe undo")
	}
	reload(t, m)
	for i := 0; i < 55; i++ {
		if i%2 == 0 {
			press(m, "H")
		} else {
			press(m, "L")
		}
	}
	if len(m.undo) != 50 {
		t.Fatalf("cap=%d", len(m.undo))
	}
}
func TestActionsTableAndPreview(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "v")
	press(m, "space")
	press(m, "L")
	if m.grab != nil || len(m.undo) != 0 {
		t.Fatal("table has geometry bindings")
	}
	press(m, "t")
	m.tags.input.SetValue("table")
	press(m, "enter")
	press(m, "esc")
	press(m, "u")
	press(m, "enter")
	press(m, "t")
	m.tags.input.SetValue("preview")
	press(m, "enter")
	press(m, "esc")
	if !slices.Contains(m.preview.task.Tags, "preview") {
		t.Fatal("preview not refreshed")
	}
}
func TestEditorCommand(t *testing.T) {
	for _, tc := range []struct {
		visual, editor, goos string
		want                 []string
	}{
		{"nvim -u NONE", "ignored", "linux", []string{"nvim", "-u", "NONE", "/tmp/a.md"}},
		{"", "code --wait", "linux", []string{"code", "--wait", "/tmp/a.md"}},
		{"", "", "linux", []string{"vi", "/tmp/a.md"}},
		{`"/path with space/nvim" -u NONE`, "", "linux", []string{"/path with space/nvim", "-u", "NONE", "/tmp/a.md"}},
		{"", "", "windows", []string{"notepad", "/tmp/a.md"}},
	} {
		cmd, err := editorCmd(func(key string) string {
			if key == "VISUAL" {
				return tc.visual
			}
			return tc.editor
		}, tc.goos, "/tmp/a.md")
		if err != nil || !slices.Equal(cmd.Args, tc.want) {
			t.Fatalf("command=%v error=%v want=%v", cmd, err, tc.want)
		}
	}
	if _, err := editorCmd(func(string) string { return `"` }, "linux", "a.md"); err == nil {
		t.Fatal("invalid quoting accepted")
	}
}
func TestEditorRefresh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script")
	}
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", `sh -c 'echo x >> "$1"' _`)
	cmd, err := editorCmd(os.Getenv, runtime.GOOS, filepath.Join(m.store.Dir(), "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	m.Update(editorDoneMsg{file: "a.md"})
	task, _ := m.task("a.md")
	if !strings.HasSuffix(task.Content, "x") || len(m.undo) != 0 {
		t.Fatal("editor refresh/undo")
	}
	m.Update(editorDoneMsg{file: "a.md", err: errors.New("exit 1")})
	if !strings.Contains(m.toast, "editor exited") {
		t.Fatal("missing editor error")
	}
}
func TestClipboardStrategy(t *testing.T) {
	for _, tc := range []struct {
		env            map[string]string
		fail, terminal bool
	}{
		{map[string]string{}, false, false}, {map[string]string{}, true, true},
		{map[string]string{"SSH_TTY": "tty"}, false, true},
		{map[string]string{"SSH_CONNECTION": "ssh", "TMUX": "tmux"}, false, true},
	} {
		calls := 0
		text, cmd := copyPath("/tasks/a.md", func(key string) string { return tc.env[key] }, func(string) error {
			calls++
			if tc.fail {
				return errors.New("no clipboard")
			}
			return nil
		})
		if (cmd != nil) != tc.terminal {
			t.Fatal("wrong strategy")
		}
		if tc.env["SSH_TTY"] != "" || tc.env["SSH_CONNECTION"] != "" {
			if calls != 0 {
				t.Fatal("SSH called native clipboard")
			}
		}
		if cmd != nil {
			raw, ok := cmd().(tea.RawMsg)
			if !ok || !strings.Contains(raw.Msg.(string), "52;") {
				t.Fatalf("bad OSC52: %v", raw)
			}
		} else if text != "copied /tasks/a.md" {
			t.Fatal(text)
		}
	}
}
func TestActionsErrorLeavesState(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "space")
	press(m, "l")
	if err := os.Remove(filepath.Join(m.store.Dir(), "a.md")); err != nil {
		t.Fatal(err)
	}
	press(m, "enter")
	if m.grab == nil || !m.toastError || len(m.undo) != 0 {
		t.Fatal("failed apply changed state")
	}
	m.actionError(board.ErrEmptyClobber)
	if m.toast != "refused: write would empty a non-empty file" {
		t.Fatal(m.toast)
	}
}
func TestActionsHashChain(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	before := sha256.Sum256(readActionFile(t, m, "a.md"))
	press(m, "L")
	press(m, "L")
	press(m, "u")
	press(m, "u")
	if len(m.undo) != 0 || sha256.Sum256(readActionFile(t, m, "a.md")) != before {
		t.Fatal("successive undo invalidated its predecessor")
	}
}

func TestActionsGhostRenderingAndReclamp(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "space")
	press(m, "l")
	screen := m.View().Content
	if !strings.Contains(screen, "MOVING a.md") || !strings.Contains(screen, "╔") {
		t.Fatal("missing ghost/status in empty target")
	}
	// A reload preserves grab state while clamping its insertion position.
	m.grab.targetIdx = 99
	reload(t, m)
	if m.grab == nil || m.grab.targetIdx != 0 {
		t.Fatal("reload did not preserve/reclamp grab")
	}
	press(m, "?")
	if !strings.Contains(m.overlay.View(), "enter  drop") {
		t.Fatal("grab help missing")
	}
	press(m, "esc")
	press(m, "esc")
	if m.grab != nil {
		t.Fatal("help close lost cancellation")
	}
}
func TestActionsEmptyTagListNavigation(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	press(m, "t")
	press(m, "tab")
	press(m, "j")
	press(m, "tab")
	m.tags.input.SetValue("foo")
	press(m, "enter")
	press(m, "tab")
	press(m, "d")
	press(m, "esc")
	task, _ := m.task("a.md")
	if len(task.Tags) != 0 {
		t.Fatal("empty list cursor must stay valid")
	}
}
func TestActionsUndoTailExternalRefusesWholeAction(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]"), "b.md": taskText("IN PROGRESS", "", "[]")})
	press(m, "L")
	moved := readActionFile(t, m, "a.md")
	external := append(readActionFile(t, m, "b.md"), []byte("external\n")...)
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "b.md"), external, 0600); err != nil {
		t.Fatal(err)
	}
	press(m, "u")
	if !bytes.Equal(moved, readActionFile(t, m, "a.md")) || !bytes.Equal(external, readActionFile(t, m, "b.md")) || len(m.undo) != 0 {
		t.Fatal("undo partially restored a stale multi-file action")
	}
}

func TestActionsDiskDiagnosticAndErrorDuration(t *testing.T) {
	m := actionModel(t, map[string]string{"a.md": taskText("TODO", "", "[]")})
	external := []byte("<<<<<<< ours\n=======\n>>>>>>> theirs")
	if err := os.WriteFile(filepath.Join(m.store.Dir(), "a.md"), external, 0600); err != nil {
		t.Fatal(err)
	}
	press(m, "L")
	if !bytes.Equal(external, readActionFile(t, m, "a.md")) || !m.toastError {
		t.Fatal("stale snapshot allowed diagnostic write")
	}
	errorText := m.toast
	m.Update(reloadMsg{})
	if m.toast != errorText || !m.toastError {
		t.Fatal("sync replaced five-second error toast")
	}
}
