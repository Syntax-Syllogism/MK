package tui

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"
	"charm.land/glamour/v2"
	"charm.land/lipgloss/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
)

type previewState struct {
	task               board.Task
	viewport           viewport.Model
	depCursor          int
	back               []string
	removed            bool
	cacheKey, rendered string
}

func (m *Model) task(file string) (board.Task, bool) {
	for _, t := range m.snap.Tasks {
		if t.File == file {
			return t, true
		}
	}
	return board.Task{}, false
}
func (m *Model) openPreview(file string) {
	t, ok := m.task(file)
	if !ok {
		return
	}
	m.preview.task = t
	m.preview.removed = false
	m.preview.depCursor = -1
	m.preview.viewport.GotoTop()
	m.renderPreview()
}
func (m *Model) refreshPreview() {
	t, ok := m.task(m.preview.task.File)
	m.preview.removed = !ok
	if ok {
		m.preview.task = t
	}
	m.preview.depCursor = min(m.preview.depCursor, len(m.preview.task.DependsOn)-1)
	m.renderPreview()
}
func (m *Model) previewHeader() string { return m.renderPreviewHeader(false) }
func (m *Model) renderPreviewHeader(record bool) string {
	t := m.preview.task
	status := t.Status
	if board.Section(t, m.snap.Columns) == board.StatusOther {
		status = empty(status) + " (OTHER)"
	}
	lines := []string{empty(t.Project) + " · " + epic(t), accent.Bold(true).Render(t.File), "status: " + status, "tags: " + empty(strings.Join(t.Tags, ", "))}
	if t.ParseError != "" {
		lines = append(lines, danger.Render("PARSE ERROR: "+t.ParseError+" — press e to fix this file's YAML frontmatter, then save."))
	}
	if m.preview.removed {
		lines = append(lines, warning.Render("file removed — displaying last content"))
	}
	header := lipWrap(strings.Join(lines[:4], "\n"), m.width)
	deps := m.previewDependencies(record, 1+lineCount(header))
	tail := []string{dim.Render(filepath.Join(m.snap.Dir, t.File))}
	tail = append(tail, lines[4:]...)
	return header + "\n" + deps + "\n" + lipWrap(strings.Join(tail, "\n"), m.width)
}
func (m *Model) renderPreview() {
	t := m.preview.task
	w := max(1, m.width-2)
	offset := m.preview.viewport.YOffset()
	key := fmt.Sprintf("%s:%d:%x:%t:%s", t.File, w, sha256.Sum256([]byte(t.Content)), m.dark, t.Status)
	if key != m.preview.cacheKey {
		text := t.Content
		if t.Status != board.StatusConflict {
			// Glamour v2 replaces WithAutoStyle with explicit terminal background detection.
			style := "dark"
			if !m.dark {
				style = "light"
			}
			renderer, err := glamour.NewTermRenderer(glamour.WithStandardStyle(style), glamour.WithWordWrap(w))
			if err == nil {
				if rendered, err := renderer.Render(text); err == nil {
					text = rendered
				}
			}
		}
		m.preview.cacheKey = key
		m.preview.rendered = text
	}
	m.preview.viewport.SetWidth(m.width)
	m.preview.viewport.SetHeight(max(1, m.height-2-min(m.height-3, lineCount(m.previewHeader()))))
	m.preview.viewport.SetContent(m.preview.rendered)
	m.preview.viewport.SetYOffset(offset)
}
func (m *Model) leavePreview() {
	t := m.preview.task
	if !m.filter.Visible(t) {
		m.clearFilters()
		m.repairSelection()
		m.notify("filters cleared to show " + strings.TrimSuffix(t.File, ".md"))
	}
	m.selectBoard(t.File)
	m.tableCur = selectedFile(m.tableTasks(), t.File, 0)
	m.mode = m.prevMode
	m.preview.back = nil
}
func (m *Model) updatePreview(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "g":
		m.preview.viewport.GotoTop()
		return nil
	case "G":
		m.preview.viewport.GotoBottom()
		return nil
	case "q":
		m.leavePreview()
		return nil
	case "esc", "backspace":
		if len(m.preview.back) > 0 {
			last := m.preview.back[len(m.preview.back)-1]
			m.preview.back = m.preview.back[:len(m.preview.back)-1]
			m.openPreview(last)
		} else {
			m.leavePreview()
		}
		return nil
	case "tab", "shift+tab":
		n := len(m.preview.task.DependsOn)
		if n > 0 {
			if msg.String() == "tab" {
				m.preview.depCursor = (m.preview.depCursor + 1) % n
			} else {
				if m.preview.depCursor < 0 {
					m.preview.depCursor = n - 1
				} else {
					m.preview.depCursor = (m.preview.depCursor - 1 + n) % n
				}
			}
			m.renderPreview()
		}
		return nil
	case "enter":
		i := m.preview.depCursor
		if i >= 0 && i < len(m.preview.task.DependsOn) {
			slug := m.preview.task.DependsOn[i]
			if m.deps.Missing(slug) {
				m.notify("unknown card: " + slug)
			} else {
				m.preview.back = append(m.preview.back, m.preview.task.File)
				m.openPreview(slug + ".md")
			}
		}
		return nil
	}
	var cmd tea.Cmd
	m.preview.viewport, cmd = m.preview.viewport.Update(msg)
	return cmd
}

// Dependencies use the same cell positions for drawing and hit testing.
func (m *Model) previewDependencies(record bool, y int) string {
	line := "deps: "
	x := 6
	var lines []string
	for i, slug := range m.preview.task.DependsOn {
		text := slug
		if m.deps.Missing(slug) {
			text += "?"
		}
		w := lipgloss.Width(text)
		if x+w > m.width && x > 6 {
			lines = append(lines, line)
			line = ""
			x = 0
			y++
		}
		if m.deps.Missing(slug) {
			text = warning.Render(text)
		}
		if i == m.preview.depCursor {
			text = lipgloss.NewStyle().Reverse(true).Render(text)
		}
		if record {
			m.addHit(rect{x, y, min(w, m.width-x), 1}, hit{kind: hitDependency, chip: i})
		}
		line += text + " "
		x += w + 1
	}
	if len(m.preview.task.DependsOn) == 0 {
		line += "-"
	}
	if m.deps.Cycles[m.preview.task.File] {
		line += " " + danger.Render("cycle")
	}
	return strings.Join(append(lines, line), "\n")
}
