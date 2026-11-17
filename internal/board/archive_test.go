package board

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"testing"
)

func writeArchiveTest(t *testing.T, path, data string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestArchiveConfig(t *testing.T) {
	s, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if s.ArchiveDir() != "archive" {
		t.Fatal(s.ArchiveDir())
	}
	for _, tc := range []struct{ value, want string }{
		{"done/archive", "done/archive"}, {"/tmp/outside", "done/archive"},
		{"../outside", "done/archive"}, {"done/../archive", "done/archive"},
		{".", "done/archive"}, {"", "archive"},
	} {
		writeArchiveTest(t, filepath.Join(s.Dir(), ConfigFileName), "columns: [TODO, DONE]\narchive_dir: '"+tc.value+"'\n")
		if err := s.ReloadConfig(); err != nil {
			t.Fatal(err)
		}
		if s.ArchiveDir() != tc.want {
			t.Fatalf("%q: got %q", tc.value, s.ArchiveDir())
		}
	}
}

func TestArchiveRoundTripAndGuards(t *testing.T) {
	s, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	body := "---\nstatus: DONE\norder: 8\n---\nBody\n"
	path := filepath.Join(s.Dir(), "a.md")
	writeArchiveTest(t, path, body)
	hash := sha256.Sum256([]byte(body))
	if _, err := s.Archive("a.md", [32]byte{}); err == nil {
		t.Fatal("accepted stale hash")
	}
	if _, err := s.Archive("../a.md", hash); err == nil {
		t.Fatal("accepted traversal")
	}
	dest, err := s.Archive("a.md", hash)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("source remains")
	}
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	snap := s.Snapshot()
	if !snap.Archived["a.md"] || len(snap.Tasks) != 0 {
		t.Fatal(snap)
	}
	delete(snap.Archived, "a.md")
	if !s.Snapshot().Archived["a.md"] {
		t.Fatal("snapshot aliases store")
	}
	if err := s.Unarchive("../a.md", hash); err == nil {
		t.Fatal("accepted traversal")
	}
	if err := s.Unarchive("a.md", [32]byte{}); err == nil {
		t.Fatal("accepted stale undo")
	}
	writeArchiveTest(t, path, body)
	if err := s.Unarchive("a.md", hash); err == nil {
		t.Fatal("overwrote active task")
	}
	if _, err := s.Archive("a.md", hash); err == nil {
		t.Fatal("overwrote archive")
	}
	if err := os.Remove(path); err != nil {
		t.Fatal(err)
	}
	writeArchiveTest(t, dest, body+"external")
	if err := s.Unarchive("a.md", hash); err == nil {
		t.Fatal("undid external edit")
	}
	writeArchiveTest(t, dest, body)
	if err := s.Unarchive("a.md", hash); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != body {
		t.Fatal("bytes changed", err)
	}
}

func TestArchivedDependencies(t *testing.T) {
	d := AnalyzeDeps([]Task{{File: "a.md", DependsOn: []string{"done", "missing"}}}, map[string]bool{"done.md": true})
	if d.Missing("done") || !d.IsArchived("done") || !d.Missing("missing") || len(d.Cycles) != 0 {
		t.Fatal(d)
	}
}

func TestArchiveRejectsSymlinkDirectory(t *testing.T) {
	s, err := Open(t.TempDir(), Options{})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), filepath.Join(s.Dir(), "archive")); err != nil {
		t.Skip(err)
	}
	writeArchiveTest(t, filepath.Join(s.Dir(), "a.md"), "body")
	if _, err := s.Archive("a.md", sha256.Sum256([]byte("body"))); err == nil {
		t.Fatal("accepted symlink")
	}
}
