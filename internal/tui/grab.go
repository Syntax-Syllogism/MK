package tui

import (
	"fmt"
	"slices"

	"github.com/Syntax-Syllogism/mk/internal/board"
)

type grabState struct {
	file, origSec, targetSec string
	origIndex, targetIdx     int
}

func (m *Model) writable(file string) bool {
	t, ok := m.task(file)
	if !ok {
		return false
	}
	if t.Status == board.StatusConflict {
		m.actionError(fmt.Errorf("can't move a conflicted card, press e to fix it"))
		return false
	}
	if t.ParseError != "" {
		m.actionError(fmt.Errorf("can't move a card with a parse error, press e to fix it"))
		return false
	}
	return true
}
func withoutFile(tasks []board.Task, file string) []board.Task {
	result := make([]board.Task, 0, len(tasks))
	for _, t := range tasks {
		if t.File != file {
			result = append(result, t)
		}
	}
	return result
}
func (m *Model) fullColumn(section, exclude string) []board.Task {
	var tasks []board.Task
	for _, t := range m.snap.Tasks {
		if t.File != exclude && board.Section(t, m.snap.Columns) == section {
			tasks = append(tasks, t)
		}
	}
	slices.SortFunc(tasks, board.ByOrder)
	return tasks
}
func lastFile(tasks []board.Task) string {
	if len(tasks) == 0 {
		return ""
	}
	return tasks[len(tasks)-1].File
}
func (m *Model) startGrab() {
	file := m.currentFile()
	if !m.writable(file) || len(m.snap.Columns) == 0 {
		return
	}
	t, _ := m.task(file)
	section := board.Section(t, m.snap.Columns)
	index := fileIndex(m.visible(section), file)
	m.grab = &grabState{file: file, origSec: section, origIndex: index, targetSec: section, targetIdx: index}
	if section == board.StatusOther {
		m.grab.targetSec = m.snap.Columns[0]
		m.grab.targetIdx = len(m.visible(m.grab.targetSec))
	}
	m.focusGhost()
}
func (m *Model) focusGhost() {
	for i, section := range m.sections {
		if section.Name == m.grab.targetSec {
			m.focusSec = i
			return
		}
	}
}
func (m *Model) refreshGrab() {
	if m.grab == nil {
		return
	}
	g := m.grab
	t, ok := m.task(g.file)
	if !ok || t.Status == board.StatusConflict || t.ParseError != "" || len(m.snap.Columns) == 0 {
		m.grab = nil
		m.actionError(fmt.Errorf("move cancelled: %s changed on disk", g.file))
		return
	}
	if !slices.Contains(m.snap.Columns, g.targetSec) {
		g.targetSec = m.snap.Columns[0]
	}
	g.targetIdx = min(g.targetIdx, len(withoutFile(m.visible(g.targetSec), g.file)))
	m.focusGhost()
}
func (m *Model) updateGrab(key string) {
	g := m.grab
	n := len(withoutFile(m.visible(g.targetSec), g.file))
	switch key {
	case "esc", "ctrl+c":
		m.grab = nil
		m.focusFile(g.file)
		return
	case "h", "left", "l", "right":
		i := slices.Index(m.snap.Columns, g.targetSec)
		if key == "h" || key == "left" {
			i--
		} else {
			i++
		}
		i = max(0, min(i, len(m.snap.Columns)-1))
		g.targetSec = m.snap.Columns[i]
		g.targetIdx = min(g.targetIdx, len(withoutFile(m.visible(g.targetSec), g.file)))
	case "j", "down":
		g.targetIdx = min(n, g.targetIdx+1)
	case "k", "up":
		g.targetIdx = max(0, g.targetIdx-1)
	case "g":
		g.targetIdx = 0
	case "G":
		g.targetIdx = n
	case "enter":
		t, _ := m.task(g.file)
		if board.Section(t, m.snap.Columns) == g.targetSec && fileIndex(m.visible(g.targetSec), g.file) == g.targetIdx {
			m.grab = nil
			m.focusFile(g.file)
			return
		}
		visible := withoutFile(m.visible(g.targetSec), g.file)
		before := ""
		if g.targetIdx < len(visible) {
			before = visible[g.targetIdx].File
		}
		if m.drop(g.file, g.targetSec, before, lastFile(visible)) {
			m.grab = nil
		}
	}
	if m.grab != nil {
		m.focusGhost()
	}
}
func (m *Model) drop(file, section, before, last string) bool {
	t, ok := m.task(file)
	if !ok {
		return false
	}
	order, updates := board.DropOrder(m.fullColumn(section, file), before, last)
	if t.Status == section && t.Order != nil && *t.Order == order {
		return true
	}
	request := board.UpdateRequest{File: file, Order: &order}
	if t.Status != section {
		request.Status = section
	}
	updates = append(updates, request)
	return m.applyAction(file, "moved to "+section, updates)
}
func (m *Model) quickMove(key string) {
	file := m.currentFile()
	if !m.writable(file) || len(m.snap.Columns) == 0 {
		return
	}
	t, _ := m.task(file)
	section := board.Section(t, m.snap.Columns)
	i := slices.Index(m.snap.Columns, section)
	if key == "H" || key == "L" {
		if i < 0 {
			if key == "L" {
				i = 0
			} else {
				i = len(m.snap.Columns) - 1
			}
		} else {
			if key == "H" {
				i--
			} else {
				i++
			}
			if i < 0 || i >= len(m.snap.Columns) {
				return
			}
		}
		target := m.snap.Columns[i]
		m.drop(file, target, "", lastFile(m.fullColumn(target, file)))
		return
	}
	if i < 0 {
		m.actionError(fmt.Errorf("can't reorder a card in OTHER"))
		return
	}
	visible := withoutFile(m.visible(section), file)
	index := fileIndex(m.visible(section), file)
	if key == "K" {
		index--
	} else {
		index++
	}
	if index < 0 || index > len(visible) {
		return
	}
	before := ""
	if index < len(visible) {
		before = visible[index].File
	}
	m.drop(file, section, before, lastFile(visible))
}
