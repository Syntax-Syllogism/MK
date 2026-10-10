package tui

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
)

type tagState struct {
	originalHash   [32]byte
	file           string
	original, tags []string
	input          textinput.Model
	list           bool
	cursor         int
}

func (m *Model) startTags(file string) tea.Cmd {
	if !m.writable(file) {
		return nil
	}
	t, _ := m.task(file)
	data, err := os.ReadFile(filepath.Join(m.snap.Dir, file))
	if err != nil {
		m.actionError(err)
		return nil
	}
	input := textinput.New()
	input.Prompt = "add tag: "
	input.SetWidth(max(1, min(40, m.width-8)))
	m.tags = &tagState{originalHash: sha256.Sum256(data), file: file, original: slices.Clone(t.Tags), tags: append([]string{}, t.Tags...), input: input}
	return m.tags.input.Focus()
}
func (m *Model) updateTags(msg tea.KeyPressMsg) tea.Cmd {
	state := m.tags
	switch msg.String() {
	case "ctrl+c":
		m.tags = nil
		return nil
	case "esc":
		if !slices.Equal(state.original, state.tags) {
			data, err := os.ReadFile(filepath.Join(m.snap.Dir, state.file))
			if err != nil || sha256.Sum256(data) != state.originalHash {
				m.actionError(fmt.Errorf("can't edit tags: %s changed on disk", state.file))
				m.tags = nil
				return nil
			}
			if !m.writable(state.file) {
				return nil
			}
			if !m.applyAction(state.file, "tags", []board.UpdateRequest{{File: state.file, Tags: state.tags, ExpectedHash: &state.originalHash}}) {
				return nil
			}
		}
		m.tags = nil
		return nil
	case "tab", "shift+tab":
		state.list = !state.list
		if state.list {
			state.input.Blur()
			return nil
		}
		return state.input.Focus()
	}
	if state.list {
		switch msg.String() {
		case "j", "down":
			state.cursor = max(0, min(len(state.tags)-1, state.cursor+1))
		case "k", "up":
			state.cursor = max(0, state.cursor-1)
		case "d", "x", "delete":
			if len(state.tags) > 0 {
				state.tags = slices.Delete(state.tags, state.cursor, state.cursor+1)
				state.cursor = max(0, min(state.cursor, len(state.tags)-1))
			}
		}
		return nil
	}
	if msg.String() == "enter" {
		for _, value := range strings.Split(state.input.Value(), ",") {
			tag := strings.ToLower(strings.TrimSpace(value))
			if tag == "" {
				continue
			}
			if slices.Contains(state.tags, tag) {
				m.notify("already tagged")
				continue
			}
			state.tags = append(state.tags, tag)
		}
		state.input.SetValue("")
		return nil
	}
	var cmd tea.Cmd
	state.input, cmd = state.input.Update(msg)
	return cmd
}
func (m *Model) tagsView() string {
	state := m.tags
	lines := []string{"TAGS · " + state.file, state.input.View()}
	for i, tag := range state.tags {
		prefix := "  "
		if state.list && i == state.cursor {
			prefix = "› "
		}
		lines = append(lines, prefix+tag)
	}
	lines = append(lines, "tab input/list · d remove · esc save · ctrl+c cancel")
	return strings.Join(lines, "\n")
}
