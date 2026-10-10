package tui

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/Syntax-Syllogism/mk/internal/board"
	"gopkg.in/yaml.v3"
)

type undoFile struct {
	file      string
	before    []byte
	keys      map[string]yaml.Node
	afterHash [32]byte
}
type undoEntry struct {
	file, label string
	files       []undoFile
}

func (m *Model) actionError(err error) {
	text := err.Error()
	if errors.Is(err, board.ErrEmptyClobber) {
		text = "refused: write would empty a non-empty file"
	}
	m.toast = text
	m.toastError = true
	m.toastUntil = time.Now().Add(5 * time.Second)
}
func (m *Model) focusFile(file string) {
	if t, ok := m.task(file); ok && !m.filter.Visible(t) {
		m.clearFilters()
		m.repairSelection()
	}
	m.selectBoard(file)
	m.tableCur = selectedFile(m.tableTasks(), file, 0)
}
func (m *Model) applyAction(file, label string, requests []board.UpdateRequest) bool {
	entry := undoEntry{file: file, label: label}
	for i := range requests {
		path := filepath.Join(m.snap.Dir, requests[i].File)
		before, err := os.ReadFile(path)
		if err != nil {
			m.actionError(err)
			return false
		}
		if requests[i].ExpectedHash != nil && sha256.Sum256(before) != *requests[i].ExpectedHash {
			m.actionError(board.ErrChanged)
			return false
		}
		current, err := board.ParseTask(path)
		if err != nil {
			m.actionError(err)
			return false
		}
		if current.Status == board.StatusConflict {
			m.actionError(fmt.Errorf("can't move a conflicted card, press e to fix it"))
			return false
		}
		if current.ParseError != "" {
			m.actionError(fmt.Errorf("can't move a card with a parse error, press e to fix it"))
			return false
		}
		keys, err := board.FrontmatterKeys(path)
		if err != nil {
			m.actionError(err)
			return false
		}
		hash := sha256.Sum256(before)
		requests[i].ExpectedHash = &hash
		entry.files = append(entry.files, undoFile{file: requests[i].File, before: before, keys: keys})
	}
	hashes, err := m.store.ApplyBatchWithHashes(requests)
	if err != nil {
		m.actionError(err)
		return false
	}
	for i := range entry.files {
		entry.files[i].afterHash = hashes[i]
	}
	m.undo = append(m.undo, entry)
	if len(m.undo) > 50 {
		m.undo = m.undo[len(m.undo)-50:]
	}
	m.reload()
	m.focusFile(file)
	if m.mode == modePreview {
		m.refreshPreview()
	}
	m.notify(label)
	return true
}
func (m *Model) undoAction() {
	if len(m.undo) == 0 {
		return
	}
	entry := m.undo[len(m.undo)-1]
	requests := make([]board.UpdateRequest, 0, len(entry.files))
	for _, file := range entry.files {
		data, err := os.ReadFile(filepath.Join(m.snap.Dir, file.file))
		if err != nil || sha256.Sum256(data) != file.afterHash {
			m.undo = m.undo[:len(m.undo)-1]
			m.actionError(fmt.Errorf("can't undo: %s changed since", file.file))
			return
		}
		hash := file.afterHash
		requests = append(requests, board.UpdateRequest{File: file.file, Restore: file.before, ExpectedHash: &hash})
	}
	if err := m.store.ApplyBatch(requests); err != nil {
		m.actionError(err)
		return
	}
	m.undo = m.undo[:len(m.undo)-1]
	m.reload()
	m.focusFile(entry.file)
	m.notify("undid " + entry.label)
}
