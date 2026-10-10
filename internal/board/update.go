package board

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
	"strings"

	"gopkg.in/yaml.v3"
)

type UpdateRequest struct {
	File    string
	Content *string
	Status  string
	Order   *float64
	Tags    []string
	Unset   []string
	// Restore is a TUI-only, hash-guarded byte-exact undo. It can change only
	// status, order and tags; the body and other frontmatter must match.
	Restore      []byte
	ExpectedHash *[32]byte
}

var (
	ErrChanged            = errors.New("file changed since action")
	ErrInvalidRestore     = errors.New("restore may only change status, order and tags")
	ErrMissingFile        = errors.New("missing file")
	ErrInvalidFile        = errors.New("invalid file")
	ErrInvalidUnset       = errors.New("invalid unset key")
	ErrConflictStatus     = errors.New("CONFLICT is a derived status and cannot be assigned")
	ErrOtherStatus        = errors.New("OTHER is a diagnostic bucket and cannot be assigned")
	ErrNothingToUpdate    = errors.New("Nothing to update: at least one of status, order, content, or tags is required")
	ErrInvalidFrontmatter = errors.New("Invalid frontmatter")
	ErrEmptyClobber       = errors.New("Refusing to write an empty file over existing content")
)

func (s *Store) Apply(req UpdateRequest) error {
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	return s.applyLocked(req)
}

func (s *Store) applyLocked(req UpdateRequest) error {
	return s.applyWithHashLocked(req, nil)
}

func (s *Store) applyWithHashLocked(req UpdateRequest, writtenHash *[32]byte) error {
	for _, key := range req.Unset {
		switch key {
		case "status":
			if req.Status != "" {
				return ErrInvalidUnset
			}
		case "order":
			if req.Order != nil {
				return ErrInvalidUnset
			}
		case "tags":
			if req.Tags != nil {
				return ErrInvalidUnset
			}
		default:
			return ErrInvalidUnset
		}
	}
	if req.Status == StatusConflict {
		return ErrConflictStatus
	}
	if req.Status == StatusOther {
		return ErrOtherStatus
	}
	if req.Status == "" && req.Order == nil && req.Content == nil && req.Tags == nil && len(req.Unset) == 0 && req.Restore == nil {
		return ErrNothingToUpdate
	}

	path, err := s.TaskPath(req.File)
	if err != nil {
		return err
	}

	existing, err := ioutil.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}

	if req.ExpectedHash != nil && sha256.Sum256(existing) != *req.ExpectedHash {
		return ErrChanged
	}
	parts := strings.SplitN(string(existing), "---", 3)
	var newContent string
	// Mirrors ParseTask's F25 fix: only treat the file as having frontmatter if
	// it actually starts with the delimiter, not merely contains two "---" lines.
	if strings.HasPrefix(string(existing), "---") && len(parts) >= 3 {
		// Parse frontmatter to update status if provided
		var fm map[string]interface{}
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
			return ErrInvalidFrontmatter
		}
		if fm == nil {
			fm = map[string]interface{}{}
		}

		if req.Status != "" {
			fm["status"] = req.Status
		}
		if req.Order != nil {
			fm["order"] = *req.Order
		}
		if req.Tags != nil {
			fm["tags"] = req.Tags
		}

		for _, key := range req.Unset {
			delete(fm, key)
		}
		updatedFM, _ := yaml.Marshal(fm)
		body := parts[2]
		// [P2] Distinguish between omitted and empty body
		if req.Content != nil {
			body = "\n" + *req.Content
		}
		newContent = "---\n" + string(updatedFM) + "---" + body
	} else {
		// No frontmatter, just update content or add frontmatter if status, order, or tags provided
		if req.Status != "" || req.Order != nil || req.Tags != nil {
			fm := map[string]interface{}{}
			if req.Status != "" {
				fm["status"] = req.Status
			}
			if req.Order != nil {
				fm["order"] = *req.Order
			}
			if req.Tags != nil {
				fm["tags"] = req.Tags
			}
			fmBytes, _ := yaml.Marshal(fm)
			// F26: default to the file's existing body, not "" — a drag (status
			// + order, no content key) must not wipe a frontmatter-less file.
			content := string(existing)
			if req.Content != nil {
				content = *req.Content
			}
			newContent = "---\n" + string(fmBytes) + "---\n" + content
		} else if req.Content != nil {
			newContent = *req.Content
		} else {
			newContent = string(existing)
		}
	}

	if req.Restore != nil {
		if req.ExpectedHash == nil || !validRestore(existing, req.Restore) {
			return ErrInvalidRestore
		}
		newContent = string(req.Restore)
	}
	// Belt-and-braces: never let a computed-empty write clobber a non-empty file
	// when nobody actually asked for an empty body (req.Content == nil means the
	// caller sent no content key at all, not that they explicitly emptied it).
	if newContent == "" && len(existing) > 0 && req.Content == nil && req.Restore == nil {
		return ErrEmptyClobber
	}

	if err := ioutil.WriteFile(path, []byte(newContent), 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}

	task, err := ParseTask(path)
	if err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	if writtenHash != nil {
		*writtenHash = sha256.Sum256([]byte(newContent))
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.tasks {
		if s.tasks[i].File == req.File {
			s.tasks[i] = task
			return nil
		}
	}
	s.tasks = append(s.tasks, task)
	return nil
}

