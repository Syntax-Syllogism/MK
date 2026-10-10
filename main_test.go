package main

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/Syntax-Syllogism/mk/internal/board"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestBroadcastDoesNotBlockOnSlowSSEClient(t *testing.T) {
	slowClient := make(chan string, 1)
	srv := &server{clients: map[chan string]bool{slowClient: true}}

	// Fill the client's one-message queue to reproduce a browser that has
	// not yet consumed its previous SSE update. The watcher must be able to
	// continue scanning and broadcasting future file edits.
	slowClient <- "update"
	done := make(chan struct{})
	go func() {
		srv.broadcast("update")
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

func writeTaskFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("failed to write %s: %v", name, err)
	}
}

func TestParseErrorCardBodySurvivesUnrelatedSave(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nImportant body I do not want to lose.")

	task, err := board.ParseTask(filepath.Join(dir, "bad.md"))
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

func doUpdate(t *testing.T, docsDir string, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/update", bytes.NewBufferString(body))
	rec := httptest.NewRecorder()
	store, err := board.Open(docsDir, board.Options{Logf: func(string, ...any) {}})
	if err != nil {
		t.Fatal(err)
	}
	srv := &server{store: store}
	srv.handleUpdate(rec, req)
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
	task, err := board.ParseTask(filepath.Join(dir, "card.md"))
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
	task, err = board.ParseTask(filepath.Join(dir, "card.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "card.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "a.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "a.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "a.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "notes.md"))
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

	task, err := board.ParseTask(filepath.Join(dir, "h.md"))
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

func TestUpdateHTTPErrorMapping(t *testing.T) {
	for _, tc := range []struct {
		name, method, body, content, message string
		code                                 int
	}{
		{"method", http.MethodGet, `{}`, "body", "Method not allowed", 405},
		{"json", http.MethodPost, `{`, "body", "unexpected EOF", 400},
		{"missing", http.MethodPost, `{"status":"TODO"}`, "body", board.ErrMissingFile.Error(), 400},
		{"invalid", http.MethodPost, `{"file":"../card.md","status":"TODO"}`, "body", board.ErrInvalidFile.Error(), 400},
		{"conflict", http.MethodPost, `{"file":"card.md","status":"CONFLICT"}`, "body", board.ErrConflictStatus.Error(), 400},
		{"other", http.MethodPost, `{"file":"card.md","status":"OTHER"}`, "body", board.ErrOtherStatus.Error(), 400},
		{"nothing", http.MethodPost, `{"file":"card.md"}`, "body", board.ErrNothingToUpdate.Error(), 400},
		{"frontmatter", http.MethodPost, `{"file":"card.md","status":"DONE"}`, "---\nstatus: [\n---\nbody", board.ErrInvalidFrontmatter.Error(), 500},
		{"IO", http.MethodPost, `{"file":"missing.md","status":"DONE"}`, "body", "", 500},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeTaskFile(t, dir, "card.md", tc.content)
			store, err := board.Open(dir, board.Options{Logf: func(string, ...any) {}})
			if err != nil {
				t.Fatal(err)
			}
			srv := &server{store: store}
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tc.method, "/api/update", strings.NewReader(tc.body))
			srv.handleUpdate(rec, req)
			if rec.Code != tc.code {
				t.Fatalf("code=%d want %d: %s", rec.Code, tc.code, rec.Body.String())
			}
			if tc.message != "" && rec.Body.String() != tc.message+"\n" {
				t.Fatalf("message=%q want %q", rec.Body.String(), tc.message)
			}
		})
	}
}
func TestHTTPDoesNotExposeUnset(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\n---\nbody")
	rec := doUpdate(t, dir, `{"file":"card.md","unset":["status"]}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("unset-only code=%d", rec.Code)
	}
	rec = doUpdate(t, dir, `{"file":"card.md","tags":[],"unset":["status"]}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("tags code=%d", rec.Code)
	}
	task, err := board.ParseTask(filepath.Join(dir, "card.md"))
	if err != nil || task.Status != "TODO" || len(task.Tags) != 0 {
		t.Fatalf("HTTP unset applied: %+v %v", task, err)
	}
}
func TestBoardHTTPResponse(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "card.md", "---\nstatus: TODO\n---\nbody")
	store, err := board.Open(dir, board.Options{})
	if err != nil {
		t.Fatal(err)
	}
	srv := &server{store: store}
	rec := httptest.NewRecorder()
	srv.handleBoard(rec, httptest.NewRequest(http.MethodGet, "/api/board", nil))
	var response struct {
		Columns []string     `json:"columns"`
		Tasks   []board.Task `json:"tasks"`
		DocsDir string       `json:"docsDir"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if rec.Code != 200 || rec.Header().Get("Content-Type") != "application/json" || response.DocsDir != store.Dir() || len(response.Tasks) != 1 || len(response.Columns) != 1 || response.Columns[0] != "TODO" {
		t.Fatalf("response=%+v", response)
	}
}

// sseRecorder signals flushes without opening a network socket.
type sseRecorder struct {
	*httptest.ResponseRecorder
	flushed chan struct{}
}

func (r *sseRecorder) Flush() { r.ResponseRecorder.Flush(); r.flushed <- struct{}{} }
func TestSSEDeliveryAndCleanup(t *testing.T) {
	srv := &server{clients: map[chan string]bool{}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	req := httptest.NewRequest(http.MethodGet, "/api/events", nil).WithContext(ctx)
	rec := &sseRecorder{ResponseRecorder: httptest.NewRecorder(), flushed: make(chan struct{}, 1)}
	done := make(chan struct{})
	go func() { srv.handleEvents(rec, req); close(done) }()
	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()
	registered := false
	for !registered {
		select {
		case <-ticker.C:
			srv.clientsMu.Lock()
			registered = len(srv.clients) == 1
			srv.clientsMu.Unlock()
		case <-deadline:
			t.Fatal("SSE client did not register")
		}
	}
	srv.broadcast("update")
	select {
	case <-rec.flushed:
	case <-time.After(time.Second):
		t.Fatal("SSE did not flush")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("SSE did not stop")
	}
	if rec.Body.String() != "data: update\n\n" || rec.Header().Get("Content-Type") != "text/event-stream" {
		t.Fatalf("SSE response=%s", rec.Body.String())
	}
	srv.clientsMu.Lock()
	n := len(srv.clients)
	srv.clientsMu.Unlock()
	if n != 0 {
		t.Fatal("SSE client did not unregister")
	}
}
