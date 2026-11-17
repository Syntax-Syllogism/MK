package board

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type BoardConfig struct {
	Columns    []string `yaml:"columns"`
	ArchiveDir string   `yaml:"archive_dir,omitempty"`
}

const ConfigFileName = ".kanban.yml"

// bootstrapIfAbsent writes docs/.kanban.yml from the current task set (via
// Scan, which must already have run) if the file does not already exist.
// Called exactly once, from Open, before the watcher is attached — this is
// a startup concern, not something ReloadConfig should redo on every reload.
func (s *Store) bootstrapIfAbsent() error {
	path := filepath.Join(s.dir, ConfigFileName)
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return s.bootstrapConfig(path)
		}
		return err
	}
	return nil
}

// ReloadConfig reads docs/.kanban.yml and swaps it into the in-memory config.
// It is read-only: if the file is transiently absent (an editor's
// atomic-save rename, a git checkout/stash, a sync client), it logs a
// warning and leaves the current in-memory config untouched rather than
// regenerating the file, which would silently discard hand-edited columns.
func (s *Store) ReloadConfig() error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	path := filepath.Join(s.dir, ConfigFileName)

	data, err := ioutil.ReadFile(path)
	if os.IsNotExist(err) {
		s.mu.Lock()
		if len(s.columns) == 0 {
			s.columns = []string{"TODO", "IN PROGRESS", "DONE"}
		}
		s.mu.Unlock()
		s.logf("Config warning: %s is missing; keeping current in-memory columns", path)
		return nil
	}
	if err != nil {
		return err
	}

	var loaded BoardConfig
	if err := yaml.Unmarshal(data, &loaded); err != nil {
		return err
	}
	archive := loaded.ArchiveDir
	if archive == "" {
		archive = "archive"
	}
	valid := !filepath.IsAbs(archive) && filepath.Clean(archive) != "." && !strings.HasPrefix(filepath.Clean(archive), "..")
	for _, part := range strings.Split(filepath.ToSlash(archive), "/") {
		if part == ".." {
			valid = false
		}
	}
	if valid {
		s.mu.Lock()
		s.archiveDir = filepath.Clean(archive)
		s.mu.Unlock()
	} else {
		s.logf("Config warning: invalid archive_dir %q; keeping current archive directory", archive)
	}

	var cols []string
	seen := map[string]bool{}
	for _, c := range loaded.Columns {
		if c == "" || seen[c] {
			if seen[c] {
				s.logf("Config warning: duplicate column %q is ignored", c)
			}
			continue
		}
		if c == StatusConflict || c == StatusOther {
			s.logf("Config warning: %q is a reserved column name and is ignored", c)
			continue
		}
		seen[c] = true
		cols = append(cols, c)
	}

	if len(cols) == 0 {
		// An empty file, an empty `columns:` list, or a list containing only
		// reserved names would otherwise leave the board with zero columns —
		// every card falls into the non-droppable OTHER bucket and the board
		// becomes silently read-only. Keep serving the last good config
		// instead (same treatment as a missing file, above). An editor that
		// truncates before writing can trigger this transiently.
		s.mu.RLock()
		hasCurrent := len(s.columns) > 0
		s.mu.RUnlock()
		if !hasCurrent {
			s.mu.Lock()
			s.columns = []string{"TODO", "IN PROGRESS", "DONE"}
			s.mu.Unlock()
			s.logf("Config warning: %s has no usable columns; using default columns", path)
			return nil
		}
		s.logf("Config warning: %s has no usable columns; keeping current in-memory columns", path)
		return nil
	}

	s.mu.Lock()
	s.columns = cols
	s.mu.Unlock()

	return nil
}

// bootstrapConfig writes docs/.kanban.yml from the statuses seen in
// the snapshot, in first-seen (filename) order, excluding CONFLICT and empty
// statuses. Falls back to TODO/IN PROGRESS/DONE when no statuses are found.
func (s *Store) bootstrapConfig(path string) error {
	tasks := s.Snapshot().Tasks

	seen := map[string]bool{}
	var cols []string
	for _, t := range tasks {
		if t.Status == "" || t.Status == StatusConflict || seen[t.Status] {
			continue
		}
		seen[t.Status] = true
		cols = append(cols, t.Status)
	}

	if len(cols) == 0 {
		cols = []string{"TODO", "IN PROGRESS", "DONE"}
	}

	body, err := yaml.Marshal(BoardConfig{Columns: cols})
	if err != nil {
		return err
	}

	var b strings.Builder
	b.WriteString("# MK board columns. Order here is the order on the board.\n")
	b.WriteString("# A card whose `status` matches none of these lands in OTHER.\n")
	b.Write(body)

	if err := ioutil.WriteFile(path, []byte(b.String()), 0644); err != nil {
		return err
	}

	s.logf("Bootstrapped %s with columns: %v", path, cols)
	return nil
}