// ApplyBatch serializes a multi-file action and rolls back completed writes if
// a later write fails. This includes DropOrder's unordered-tail updates.
func (s *Store) ApplyBatch(requests []UpdateRequest) error {
	_, err := s.ApplyBatchWithHashes(requests)
	return err
}

// ApplyBatchWithHashes returns hashes of the bytes written, in request order.
// Callers can record guarded undo without a fallible post-write disk read.
// On failure it returns no hashes and uses ApplyBatch's rollback behavior.
func (s *Store) ApplyBatchWithHashes(requests []UpdateRequest) ([][32]byte, error) {
	hashes := make([][32]byte, len(requests))
	s.ioMu.Lock()
	defer s.ioMu.Unlock()
	if len(requests) == 1 {
		if err := s.applyWithHashLocked(requests[0], &hashes[0]); err != nil {
			return nil, err
		}
		return hashes, nil
	}
	originals := make(map[string][]byte)
	for _, req := range requests {
		path, err := s.TaskPath(req.File)
		if err != nil {
			return nil, err
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}
		if req.ExpectedHash != nil && sha256.Sum256(data) != *req.ExpectedHash {
			return nil, ErrChanged
		}
		originals[req.File] = data
	}
	before := s.Snapshot()
	for i, req := range requests {
		if err := s.applyWithHashLocked(req, &hashes[i]); err != nil {
			var rollbackErr error
			for _, done := range requests[:i] {
				path, _ := s.TaskPath(done.File)
				rollbackErr = errors.Join(rollbackErr, ioutil.WriteFile(path, originals[done.File], 0644))
			}
			s.mu.Lock()
			s.tasks = before.Tasks
			s.mu.Unlock()
			return nil, errors.Join(err, rollbackErr)
		}
	}
	return hashes, nil
}

func restoreParts(data []byte) (map[string]interface{}, string, error) {
	text := string(data)
	parts := strings.SplitN(text, "---", 3)
	fm := map[string]interface{}{}
	body := text
	if strings.HasPrefix(text, "---") && len(parts) == 3 {
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
			return nil, "", err
		}
		body = parts[2]
	} else {
		body = "\n" + body
	}
	if fm == nil {
		fm = map[string]interface{}{}
	}
	for _, key := range []string{"status", "order", "tags"} {
		delete(fm, key)
	}
	return fm, body, nil
}
func validRestore(existing, original []byte) bool {
	if bytes.Equal(existing, original) {
		return true
	}
	a, bodyA, errA := restoreParts(existing)
	b, bodyB, errB := restoreParts(original)
	return errA == nil && errB == nil && bodyA == bodyB && reflect.DeepEqual(a, b)
}
