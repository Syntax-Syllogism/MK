package main

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestTaskPathAcceptsFlatFilenames(t *testing.T) {
	path, err := taskPath("docs", "notes..draft.md")
	if err != nil {
		t.Fatalf("taskPath returned error: %v", err)
	}

	want := filepath.Join("docs", "notes..draft.md")
	if path != want {
		t.Fatalf("taskPath() = %q, want %q", path, want)
	}
}

func TestTaskPathRejectsMissingFile(t *testing.T) {
	if _, err := taskPath("docs", ""); err == nil {
		t.Fatal("taskPath accepted empty file")
	}
}

func TestBroadcastDoesNotBlockOnSlowSSEClient(t *testing.T) {
	clientsMu.Lock()
	originalClients := clients
	clients = make(map[chan string]bool)
	slowClient := make(chan string, 1)
	clients[slowClient] = true
	clientsMu.Unlock()
	t.Cleanup(func() {
		clientsMu.Lock()
		clients = originalClients
		clientsMu.Unlock()
	})

	// Fill the client's one-message queue to reproduce a browser that has
	// not yet consumed its previous SSE update. The watcher must be able to
	// continue scanning and broadcasting future file edits.
	slowClient <- "update"
	done := make(chan struct{})
	go func() {
		broadcast("update")
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("broadcast blocked on a slow SSE client")
	}

	select {
	case got := <-slowClient:
		if got != "update" {
			t.Fatalf("queued message = %q, want update", got)
		}
	default:
		t.Fatal("broadcast unexpectedly removed the existing pending update")
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
			if _, err := taskPath("docs", file); err == nil {
				t.Fatalf("taskPath accepted %q", file)
			}
		})
	}
}

func TestTaskPathRejectsAbsolutePaths(t *testing.T) {
	if _, err := taskPath("docs", filepath.Join(string(filepath.Separator), "tmp", "card.md")); err == nil {
		t.Fatal("taskPath accepted absolute path")
	}
}

func writeTaskFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestLoadConfigBootstrapsFromDiscoveredStatuses(t *testing.T) {
	dir := t.TempDir()
	// Filename order: a_todo.md, b_progress.md, c_done.md, d_dup.md
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "b_progress.md", "---\nstatus: IN PROGRESS\n---\nbody")
	writeTaskFile(t, dir, "c_done.md", "---\nstatus: DONE\n---\nbody")
	writeTaskFile(t, dir, "d_dup.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "e_none.md", "no frontmatter here")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := bootstrapIfAbsent(dir); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "IN PROGRESS", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	if _, err := os.Stat(filepath.Join(dir, configFileName)); err != nil {
		t.Fatalf("expected %s to be written: %v", configFileName, err)
	}
}

func TestParseTaskSurvivesMalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	// order: expects a float; tags: expects a list. Either is a type
	// mismatch yaml.Unmarshal will reject.
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nbody")

	task, err := parseTask(filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("parseTask returned an error; malformed frontmatter must not drop the task: %v", err)
	}
	if task.Status != "" {
		t.Fatalf("Status = %q, want empty so the task routes to OTHER", task.Status)
	}
	if task.Epic != "" {
		t.Fatalf("Epic = %q, want empty (F20: PARSE ERROR must not leak into the epic filter)", task.Epic)
	}
	if task.ParseError == "" {
		t.Fatal("ParseError = \"\", want the underlying YAML error")
	}
	if task.Content != "body" {
		t.Fatalf("Content = %q, want the real file body (%q) preserved, not an error message",
			task.Content, "body")
	}
	if task.File != "bad.md" {
		t.Fatalf("File = %q, want %q", task.File, "bad.md")
	}
}

