package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"github.com/charmbracelet/x/ansi"
)

func findBody(t *testing.T, body string) *Model {
	t.Helper()
	m := newTestModel(t)
	m.mode = modePreview
	m.prevMode = modeBoard
	m.openPreview("a.md")
	m.preview.task.Content = body
	m.renderPreview()
	return m
}
func findQuery(m *Model, query string) {
	m.preview.find.input.SetValue(query)
	m.updatePreviewFindInput(nil)
}
func assertFindGeometry(t *testing.T, m *Model) {
	t.Helper()
	f := m.preview.find
	if ansi.Strip(m.previewFindBody()) != ansi.Strip(m.preview.rendered) {
		t.Fatal("highlighting changed displayed text")
	}
	if f.active >= len(f.matches) {
		t.Fatal("invalid active index")
	}
}
func TestPreviewFindInputAndNavigation(t *testing.T) {
	for _, opener := range []string{"/", "ctrl+f"} {
		t.Run(opener, func(t *testing.T) {
			m := findBody(t, "Hello hello HELLO. q r ! e t y /\n")
			m.filter.Query = "kept"
			m.filter.Tags["blue"] = true
			press(m, opener)
			if m.preview.find == nil || m.search != nil {
				t.Fatal("opener did not start independent find")
			}
			for _, r := range "hello" {
				press(m, string(r))
			}
			f := m.preview.find
			if len(f.matches) != 3 || f.active != 0 {
				t.Fatalf("matches=%d active=%d", len(f.matches), f.active)
			}
			for _, want := range []int{1, 2, 0} {
				press(m, "enter")
				if f.active != want {
					t.Fatal("forward wrap")
				}
			}
			m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
			if f.active != 2 {
				t.Fatal("backward wrap")
			}
			assertFindGeometry(t, m)
			press(m, "?")
			if m.mode != modeHelp || m.helpGroup() != "Preview find" {
				t.Fatal("find help missing")
			}
			m.Update(tea.WindowSizeMsg{Width: 32, Height: 12})
			press(m, "esc")
			if m.mode != modePreview || f.query != "hello" || !f.input.Focused() {
				t.Fatal("help lost find")
			}
			findQuery(m, "")
			for _, r := range "qr!ety/" {
				press(m, string(r))
			}
			if f.query != "qr!ety/" || m.tags != nil || m.mode != modePreview {
				t.Fatal("action keys escaped input")
			}
			if !strings.Contains(m.previewFindStatus(), "no matches") {
				t.Fatal("unmatched feedback")
			}
			press(m, "enter")
			m.preview.back = []string{"b.md"}
			press(m, "esc")
			if m.preview.find != nil || len(m.preview.back) != 1 || m.mode != modePreview {
				t.Fatal("escape consumed history")
			}
			if m.filter.Query != "kept" || !m.filter.Tags["blue"] {
				t.Fatal("find modified board filters")
			}
			press(m, "tab")
			press(m, "enter")
			if m.preview.task.File != "b.md" {
				t.Fatal("dependency follow not restored")
			}
		})
	}
}
func TestPreviewFindRenderedTextAndSoftWrap(t *testing.T) {
	body := "# Heading\n\n**Bold** and _emphasis_. `Code` [label](https://example.org)\n\n" + strings.Repeat("prefix ", 12) + "needle phrase " + strings.Repeat("suffix ", 12) + "\n\nSecond paragraph.\n"
	m := findBody(t, body)
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 12})
	press(m, "/")
	for query, want := range map[string]int{"Heading": 1, "Bold and emphasis.": 1, "**": 0, "_emphasis_": 0, "needle phrase": 1, "Code": 1, "label": 1, "phrase suffix": 1, "paragraph. Second": 0} {
		findQuery(m, query)
		if got := len(m.preview.find.matches); got != want {
			t.Errorf("%q: got %d want %d", query, got, want)
		}
		assertFindGeometry(t, m)
	}
	findQuery(m, "needle phrase")
	before := m.preview.find.matches[0].start
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if len(m.preview.find.matches) != 1 || m.preview.find.matches[0].start != before {
		t.Fatal("resize lost logical occurrence")
	}
	m.dark = false
	m.preview.cacheKey = ""
	m.renderPreview()
	if len(m.preview.find.matches) != 1 {
		t.Fatal("theme lost matches")
	}
}
func TestPreviewFindUnicodeAndRawHorizontal(t *testing.T) {
	m := findBody(t, "É é Σ σ ς 界 e\u0301 e\u0301\n")
	press(m, "/")
	for query, want := range map[string]int{"é": 2, "Σ": 3, "界": 1, "e\u0301": 2} {
		findQuery(m, query)
		if len(m.preview.find.matches) != want {
			t.Fatalf("%q got %d", query, len(m.preview.find.matches))
		}
		assertFindGeometry(t, m)
	}
	press(m, "esc")
	m.preview.task.Status = board.StatusConflict
	m.preview.task.Content = strings.Repeat("x", 150) + " target target\n" + strings.Repeat("line\n", 40) + "target\n"
	m.renderPreview()
	press(m, "/")
	findQuery(m, "target")
	if len(m.preview.find.matches) != 3 || m.preview.viewport.XOffset() == 0 {
		t.Fatal("raw horizontal match hidden")
	}
	press(m, "enter")
	press(m, "enter")
	f := m.preview.find
	token := f.tokens[f.matches[f.active].tokens[0]]
	if token.line < m.preview.viewport.YOffset() || token.line >= m.preview.viewport.YOffset()+m.preview.viewport.Height() {
		t.Fatal("offscreen active match")
	}
	assertFindGeometry(t, m)
}
func TestPreviewFindReloadRemovedAndReadOnly(t *testing.T) {
	m := newTestModel(t)
	path := filepath.Join(m.snap.Dir, "a.md")
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	press(m, "enter")
	press(m, "/")
	findQuery(m, "bold")
	press(m, "enter")
	m.Update(tea.WindowSizeMsg{Width: 40, Height: 20})
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(original) {
		t.Fatal("find wrote task")
	}
	changed := strings.ReplaceAll(string(original), "Body **bold**.", "bold bold")
	if err := os.WriteFile(path, []byte(changed), 0600); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if len(m.preview.find.matches) != 2 {
		t.Fatal("reload didn't rebuild matches")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	reload(t, m)
	if !m.preview.removed || len(m.preview.find.matches) != 2 {
		t.Fatal("removed content no longer searchable")
	}
}

func TestPreviewFindPasteMouseAndLiveSelection(t *testing.T) {
	m := findBody(t, "first needle\n\n"+strings.Repeat("paragraph.\n\n", 20)+"last needle\n")
	m.preview.viewport.GotoBottom()
	press(m, "/")
	m.Update(tea.PasteMsg{Content: "needle"})
	if len(m.preview.find.matches) != 2 || m.preview.find.active != 1 {
		t.Fatal("paste did not select occurrence below current scroll")
	}
	region := findHit(t, m, hitDependency, "")
	click(m, region)
	if m.preview.task.File != "a.md" || m.preview.find == nil {
		t.Fatal("mouse changed dependency while finding")
	}
	m.Update(tea.WindowSizeMsg{Width: 10, Height: 10})
	if !strings.Contains(ansi.Strip(m.previewFindStatus()), "/2") {
		t.Fatal("narrow status lost count")
	}
	press(m, "esc")
	press(m, "/")
	if m.preview.find.query != "" || len(m.preview.find.matches) != 0 {
		t.Fatal("reopened find retained query")
	}
	findQuery(m, "not present")
	press(m, "enter")
	if m.preview.find.active != -1 {
		t.Fatal("unmatched navigation changed index")
	}
	m.leavePreview()
	if m.preview.find != nil {
		t.Fatal("find leaked onto board")
	}
}
func TestPreviewFindCodeAndParseError(t *testing.T) {
	m := findBody(t, "```go\n"+strings.Repeat("x", 120)+" target\nsecond target\n```\n")
	press(m, "/")
	findQuery(m, "target")
	if len(m.preview.find.matches) != 2 {
		t.Fatal("code matches missing")
	}
	assertFindGeometry(t, m)
	press(m, "esc")
	m.preview.task.ParseError = "invalid YAML"
	m.preview.task.Content = "raw parse body"
	m.renderPreview()
	press(m, "/")
	findQuery(m, "raw parse body")
	if len(m.preview.find.matches) != 1 {
		t.Fatal("parse error body not searchable")
	}
	findQuery(m, "invalid YAML")
	if len(m.preview.find.matches) != 0 {
		t.Fatal("searched header error")
	}
}

func TestPreviewFindWrappedQuote(t *testing.T) {
	m := findBody(t, "> "+strings.Repeat("prefix ", 10)+"needle phrase "+strings.Repeat("suffix ", 10)+"\n")
	m.Update(tea.WindowSizeMsg{Width: 24, Height: 14})
	press(m, "/")
	findQuery(m, "needle phrase")
	if len(m.preview.find.matches) != 1 {
		t.Fatal("wrapped quote phrase missing")
	}
	assertFindGeometry(t, m)
}
