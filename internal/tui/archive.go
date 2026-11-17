package tui

import (
	"crypto/sha256"
	"os"
)

func (m *Model) archiveCard(file string) {
	if !m.writableFor(file, "archive") {
		return
	}
	path, err := m.store.TaskPath(file)
	if err != nil {
		m.actionError(err)
		return
	}
	data, err := os.ReadFile(path)
	if err != nil {
		m.actionError(err)
		return
	}
	hash := sha256.Sum256(data)
	if _, err := m.store.Archive(file, hash); err != nil {
		m.actionError(err)
		return
	}
	// Dependency navigation can preview a different card from the board selection.
	// Capture its position before reloading so the selection goes to its neighbour.
	m.focusFile(file)
	m.undo = append(m.undo, undoEntry{file: file, label: "archive", archived: &archivedUndo{file, hash}})
	if len(m.undo) > 50 {
		m.undo = m.undo[len(m.undo)-50:]
	}
	if m.mode == modePreview {
		m.mode = m.prevMode
		m.preview.back = nil
		m.preview.find = nil
	}
	if err := m.store.Scan(); err != nil {
		m.actionError(err)
		return
	}
	m.reload()
	m.notify("archived " + file + " (u to undo)")
}
