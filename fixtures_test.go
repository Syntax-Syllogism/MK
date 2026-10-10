package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Syntax-Syllogism/mk/internal/board"
)

func TestWorkItemFixtures(t *testing.T) {
	// Open only a scratch copy: board writes must never repair pristine fixtures.
	dir := t.TempDir()
	entries, err := os.ReadDir("test-work-items")
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		content, err := os.ReadFile(filepath.Join("test-work-items", entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		writeTaskFile(t, dir, entry.Name(), string(content))
	}
	store, err := board.Open(dir, board.Options{})
	if err != nil {
		t.Fatal(err)
	}
	srv := &server{store: store}
	rec := httptest.NewRecorder()
	srv.handleBoard(rec, httptest.NewRequest(http.MethodGet, "/api/board", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("board status = %d", rec.Code)
	}
	var snapshot board.Snapshot
	if err := json.Unmarshal(rec.Body.Bytes(), &snapshot); err != nil {
		t.Fatal(err)
	}
	var sections []string
	for _, section := range board.Sections(snapshot) {
		sections = append(sections, section.Name)
	}
	if want := []string{"CONFLICT", "TODO", "IN PROGRESS", "UNDER REVIEW", "DONE", "OTHER"}; !slices.Equal(sections, want) {
		t.Fatalf("sections = %v, want %v", sections, want)
	}
	tasks := make(map[string]board.Task)
	var conflicts, parseErrors []string
	var todo []board.Task
	for _, task := range snapshot.Tasks {
		tasks[task.File] = task
		if task.Status == board.StatusConflict {
			conflicts = append(conflicts, task.File)
		}
		if task.ParseError != "" {
			parseErrors = append(parseErrors, task.File)
		}
		if board.Section(task, snapshot.Columns) == "TODO" {
			todo = append(todo, task)
		}
	}
	if !slices.Equal(conflicts, []string{"resolve_merge_conflict_in_onboarding.md"}) {
		t.Fatalf("conflicts = %v", conflicts)
	}
	if !slices.Equal(parseErrors, []string{"migrate_legacy_priority_field.md"}) {
		t.Fatalf("parse errors = %v", parseErrors)
	}
	for file, status := range map[string]string{
		"verify_status_after_board_rename.md": "REVIEW",
		"draft_changelog_for_v2_release.md":   "",
		"capture_customer_feedback_notes.md":  "",
		"spike_notification_service.md":       "",
	} {
		task, ok := tasks[file]
		if !ok || task.Status != status || task.ParseError != "" || board.Section(task, snapshot.Columns) != "OTHER" {
			t.Errorf("%s should be an ordinary OTHER card with status %q: %+v", file, status, task)
		}
	}
	if task := tasks["design_empty_state_illustration.md"]; task.Status != "DONE" || task.ParseError != "" || task.Order == nil {
		t.Errorf("illustration should be an ordered DONE card: %+v", task)
	}
	changelog := tasks["draft_changelog_for_v2_release.md"]
	if !strings.HasPrefix(changelog.Content, "# ") || strings.Count(changelog.Content, "\n---\n") != 3 {
		t.Errorf("changelog lost its heading or dividers: %q", changelog.Content)
	}
	plan := tasks["document_smoke_test_plan.md"]
	if strings.Contains(plan.Content, strings.Repeat("<", 7)) {
		t.Error("smoke plan contains a conflict marker")
	}

	t.Run("ordering and filter setup", func(t *testing.T) {
		slices.SortFunc(todo, board.ByOrder)
		want := []string{"test_frontend.md", "read_prd.md", "wire_up_analytics_events.md", "polish_mobile_nav_transitions.md"}
		var files, visible []string
		filter := board.Filter{Tags: map[string]bool{"docs": true, "mobile": true}}
		for _, task := range todo {
			files = append(files, task.File)
			if filter.Visible(task) {
				visible = append(visible, task.File)
			}
		}
		if !slices.Equal(files, want) {
			t.Errorf("TODO = %v, want %v", files, want)
		}
		if !slices.Equal(visible, []string{"read_prd.md", "polish_mobile_nav_transitions.md"}) {
			t.Errorf("visible TODO = %v", visible)
		}
	})

	t.Run("resolve existing conflict", func(t *testing.T) {
		file := "resolve_merge_conflict_in_onboarding.md"
		content := tasks[file].Content
		start, end := strings.Index(content, "<<<<<<< HEAD\n"), strings.Index(content, ">>>>>>> release/1.4\n")
		if start < 0 || end < start || strings.Count(content[start:end], "\n=======\n") != 1 {
			t.Fatal("onboarding fixture needs a well-formed conflict block")
		}
		resolved := content[:start] + "Step 3: Ask your onboarding buddy to add you to #eng-general and #eng-questions.\n" + content[end+len(">>>>>>> release/1.4\n"):]
		writeTaskFile(t, dir, file, resolved)
		if err := store.Scan(); err != nil {
			t.Fatal(err)
		}
		task, err := board.ParseTask(filepath.Join(dir, file))
		if err != nil {
			t.Fatal(err)
		}
		if task.Status != "TODO" || task.Project != "MK" || task.Epic != "QA" || task.Order == nil || *task.Order != 4 || !slices.Equal(task.Tags, []string{"docs", "onboarding"}) || strings.HasPrefix(task.Content, "---") {
			t.Fatalf("resolved onboarding frontmatter/body = %+v", task)
		}
		for _, section := range board.Sections(store.Snapshot()) {
			if section.Name == board.StatusConflict {
				t.Error("CONFLICT remains after resolving the only conflict")
			}
		}
	})

	t.Run("clear pricing body", func(t *testing.T) {
		file := "draft_pricing_page_copy.md"
		if strings.TrimSpace(tasks[file].Content) == "" {
			t.Fatal("pricing fixture must start with a body to clear")
		}
		empty := ""
		if err := store.Apply(board.UpdateRequest{File: file, Content: &empty}); err != nil {
			t.Fatal(err)
		}
		task, err := board.ParseTask(filepath.Join(dir, file))
		if err != nil || task.Content != "" || task.Status != "UNDER REVIEW" {
			t.Fatalf("cleared pricing card = %+v, error = %v", task, err)
		}
	})
}