// A card whose frontmatter fails to parse must round-trip its real body
// unchanged through a save that doesn't touch the editor (e.g. adding a
// tag) — Content must never be the synthetic error message, or a save like
// this would silently overwrite the file's real body with it.
func TestParseErrorCardBodySurvivesUnrelatedSave(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nImportant body I do not want to lose.")

	task, err := parseTask(filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}

	// Simulates saveChanges(): posts back exactly what was loaded into the
	// editor, plus a tags change, without the user having edited the body.
	body := `{"file":"bad.md","content":` + jsonQuote(task.Content) + `,"tags":["x"]}`
	rec := doUpdate(t, dir, body)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	after, err := os.ReadFile(filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if !strings.Contains(string(after), "Important body I do not want to lose") {
		t.Fatalf("DATA LOSS: real body missing after round-trip save; file now:\n%s", after)
	}
}

func jsonQuote(s string) string {
	r := strings.NewReplacer("\\", "\\\\", "\"", "\\\"", "\n", "\\n")
	return "\"" + r.Replace(s) + "\""
}

func TestScanDocsIncludesMalformedFrontmatterTasks(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nbody")
	writeTaskFile(t, dir, "good.md", "---\nstatus: TODO\n---\nbody")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}

	board.mu.RLock()
	tasks := board.Tasks
	board.mu.RUnlock()

	if len(tasks) != 2 {
		t.Fatalf("got %d tasks, want 2 (malformed frontmatter must not be dropped)", len(tasks))
	}
}

func TestBootstrapConfigExcludesConflict(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")
	writeTaskFile(t, dir, "b_conflict.md", "<<<<<<< HEAD\nstuff\n=======\nother\n>>>>>>> branch")
	writeTaskFile(t, dir, "c_done.md", "---\nstatus: DONE\n---\nbody")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := bootstrapIfAbsent(dir); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v (CONFLICT must never appear)", got, want)
	}
}

func TestLoadConfigKeepsCurrentColumnsWhenNewFileIsEmpty(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	// An editor truncating the file mid-save, or a config left with only
	// reserved names, must not brick the board by dropping it to zero
	// columns (every card would fall into the non-droppable OTHER bucket).
	writeTaskFile(t, dir, configFileName, "columns:\n  - CONFLICT\n  - OTHER\n")
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error on empty-after-filtering file: %v", err)
	}

	configMu.RLock()
	got = config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns after all-reserved file = %v, want unchanged %v", got, want)
	}
}

func TestLoadConfigDoesNotClobberOnTransientAbsence(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")
	writeTaskFile(t, dir, "a_todo.md", "---\nstatus: TODO\n---\nbody")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}

	// Simulate an editor's atomic save (temp-write-then-rename) or a git
	// checkout transiently removing the file. loadConfig must not
	// regenerate it from the current task set — that would silently
	// discard the hand-edited REVIEW/BLOCKED columns.
	if err := os.Remove(filepath.Join(dir, configFileName)); err != nil {
		t.Fatalf("failed to remove config: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error on missing file: %v", err)
	}

	configMu.RLock()
	got = config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns after transient removal = %v, want unchanged %v", got, want)
	}
	if _, err := os.Stat(filepath.Join(dir, configFileName)); !os.IsNotExist(err) {
		t.Fatalf("loadConfig must not regenerate the file; got err = %v", err)
	}
}

func TestLoadConfigBootstrapsDefaultOnEmptyDocs(t *testing.T) {
	dir := t.TempDir()

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := bootstrapIfAbsent(dir); err != nil {
		t.Fatalf("bootstrapIfAbsent error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"TODO", "IN PROGRESS", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigPreservesExistingOrder(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - REVIEW\n  - BLOCKED\n  - DONE\n")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}

	want := []string{"REVIEW", "BLOCKED", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigRejectsReservedColumnNames(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - TODO\n  - CONFLICT\n  - OTHER\n  - DONE\n")

	if err := scanDocs(dir, filepath.Join(t.TempDir(), "board.json")); err != nil {
		t.Fatalf("scanDocs error: %v", err)
	}
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig should not fatal on reserved names: %v", err)
	}

	want := []string{"TODO", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want %v", got, want)
	}
}

func TestLoadConfigUsesDefaultsWhenInitialConfigHasNoUsableColumns(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - CONFLICT\n  - OTHER\n")
	configMu.Lock()
	config = &BoardConfig{}
	configMu.Unlock()

	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	want := []string{"TODO", "IN PROGRESS", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want safe defaults %v", got, want)
	}
}

func TestLoadConfigDeduplicatesColumns(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, configFileName, "columns:\n  - TODO\n  - TODO\n  - DONE\n  - TODO\n")
	if err := loadConfig(dir); err != nil {
		t.Fatalf("loadConfig error: %v", err)
	}
	want := []string{"TODO", "DONE"}
	configMu.RLock()
	got := config.Columns
	configMu.RUnlock()
	if !equalStrings(got, want) {
		t.Fatalf("config.Columns = %v, want deduplicated %v", got, want)
	}
}

