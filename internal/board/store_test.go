package board

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestTaskPathAcceptsFlatFilenames(t *testing.T) {
	path, err := (&Store{dir: "docs"}).TaskPath("notes..draft.md")
	if err != nil {
		t.Fatalf("taskPath returned error: %v", err)
	}

	want := filepath.Join("docs", "notes..draft.md")
	if path != want {
		t.Fatalf("taskPath() = %q, want %q", path, want)
	}
}

func TestTaskPathRejectsMissingFile(t *testing.T) {
	if _, err := (&Store{dir: "docs"}).TaskPath(""); err == nil {
		t.Fatal("taskPath accepted empty file")
	}
}

func TestTaskPathRejectsNestedAndTraversalPaths(t *testing.T) {
	tests := []string{
		"../main.go",
		"foo/../bar.md",
		"subdir/card.md",
		filepath.Join("subdir", "card.md"),
		filepath.Join("..", "main.go"),
	}

	for _, file := range tests {
		t.Run(file, func(t *testing.T) {
			if _, err := (&Store{dir: "docs"}).TaskPath(file); err == nil {
				t.Fatalf("taskPath accepted %q", file)
			}
		})
	}
}

func TestTaskPathRejectsAbsolutePaths(t *testing.T) {
	if _, err := (&Store{dir: "docs"}).TaskPath(filepath.Join(string(filepath.Separator), "tmp", "card.md")); err == nil {
		t.Fatal("taskPath accepted absolute path")
	}
}

func TestScanDocsWritesDependsOnToBoardJSON(t *testing.T) {
	dir := t.TempDir()
	cachePath := filepath.Join(t.TempDir(), "board.json")
	store := &Store{dir: dir, opts: Options{CachePath: cachePath}}
	writeTaskFile(t, dir, "blocked.md", "---\nstatus: TODO\ndepends_on: [design]\n---\nbody")
	writeTaskFile(t, dir, "standalone.md", "---\nstatus: TODO\n---\nbody")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	boardJSON, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("read board.json: %v", err)
	}
	if !strings.Contains(string(boardJSON), `"dependsOn": [`) {
		t.Fatalf("board.json = %s, want dependsOn", boardJSON)
	}
	if strings.Count(string(boardJSON), `"dependsOn"`) != 1 {
		t.Fatalf("board.json = %s, want dependsOn only for the task that declares it", boardJSON)
	}
}

func TestScanDocsIncludesMalformedFrontmatterTasks(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nbody")
	writeTaskFile(t, dir, "good.md", "---\nstatus: TODO\n---\nbody")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}

	tasks := store.Snapshot().Tasks

	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2 (malformed frontmatter must not be dropped)", len(tasks))
	}
}

func TestSnapshotIsIndependent(t *testing.T) {
	s := testStore(t)
	writeTaskFile(t, s.Dir(), "card.md", "---\nstatus: TODO\norder: 1\ntags: [a]\ndepends_on: [b]\n---\nbody")
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	first := s.Snapshot()
	first.Columns[0] = "changed"
	first.Tasks[0].Tags[0] = "changed"
	first.Tasks[0].DependsOn[0] = "changed"
	*first.Tasks[0].Order = 9
	first.Tasks[0].Status = "changed"
	second := s.Snapshot()
	if second.Columns[0] != "TODO" || second.Tasks[0].Tags[0] != "a" || second.Tasks[0].DependsOn[0] != "b" || *second.Tasks[0].Order != 1 || second.Tasks[0].Status != "TODO" {
		t.Fatalf("mutation leaked: %+v", second)
	}
}
func TestStoreUsesInjectedLogger(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, ConfigFileName, "columns: [TODO, CONFLICT]\n")
	writeTaskFile(t, dir, "bad.md", "---\norder: soon\n---\nbody")
	var messages []string
	s, err := Open(dir, Options{Logf: func(format string, args ...any) { messages = append(messages, fmt.Sprintf(format, args...)) }})
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(messages, "\n")
	if !strings.Contains(joined, "reserved column") || !strings.Contains(joined, "Error parsing bad.md") {
		t.Fatalf("warnings=%s", joined)
	}
	if len(s.Snapshot().Tasks) != 1 {
		t.Fatal("malformed task was lost")
	}
}
func TestOpenCreatesDirectoryAndCache(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "new")
	cache := filepath.Join(t.TempDir(), "cache", "board.json")
	s, err := Open(dir, Options{CachePath: cache})
	if err != nil {
		t.Fatal(err)
	}
	if !filepath.IsAbs(s.Dir()) || s.Snapshot().Dir != s.Dir() {
		t.Fatal("dir is not absolute")
	}
	data, err := os.ReadFile(cache)
	if err != nil || string(data) != "null" {
		t.Fatalf("empty cache=%s err=%v", data, err)
	}
}

func TestSnapshotPreservesEmptyLists(t *testing.T) {
	s := testStore(t)
	writeTaskFile(t, s.Dir(), "card.md", "---\nstatus: TODO\ntags: []\ndepends_on: []\n---\nbody")
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	task := s.Snapshot().Tasks[0]
	if task.Tags == nil || task.DependsOn == nil {
		t.Fatal("empty lists became null in snapshot")
	}
}

func TestStoreConcurrentAccess(t *testing.T) {
	s := testStore(t)
	writeTaskFile(t, s.Dir(), "card.md", "---\nstatus: TODO\ntags: [x]\n---\nbody")
	if err := s.Scan(); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	errors := make(chan error, 40)
	workers.Add(3)
	go func() {
		defer workers.Done()
		for i := 0; i < 20; i++ {
			if err := s.Apply(UpdateRequest{File: "card.md", Order: number(float64(i))}); err != nil {
				errors <- err
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 20; i++ {
			if err := s.Scan(); err != nil {
				errors <- err
			}
		}
	}()
	go func() {
		defer workers.Done()
		for i := 0; i < 20; i++ {
			snapshot := s.Snapshot()
			snapshot.Tasks[0].Tags[0] = "copy"
		}
	}()
	workers.Wait()
	close(errors)
	for err := range errors {
		t.Error(err)
	}
	task := s.Snapshot().Tasks[0]
	if task.Order == nil || *task.Order != 19 || task.Tags[0] != "x" {
		t.Fatalf("stale final snapshot: %+v", task)
	}
}
