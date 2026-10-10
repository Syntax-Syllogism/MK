package board

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestFrontmatterKeys(t *testing.T) {
	for _, text := range []string{"plain\n---\nstatus: TODO\n---", "---\nstatus: ''\ntags: []\n---\nBody"} {
		path := filepath.Join(t.TempDir(), "a.md")
		if err := os.WriteFile(path, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		keys, err := FrontmatterKeys(path)
		if err != nil {
			t.Fatal(err)
		}
		if text[0] == 'p' {
			if len(keys) != 0 {
				t.Fatal("thematic breaks treated as frontmatter")
			}
			continue
		}
		if _, ok := keys["order"]; ok {
			t.Fatal("absent order present")
		}
		node, ok := keys["status"]
		if !ok || node.Value != "" {
			t.Fatal("explicit empty status missing")
		}
		tags, ok := keys["tags"]
		if !ok || len(tags.Content) != 0 {
			t.Fatal("explicit empty tags missing")
		}
	}
}
func TestGuardedRestore(t *testing.T) {
	for _, original := range []string{"", "Body without frontmatter\n", "---\n# comment\nstatus: ''\ntags: []\nproject: mk\n---\nBody\n", "---\nstatus: TODO\n---\nBody\n"} {
		dir := t.TempDir()
		path := filepath.Join(dir, "a.md")
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		store, err := Open(dir, Options{})
		if err != nil {
			t.Fatal(err)
		}
		order := 2.0
		if err := store.Apply(UpdateRequest{File: "a.md", Status: "DONE", Order: &order, Tags: []string{}}); err != nil {
			t.Fatal(err)
		}
		after, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		hash := sha256.Sum256(after)
		if err := store.Apply(UpdateRequest{File: "a.md", Restore: []byte(original), ExpectedHash: &hash}); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(data, []byte(original)) {
			t.Fatal("restore not byte-identical")
		}
		if err := store.Apply(UpdateRequest{File: "a.md", Restore: []byte("other body"), ExpectedHash: &hash}); !errors.Is(err, ErrChanged) {
			t.Fatalf("stale hash: %v", err)
		}
		hash = sha256.Sum256(data)
		if err := store.Apply(UpdateRequest{File: "a.md", Restore: []byte("other body"), ExpectedHash: &hash}); !errors.Is(err, ErrInvalidRestore) {
			t.Fatalf("body edit accepted: %v", err)
		}
	}
}
func TestActionBatchRollback(t *testing.T) {
	dir := t.TempDir()
	original := []byte("---\nstatus: TODO\n---\nBody\n")
	for _, file := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(dir, file), original, 0600); err != nil {
			t.Fatal(err)
		}
	}
	store, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	hashes, err := store.ApplyBatchWithHashes([]UpdateRequest{{File: "a.md", Status: "DONE"}, {File: "b.md", Status: StatusOther}})
	if hashes != nil {
		t.Fatal("failed batch returned undo hashes")
	}
	if !errors.Is(err, ErrOtherStatus) {
		t.Fatalf("%v", err)
	}
	for _, file := range []string{"a.md", "b.md"} {
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil || !bytes.Equal(data, original) {
			t.Fatalf("%s rollback failed", file)
		}
	}
	if store.Snapshot().Tasks[0].Status != "TODO" {
		t.Fatal("snapshot not rolled back")
	}
}
func TestActionEmptyTagsPreserveBody(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	text := "---\nstatus: TODO\ntags: [foo]\n---\n\nBody\n\n"
	if err := os.WriteFile(path, []byte(text), 0600); err != nil {
		t.Fatal(err)
	}
	store, err := Open(dir, Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Apply(UpdateRequest{File: "a.md", Tags: []string{}}); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(data, []byte("tags: []")) || !bytes.HasSuffix(data, []byte("---\n\nBody\n\n")) {
		t.Fatal("empty tag output or body changed")
	}
}

func TestActionBatchWrittenHashes(t *testing.T) {
	for _, count := range []int{1, 2} {
		t.Run(fmt.Sprintf("%d files", count), func(t *testing.T) {
			dir := t.TempDir()
			original := []byte("---\n# original formatting\nstatus: TODO\n---\nBody\n")
			requests := []UpdateRequest{{File: "b.md", Status: "DONE"}, {File: "a.md", Tags: []string{"new"}}}[:count]
			for _, req := range requests {
				if err := os.WriteFile(filepath.Join(dir, req.File), original, 0600); err != nil {
					t.Fatal(err)
				}
			}
			store, err := Open(dir, Options{})
			if err != nil {
				t.Fatal(err)
			}
			hashes, err := store.ApplyBatchWithHashes(requests)
			if err != nil || len(hashes) != count {
				t.Fatalf("hashes: %v, error: %v", hashes, err)
			}
			for i, req := range requests {
				data, err := os.ReadFile(filepath.Join(dir, req.File))
				if err != nil {
					t.Fatal(err)
				}
				if hashes[i] != sha256.Sum256(data) || hashes[i] == sha256.Sum256(original) {
					t.Fatalf("%s hash does not match written bytes", req.File)
				}
				if err := store.Apply(UpdateRequest{File: req.File, Restore: original, ExpectedHash: &hashes[i]}); err != nil {
					t.Fatalf("%s undo: %v", req.File, err)
				}
			}
		})
	}
}
