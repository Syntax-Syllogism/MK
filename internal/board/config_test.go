package board

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigBootstrapsFromDiscoveredStatuses(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	// Filename order: a_todo.md, b_progress.md, c_done.md, d_dup.md
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "b_progress.md", "---\nstatus: IN PROGRESS\n---\nbody")
	writeTaskFile(t, dir, "c_done.md", "---\nstatus: DONE\n---\nbody")
	writeTaskFile(t, dir, "d_dup.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "e_none.md", "no frontmatter here")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.bootstrapIfAbsent(); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "IN PROGRESS", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	if _, err := os.Stat(filepath.Join(dir, ConfigFileName)); err != nil {
		t.Fatalf("expected %s to be written: %v", ConfigFileName, err)
	}
}

func TestBootstrapConfigExcludesConflict(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "b_conflict.md", "<<<<<<< HEAD\nstuff\n=======\nother\n>>>>>>> branch")
	writeTaskFile(t, dir, "c_done.md", "---\nstatus: DONE\n---\nbody")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.bootstrapIfAbsent(); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v (CONFLICT must never appear)", got, want)
	}
}

func TestLoadConfigKeepsCurrentColumnsWhenNewFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	// An editor truncating the file mid-save, or a config left with only
	// reserved names, must not brick the board by dropping it to zero
	// columns (every card would fall into the non-droppable OTHER bucket).
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - CONFLICT\n  - OTHER\n")
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error on empty-after-filtering file: %v", err)
	}

	got = store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns after all-reserved file = %v, want unchanged %v", got, want)
	}
}

func TestLoadConfigDoesNotClobberOnTransientAbsence(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	// Simulate an editor's atomic save (temp-write-then-rename) or a git
	// checkout transiently removing the file. loadConfig must not
	// regenerate it from the current task set — that would silently
	// discard the hand-edited REVIEW/BLOCKED columns.
	if err := os.Remove(filepath.Join(dir, ConfigFileName)); err != nil {
		t.Fatalf("failed to remove config: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error on missing file: %v", err)
	}

	got = store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns after transient removal = %v, want unchanged %v", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, ConfigFileName)); !os.IsNotExist(err) {
		t.Fatalf("loadConfig must not regenerate the file; got err = %v", err)
	}
}

func TestLoadConfigBootstrapsDefaultOnEmptyDocs(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.bootstrapIfAbsent(); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "IN PROGRESS", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigPreservesExistingOrder(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigRejectsReservedColumnNames(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - TODO\n  - CONFLICT\n  - OTHER\n  - DONE\n")

	if err := store.Scan(); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig should not fatal on reserved names: %v", err)
	}

	want := []string{"TODO", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigUsesDefaultsWhenInitialConfigHasNoUsableColumns(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - CONFLICT\n  - OTHER\n")

	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	want := []string{"TODO", "IN PROGRESS", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want safe defaults %v", got, want)
	}
}

func TestLoadConfigDeduplicatesColumns(t *testing.T) {
	dir := t.TempDir()
	store := &Store{dir: dir, opts: Options{CachePath: filepath.Join(t.TempDir(), "board.json")}}
	writeTaskFile(t, dir, ConfigFileName, "columns:\n  - TODO\n  - TODO\n  - DONE\n  - TODO\n")
	if err := store.ReloadConfig(); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	want := []string{"TODO", "DONE"}
	got := store.Snapshot().Columns
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want deduplicated %v", got, want)
	}
}

func TestLoadConfigUsesDefaultsWhenInitiallyMissing(t *testing.T) {
	store := &Store{dir: t.TempDir(), opts: Options{Logf: func(string, ...any) {}}}
	if err := store.ReloadConfig(); err != nil {
		t.Fatal(err)
	}
	if !equalStrings(store.Snapshot().Columns, []string{"TODO", "IN PROGRESS", "DONE"}) {
		t.Fatalf("missing defaults: %v", store.Snapshot().Columns)
	}
}
