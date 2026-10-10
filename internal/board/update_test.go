package board

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	s, err := Open(t.TempDir(), Options{Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func number(v float64) *float64 { return &v }
func TestApplyErrors(t *testing.T) {
	cases := []struct {
		name string
		req  UpdateRequest
		want error
	}{
		{"missing file", UpdateRequest{Status: "TODO"}, ErrMissingFile},
		{"invalid file", UpdateRequest{File: "../card.md", Status: "TODO"}, ErrInvalidFile},
		{"conflict", UpdateRequest{File: "card.md", Status: StatusConflict}, ErrConflictStatus},
		{"other", UpdateRequest{File: "card.md", Status: StatusOther}, ErrOtherStatus},
		{"nothing", UpdateRequest{File: "card.md"}, ErrNothingToUpdate},
		{"frontmatter", UpdateRequest{File: "card.md", Status: "DONE"}, ErrInvalidFrontmatter},
		{"invalid unset", UpdateRequest{File: "card.md", Unset: []string{"project"}}, ErrInvalidUnset},
		{"set and unset status", UpdateRequest{File: "card.md", Status: "DONE", Unset: []string{"status"}}, ErrInvalidUnset},
		{"set and unset order", UpdateRequest{File: "card.md", Order: number(1), Unset: []string{"order"}}, ErrInvalidUnset},
		{"set and unset tags", UpdateRequest{File: "card.md", Tags: []string{}, Unset: []string{"tags"}}, ErrInvalidUnset},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := testStore(t)
			original := "---\nstatus: [\n---\nbody"
			writeTaskFile(t, s.Dir(), "card.md", original)
			if err := s.Apply(tc.req); !errors.Is(err, tc.want) {
				t.Fatalf("error=%v, want %v", err, tc.want)
			}
			after, err := os.ReadFile(filepath.Join(s.Dir(), "card.md"))
			if err != nil || string(after) != original {
				t.Fatalf("rejected update changed file: %q, %v", after, err)
			}
		})
	}
	// ErrEmptyClobber is a defensive invariant: validated requests always produce
	// content, or explicitly ask for an empty body. No public request reaches it.
	if ErrEmptyClobber.Error() != "Refusing to write an empty file over existing content" {
		t.Fatal("empty-clobber message changed")
	}
}
func TestApplyUnset(t *testing.T) {
	for _, key := range []string{"status", "order", "tags"} {
		t.Run(key, func(t *testing.T) {
			s := testStore(t)
			writeTaskFile(t, s.Dir(), "card.md", "---\nstatus: TODO\norder: 2\ntags: [urgent]\nproject: mk\n---\nbody")
			if err := s.Apply(UpdateRequest{File: "card.md", Unset: []string{key}}); err != nil {
				t.Fatal(err)
			}
			data, err := os.ReadFile(filepath.Join(s.Dir(), "card.md"))
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), key+":") || !strings.Contains(string(data), "project: mk") || !strings.HasSuffix(string(data), "body") {
				t.Fatalf("unexpected result %s", data)
			}
		})
	}
}
func TestApplyUnsetWithoutFrontmatter(t *testing.T) {
	s := testStore(t)
	original := "# Body\n\n---\n\nnotes\n"
	writeTaskFile(t, s.Dir(), "card.md", original)
	if err := s.Apply(UpdateRequest{File: "card.md", Unset: []string{"status", "order", "tags"}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(s.Dir(), "card.md"))
	if err != nil || string(data) != original {
		t.Fatalf("result=%q, err=%v", data, err)
	}
}
func TestApplyEmptyTagsRefreshesSnapshotWithoutCacheWrite(t *testing.T) {
	dir := t.TempDir()
	cache := filepath.Join(t.TempDir(), "cache.json")
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\ntags: [urgent]\n---\nbody")
	s, err := Open(dir, Options{CachePath: cache})
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cache)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Apply(UpdateRequest{File: "card.md", Status: "DONE", Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	tasks := s.Snapshot().Tasks
	if len(tasks) != 1 || tasks[0].Status != "DONE" || len(tasks[0].Tags) != 0 {
		t.Fatalf("stale snapshot: %+v", tasks)
	}
	data, err := os.ReadFile(filepath.Join(dir, "card.md"))
	if err != nil || !strings.Contains(string(data), "tags: []") {
		t.Fatalf("missing empty tags: %s, %v", data, err)
	}
	after, err := os.ReadFile(cache)
	if err != nil || string(before) != string(after) {
		t.Fatal("Apply rewrote cache")
	}
}
func TestApplyWrapsIOErrors(t *testing.T) {
	s := testStore(t)
	err := s.Apply(UpdateRequest{File: "absent.md", Status: "TODO"})
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("not wrapped: %v", err)
	}
}
