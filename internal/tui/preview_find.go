package tui

import (
	"fmt"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
)

type previewFindState struct {
	input   textinput.Model
	query   string
	matches []previewMatch
	active  int
	tokens  []previewToken
}
type previewMatch struct {
	start, end int
	tokens     []int
}
type previewToken struct {
	text              string
	width, line, cell int
	start, end        int
}

func (m *Model) startPreviewFind() tea.Cmd {
	input := textinput.New()
	input.Prompt = "find: "
	input.CharLimit = 0
	m.preview.find = &previewFindState{input: input, active: -1}
	return m.preview.find.input.Focus()
}
func (m *Model) updatePreviewFind(msg tea.KeyPressMsg) tea.Cmd {
	switch msg.String() {
	case "ctrl+c":
		return tea.Quit
	case "ctrl+f":
		return m.preview.find.input.Focus()
	case "esc":
		y, x := m.preview.viewport.YOffset(), m.preview.viewport.XOffset()
		m.preview.find = nil
		m.preview.viewport.SetContent(m.preview.rendered)
		m.preview.viewport.SetYOffset(y)
		m.preview.viewport.SetXOffset(x)
		return nil
	case "enter", "shift+enter":
		f := m.preview.find
		if len(f.matches) > 0 {
			step := 1
			if msg.String() == "shift+enter" {
				step = -1
			}
			f.active = (f.active + step + len(f.matches)) % len(f.matches)
			m.highlightPreviewFind()
			m.revealPreviewMatch()
		}
		return nil
	}
	return m.updatePreviewFindInput(msg)
}
func (m *Model) updatePreviewFindInput(msg tea.Msg) tea.Cmd {
	f := m.preview.find
	var cmd tea.Cmd
	f.input, cmd = f.input.Update(msg)
	if f.query != f.input.Value() {
		f.query = f.input.Value()
		m.rebuildPreviewFind(true)
	}
	return cmd
}

// Decode ANSI and whole graphemes separately, retaining exact display bytes and cells.
func previewTokens(body string) ([]previewToken, []rune) {
	var tokens []previewToken
	var plain []rune
	line, cell := 0, 0
	var state byte
	for body != "" {
		seq, width, n, next := ansi.DecodeSequence(body, state, nil)
		state = next
		if n == 0 {
			break
		}
		token := previewToken{text: seq, width: width, line: line, cell: cell, start: len(plain)}
		if !strings.HasPrefix(seq, "\x1b") {
			plain = append(plain, []rune(seq)...)
			if seq == "\n" {
				line++
				cell = 0
			} else {
				cell += width
			}
		}
		token.end = len(plain)
		tokens = append(tokens, token)
		body = body[n:]
	}
	return tokens, plain
}

// Wrapping changes whitespace and margins, but preserves visible nonspace text.
// Ignore repeated box borders in wrapped quotes and tables when verifying
// alignment. Other width-sensitive text changes fall back to displayed text.
func previewLogicalPositions(logical string, display []rune) ([]rune, []int) {
	lines := strings.Split(ansi.Strip(logical), "\n")
	for i := range lines {
		lines[i] = strings.TrimSpace(lines[i])
	}
	text := []rune(strings.Join(lines, "\n"))
	nonspace := func(rs []rune) string {
		var out strings.Builder
		for _, r := range rs {
			if !unicode.IsSpace(r) && !previewBorder(r) {
				out.WriteRune(r)
			}
		}
		return out.String()
	}
	if nonspace(text) != nonspace(display) {
		text = display
		positions := make([]int, len(text))
		for i := range positions {
			positions[i] = i
		}
		return text, positions
	}
	positions := make([]int, len(text))
	j := 0
	for i, r := range text {
		positions[i] = -1
		if unicode.IsSpace(r) {
			if j < len(display) && unicode.IsSpace(display[j]) {
				positions[i] = j
				j++
			}
			continue
		}
		for j < len(display) && (unicode.IsSpace(display[j]) || (previewBorder(display[j]) && display[j] != r)) {
			j++
		}
		if j < len(display) && display[j] == r {
			positions[i] = j
			j++
		}
	}
	return text, positions
}
func previewBorder(r rune) bool {
	return strings.ContainsRune("│─┌┐└┘├┤┬┴┼╭╮╯╰", r)
}