func TestParseTaskRejectsNonFiniteOrder(t *testing.T) {
	for _, value := range []string{".nan", ".inf", "-.inf"} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nonfinite.md")
			if err := os.WriteFile(path, []byte("---\nstatus: TODO\norder: "+value+"\n---\nbody"), 0644); err != nil {
				t.Fatal(err)
			}
			task, err := parseTask(path)
			if err != nil {
				t.Fatalf("parseTask error: %v", err)
			}
			if task.Status != "" || task.ParseError == "" || task.Order != nil {
				t.Fatalf("task = %+v, want parse error in OTHER with no order", task)
			}
		})
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func doUpdate(t *testing.T, docsDir string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/update", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	updateTask(rec, req, docsDir)
	return rec
}

func TestUpdateTaskRejectsConflictStatus(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\n---\nbody")
	original, _ := os.ReadFile(filepath.Join(dir, "card.md"))

	rec := doUpdate(t, dir, `{"file":"card.md","status":"CONFLICT"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	after, _ := os.ReadFile(filepath.Join(dir, "card.md"))
	if !bytes.Equal(original, after) {
		t.Fatalf("file was modified: got %q, want %q", after, original)
	}
}

func TestUpdateTaskRejectsOtherStatus(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\n---\nbody")

	rec := doUpdate(t, dir, `{"file":"card.md","status":"OTHER"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}
}

func TestUpdateTaskWritesAndOverwritesOrder(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\n---\nbody")

	rec := doUpdate(t, dir, `{"file":"card.md","order":1.5}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	task, err := parseTask(filepath.Join(dir, "card.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil || *task.Order != 1.5 {
		t.Fatalf("Order = %v, want 1.5", task.Order)
	}

	rec = doUpdate(t, dir, `{"file":"card.md","order":2.25}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
	task, err = parseTask(filepath.Join(dir, "card.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil || *task.Order != 2.25 {
		t.Fatalf("Order = %v, want 2.25 (overwrite, not duplicate)", task.Order)
	}

	content, _ := os.ReadFile(filepath.Join(dir, "card.md"))
	if strings.Count(string(content), "order:") != 1 {
		t.Fatalf("expected exactly one order: key, got content:\n%s", content)
	}
}

func TestUpdateTaskWritesOrderWithNoExistingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "just some body text, no frontmatter")

	rec := doUpdate(t, dir, `{"file":"card.md","order":3}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	task, err := parseTask(filepath.Join(dir, "card.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil || *task.Order != 3 {
		t.Fatalf("Order = %v, want 3", task.Order)
	}
}

// F16: a request with no status, order, content, or tags must be rejected
// before any file I/O, not fall through to writing an empty file.
func TestUpdateTaskRejectsEmptyRequest(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "b.md", "---\nstatus: TODO\n---\nPrecious body\n\nplease do not delete\n")
	original, _ := os.ReadFile(filepath.Join(dir, "b.md"))

	rec := doUpdate(t, dir, `{"file":"b.md"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusBadRequest)
	}

	after, _ := os.ReadFile(filepath.Join(dir, "b.md"))
	if !bytes.Equal(original, after) {
		t.Fatalf("file was modified: got %q, want %q", after, original)
	}
}

// F17: adding tags to a frontmatter-less file must actually write them, not
// silently succeed while discarding the tags.
func TestUpdateTaskWritesTagsWithNoExistingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "a.md", "just a body, no frontmatter")

	body := "urgent"
	rec := doUpdate(t, dir, `{"file":"a.md","content":"just a body, no frontmatter","tags":["`+body+`"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	task, err := parseTask(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "urgent" {
		t.Fatalf("Tags = %v, want [urgent]", task.Tags)
	}
}

// F26: dragging a card (status + order, no content key — exactly what
// handleDrop sends) whose file has no frontmatter must preserve the file's
// existing body, not replace it with an empty one.
func TestUpdateTaskDragPreservesBodyWithNoExistingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	body := "# Important Notes\n\nparagraph one\n\nparagraph two\n"
	writeTaskFile(t, dir, "a.md", body)

	rec := doUpdate(t, dir, `{"file":"a.md","status":"DONE","order":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	task, err := parseTask(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if !strings.Contains(task.Content, "paragraph one") || !strings.Contains(task.Content, "paragraph two") {
		t.Fatalf("Content = %q, want the real body preserved", task.Content)
	}
}

// F26: a tag-only write to a frontmatter-less file must preserve the body too.
func TestUpdateTaskTagOnlyWritePreservesBodyWithNoExistingFrontmatter(t *testing.T) {
	dir := t.TempDir()
	body := "# Important Notes\n\nparagraph one\n\nparagraph two\n"
	writeTaskFile(t, dir, "a.md", body)

	rec := doUpdate(t, dir, `{"file":"a.md","tags":["urgent"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	task, err := parseTask(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if !strings.Contains(task.Content, "paragraph one") || !strings.Contains(task.Content, "paragraph two") {
		t.Fatalf("Content = %q, want the real body preserved", task.Content)
	}
	if len(task.Tags) != 1 || task.Tags[0] != "urgent" {
		t.Fatalf("Tags = %v, want [urgent]", task.Tags)
	}
}

// F26: dragging a thematic-break-only note (F25's no-frontmatter case) must
// also preserve the body, not just be correctly diagnosed as frontmatter-less.
func TestUpdateTaskDragPreservesThematicBreakBody(t *testing.T) {
	dir := t.TempDir()
	body := "# My Notes\n\nintro\n\n---\n\nmiddle\n\n---\n\nend\n"
	writeTaskFile(t, dir, "notes.md", body)

	rec := doUpdate(t, dir, `{"file":"notes.md","status":"DONE","order":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	task, err := parseTask(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if !strings.Contains(task.Content, "intro") || !strings.Contains(task.Content, "middle") || !strings.Contains(task.Content, "end") {
		t.Fatalf("Content = %q, want the full body preserved", task.Content)
	}
}

// F18: an empty frontmatter block unmarshals to a nil map; updateTask must
// not panic assigning into it.
func TestUpdateTaskHandlesEmptyFrontmatterBlock(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "h.md", "---\n---\n\n# Title\n\nbody\n")

	rec := doUpdate(t, dir, `{"file":"h.md","status":"DONE","order":1}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}

	task, err := parseTask(filepath.Join(dir, "h.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Status != "DONE" {
		t.Fatalf("Status = %q, want DONE", task.Status)
	}
	if task.Order == nil || *task.Order != 1 {
		t.Fatalf("Order = %v, want 1", task.Order)
	}
}

// F24: deliberately clearing a frontmatter-less file's body (content: "")
// must be allowed — the empty-write guard should only block the case where
// nobody asked for an empty body at all.
func TestUpdateTaskAllowsDeliberateEmptyBody(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "a.md", "some body")

	rec := doUpdate(t, dir, `{"file":"a.md","content":""}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200: %s", rec.Code, rec.Body.String())
	}

	after, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("read error: %v", err)
	}
	if string(after) != "" {
		t.Fatalf("content = %q, want empty file", after)
	}
}

// F25: a file with no frontmatter but two "---" thematic breaks in its body
// must not be misparsed as having frontmatter.
func TestParseTaskIgnoresNonLeadingThematicBreaks(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "notes.md", "# My Notes\n\nintro\n\n---\n\nmiddle\n\n---\n\nend\n")

	task, err := parseTask(filepath.Join(dir, "notes.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.ParseError != "" {
		t.Fatalf("ParseError = %q, want empty — this file never had frontmatter", task.ParseError)
	}
	if !strings.Contains(task.Content, "intro") || !strings.Contains(task.Content, "end") {
		t.Fatalf("Content = %q, want the full body preserved", task.Content)
	}
}

func TestParseTaskOrderRoundTrip(t *testing.T) {
	dir := t.TempDir()

	writeTaskFile(t, dir, "with_order.md", "---\nstatus: TODO\norder: 1.5\n---\nbody")
	task, err := parseTask(filepath.Join(dir, "with_order.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil || *task.Order != 1.5 {
		t.Fatalf("Order = %v, want *float64(1.5)", task.Order)
	}

	writeTaskFile(t, dir, "no_order.md", "---\nstatus: TODO\n---\nbody")
	task, err = parseTask(filepath.Join(dir, "no_order.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order != nil {
		t.Fatalf("Order = %v, want nil", task.Order)
	}

	writeTaskFile(t, dir, "zero_order.md", "---\nstatus: TODO\norder: 0\n---\nbody")
	task, err = parseTask(filepath.Join(dir, "zero_order.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil {
		t.Fatal("Order = nil, want *float64(0), not nil")
	}
	if *task.Order != 0 {
		t.Fatalf("Order = %v, want 0", *task.Order)
	}
}
