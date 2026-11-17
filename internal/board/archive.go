package board

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func (s *Store) ArchiveDir() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if s.archiveDir == "" {
		return "archive"
	}
	return s.archiveDir
}

// archivePath rejects symlinks that would place the archive outside the board.
func (s *Store) archivePath() (string, error) {
	path := s.dir
	for _, part := range strings.Split(s.ArchiveDir(), string(filepath.Separator)) {
		path = filepath.Join(path, part)
		info, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return "", err
		}
		if err == nil && (info.Mode()&os.ModeSymlink != 0 || !info.IsDir()) {
			return "", fmt.Errorf("archive directory must contain only real directories")
		}
	}
	return path, nil
}

func (s *Store) scanArchived() (map[string]bool, error) {
	path, err := s.archivePath()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	result := map[string]bool{}
	for _, entry := range entries {
		if entry.Type().IsRegular() && strings.HasSuffix(entry.Name(), ".md") {
			result[entry.Name()] = true
		}
	}
	return result, nil
}

// Archive moves an unchanged task into the configured archive directory.
func (s *Store) Archive(file string, expected [32]byte) (string, error) {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	source, err := s.TaskPath(file)
	if err != nil {
		return "", err
	}
	dir, err := s.archivePath()
	if err != nil {
		return "", err
	}
	// Recheck diagnostics on disk so a stale TUI snapshot cannot archive them.
	task, err := ParseTask(source)
	if err != nil {
		return "", err
	}
	if task.Status == StatusConflict || task.ParseError != "" {
		return "", fmt.Errorf("can't archive a conflicted or parse-error card, press e to fix it")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", err
	}
	dest := filepath.Join(dir, file)
	if err := moveUnchanged(source, dest, expected); err != nil {
		return "", err
	}
	return dest, nil
}

// Unarchive restores an unchanged archived task when its original name is free.
func (s *Store) Unarchive(file string, expected [32]byte) error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	dest, err := s.TaskPath(file)
	if err != nil {
		return err
	}
	dir, err := s.archivePath()
	if err != nil {
		return err
	}
	return moveUnchanged(filepath.Join(dir, file), dest, expected)
}

func moveUnchanged(source, dest string, expected [32]byte) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return ErrInvalidFile
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	if sha256.Sum256(data) != expected {
		return ErrChanged
	}
	if _, err := os.Lstat(dest); err == nil {
		return fmt.Errorf("name taken: %s", filepath.Base(dest))
	} else if !os.IsNotExist(err) {
		return err
	}
	// Another process can create dest between Lstat and Rename, which overwrites
	// on POSIX. ioMu serializes MK writes; external writers remain a race.
	return os.Rename(source, dest)
}
