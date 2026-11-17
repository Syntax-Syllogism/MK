package board

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"log"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

type Options struct {
	// CachePath is relative to the working directory; empty disables cache writes.
	CachePath string
	// Logf receives warnings. Nil uses log.Printf.
	Logf func(format string, args ...any)
}
type Snapshot struct {
	Archived map[string]bool
	Columns  []string
	Tasks    []Task
	Dir      string
}
type Store struct {
	dir  string
	opts Options
	mu   sync.RWMutex
	// Serialize file reads/writes so a scan cannot replace a newer Apply result.
	ioMu       sync.Mutex
	tasks      []Task
	columns    []string
	archiveDir string
	archived   map[string]bool
}

// Open creates the task directory, scans it, and bootstraps then loads its config.
func Open(dir string, opts Options) (*Store, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(abs, 0755); err != nil {
		return nil, err
	}
	s := &Store{dir: abs, opts: opts}
	if err := s.Scan(); err != nil {
		s.logf("Initial scan error: %v", err)
	}
	if err := s.bootstrapIfAbsent(); err != nil {
		s.logf("Config bootstrap error: %v", err)
	}
	if err := s.ReloadConfig(); err != nil {
		s.logf("Config load error: %v", err)
	}
	if err := s.Scan(); err != nil {
		s.logf("Archive scan error: %v", err)
	}
	return s, nil
}
func (s *Store) logf(format string, args ...any) {
	if s.opts.Logf != nil {
		s.opts.Logf(format, args...)
		return
	}
	log.Printf(format, args...)
}
func (s *Store) Dir() string { return s.dir }

// Snapshot returns independent copies, including slices and order pointers.
func (s *Store) Snapshot() Snapshot {
	s.mu.RLock()
	defer s.mu.RUnlock()
	tasks := cloneTasks(s.tasks)
	return Snapshot{Columns: append([]string(nil), s.columns...), Tasks: tasks, Dir: s.dir, Archived: maps.Clone(s.archived)}
}
func cloneTasks(tasks []Task) []Task {
	if tasks == nil {
		return nil
	}
	result := append([]Task{}, tasks...)
	for i := range result {
		result[i].Tags = slices.Clone(tasks[i].Tags)
		result[i].DependsOn = slices.Clone(tasks[i].DependsOn)
		if tasks[i].Order != nil {
			order := *tasks[i].Order
			result[i].Order = &order
		}
	}
	return result
}

func (s *Store) TaskPath(file string) (string, error) {
	if file == "" {
		return "", ErrMissingFile
	}
	if file != filepath.Base(file) {
		return "", ErrInvalidFile
	}
	return filepath.Join(s.dir, file), nil
}

func (s *Store) Scan() error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	files, err := ioutil.ReadDir(s.dir)
	if err != nil {
		return err
	}

	var newTasks []Task
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".md") {
			task, err := ParseTask(filepath.Join(s.dir, f.Name()))
			if err != nil {
				s.logf("Error parsing %s: %v", f.Name(), err)
				continue
			}
			if task.ParseError != "" {
				s.logf("Error parsing %s: %s", f.Name(), task.ParseError)
			}
			newTasks = append(newTasks, task)
		}
	}

	archived, archiveErr := s.scanArchived()
	s.mu.Lock()
	s.tasks = newTasks
	s.archived = archived
	s.mu.Unlock()

	if s.opts.CachePath == "" {
		return archiveErr
	}
	data, err := json.MarshalIndent(newTasks, "", "  ")
	if err != nil {
		return errors.Join(archiveErr, err)
	}
	if err := os.MkdirAll(filepath.Dir(s.opts.CachePath), 0755); err != nil {
		return errors.Join(archiveErr, err)
	}
	return errors.Join(archiveErr, ioutil.WriteFile(s.opts.CachePath, data, 0644))
}