func (m *Model) rebuildPreviewFind(selectFromScroll bool) {
	f := m.preview.find
	oldPosition := -1
	if f.active >= 0 && f.active < len(f.matches) {
		oldPosition = f.matches[f.active].start
	}
	var display []rune
	f.tokens, display = previewTokens(m.preview.rendered)
	logical, positions := previewLogicalPositions(m.preview.logical, display)
	// Raw previews retain intentional indentation and line boundaries exactly.
	if m.preview.logical == m.preview.rendered && !strings.Contains(m.preview.rendered, "\x1b") {
		logical = display
		positions = make([]int, len(display))
		for i := range positions {
			positions[i] = i
		}
	}
	runeToken := make([]int, len(display))
	for i, token := range f.tokens {
		for j := token.start; j < token.end; j++ {
			runeToken[j] = i
		}
	}
	f.matches = nil
	query := []rune(f.query)
	if len(query) > 0 {
		for i := 0; i+len(query) <= len(logical); {
			equal := true
			for j, r := range query {
				if !strings.EqualFold(string(r), string(logical[i+j])) {
					equal = false
					break
				}
			}
			if !equal {
				i++
				continue
			}
			match := previewMatch{start: i, end: i + len(query)}
			for _, pos := range positions[match.start:match.end] {
				if pos >= 0 {
					ti := runeToken[pos]
					if len(match.tokens) == 0 || match.tokens[len(match.tokens)-1] != ti {
						match.tokens = append(match.tokens, ti)
					}
				}
			}
			if len(match.tokens) > 0 {
				f.matches = append(f.matches, match)
			}
			i += len(query)
		}
	}
	f.active = -1
	if len(f.matches) > 0 {
		f.active = len(f.matches) - 1
		if selectFromScroll {
			f.active = 0
		}
		for i, match := range f.matches {
			if (selectFromScroll && f.tokens[match.tokens[0]].line >= m.preview.viewport.YOffset()) || (!selectFromScroll && match.start >= oldPosition) {
				f.active = i
				break
			}
		}
	}
	m.highlightPreviewFind()
	if f.active >= 0 {
		m.revealPreviewMatch()
	}
}
func (m *Model) previewFindBody() string {
	f := m.preview.find
	highlights := make(map[int]bool)
	for i, match := range f.matches {
		for _, token := range match.tokens {
			highlights[token] = i == f.active
		}
	}
	var out, style strings.Builder
	for i, token := range f.tokens {
		if strings.HasPrefix(token.text, "\x1b[") && strings.HasSuffix(token.text, "m") {
			if token.text == "\x1b[0m" || token.text == "\x1b[m" {
				style.Reset()
			} else {
				style.WriteString(token.text)
			}
		}
		active, marked := highlights[i]
		if marked && token.width >= 0 && token.text != "\n" {
			// Override only background/underline; replay original SGR after each grapheme.
			if active {
				out.WriteString("\x1b[43;4m")
			} else {
				out.WriteString("\x1b[100m")
			}
			out.WriteString(token.text)
			out.WriteString("\x1b[0m")
			out.WriteString(style.String())
		} else {
			out.WriteString(token.text)
		}
	}
	return out.String()
}

func (m *Model) highlightPreviewFind() {
	y, x := m.preview.viewport.YOffset(), m.preview.viewport.XOffset()
	m.preview.viewport.SetContent(m.previewFindBody())
	m.preview.viewport.SetYOffset(y)
	m.preview.viewport.SetXOffset(x)
}
func (m *Model) revealPreviewMatch() {
	f := m.preview.find
	if f.active < 0 {
		return
	}
	match := f.matches[f.active]
	first := f.tokens[match.tokens[0]]
	last := f.tokens[match.tokens[len(match.tokens)-1]]
	v := &m.preview.viewport
	if first.line < v.YOffset() {
		v.SetYOffset(first.line)
	} else if last.line >= v.YOffset()+v.Height() {
		v.SetYOffset(min(first.line, last.line-v.Height()+1))
	}
	if first.cell < v.XOffset() {
		v.SetXOffset(first.cell)
	} else if last.line == first.line && last.cell+last.width > v.XOffset()+v.Width() {
		v.SetXOffset(min(first.cell, last.cell+last.width-v.Width()))
	} else if first.cell >= v.XOffset()+v.Width() {
		v.SetXOffset(first.cell)
	}
}
func (m *Model) previewFindStatus() string {
	f := m.preview.find
	count := fmt.Sprintf(" %d/%d", f.active+1, len(f.matches))
	if f.query != "" && len(f.matches) == 0 {
		count = " no matches"
	}
	// Reserve feedback even when the input needs horizontal scrolling.
	if m.width < ansi.StringWidth(count)+8 {
		return ansi.Truncate(strings.TrimSpace(count), m.width, "")
	}
	f.input.SetWidth(max(1, m.width-ansi.StringWidth(count)-6))
	return ansi.Truncate(f.input.View(), m.width-ansi.StringWidth(count), "") + count
}
