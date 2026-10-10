package board

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseTaskSurvivesMalformedFrontmatter(t *testing.T) {
	dir := t.TempDir()
	// order: expects a float; tags: expects a list. Either is a type
	// mismatch yaml.Unmarshal will reject.
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\norder: soon\n---\nbody")

	task, err := ParseTask(filepath.Join(dir, "bad.md"))
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

func TestParseTaskReadsDependsOn(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "blocked.md", "---\nstatus: TODO\ndepends_on: [design, api]\n---\nbody")

	task, err := ParseTask(filepath.Join(dir, "blocked.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if !equalStrings(task.DependsOn, []string{"design", "api"}) {
		t.Fatalf("DependsOn = %v, want [design api]", task.DependsOn)
	}
}

func TestParseTaskLeavesDependsOnEmptyWhenAbsent(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "standalone.md", "---\nstatus: TODO\n---\nbody")

	task, err := ParseTask(filepath.Join(dir, "standalone.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if len(task.DependsOn) != 0 {
		t.Fatalf("DependsOn = %v, want empty", task.DependsOn)
	}
}

func TestTaskJSONIncludesDependsOnOnlyWhenPresent(t *testing.T) {
	withDependencies, err := json.Marshal(Task{DependsOn: []string{"design", "api"}})
	if err != nil {
		t.Fatalf("json.Marshal with dependencies: %v", err)
	}
	if !strings.Contains(string(withDependencies), `"dependsOn":["design","api"]`) {
		t.Fatalf("JSON = %s, want dependsOn", withDependencies)
	}

	withoutDependencies, err := json.Marshal(Task{})
	if err != nil {
		t.Fatalf("json.Marshal without dependencies: %v", err)
	}
	if strings.Contains(string(withoutDependencies), `"dependsOn"`) {
		t.Fatalf("JSON = %s, did not want dependsOn", withoutDependencies)
	}
}

func TestParseTaskRejectsScalarDependsOn(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "bad.md", "---\nstatus: TODO\ndepends_on: design\n---\nbody")

	task, err := ParseTask(filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("parseTask returned an error: %v", err)
	}
	if task.Status != "" || task.ParseError == "" {
		t.Fatalf("task = %+v, want parse error routed to OTHER", task)
	}
}

// A card whose frontmatter fails to parse must round-trip its real body
// unchanged through a save that doesn't touch the editor (e.g. adding a
// tag) — Content must never be the synthetic error message, or a save like
// this would silently overwrite the file's real body with it.
func TestParseTaskRejectsNonFiniteOrder(t *testing.T) {
	for _, value := range []string{".nan", ".inf", "-.inf"} {
		t.Run(value, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nonfinite.md")
			if err := os.WriteFile(path, []byte("---\nstatus: TODO\norder: "+value+"\n---\nbody"), 0644); err != nil {
				t.Fatal(err)
			}
			task, err := ParseTask(path)
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

func TestParseTaskIgnoresNonLeadingThematicBreaks(t *testing.T) {
	dir := t.TempDir()
	writeTaskFile(t, dir, "notes.md", "# My Notes\n\nintro\n\n---\n\nmiddle\n\n---\n\nend\n")

	task, err := ParseTask(filepath.Join(dir, "notes.md"))
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
	task, err := ParseTask(filepath.Join(dir, "with_order.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order == nil || *task.Order != 1.5 {
		t.Fatalf("Order = %v, want *float64(1.5)", task.Order)
	}

	writeTaskFile(t, dir, "no_order.md", "---\nstatus: TODO\n---\nbody")
	task, err = ParseTask(filepath.Join(dir, "no_order.md"))
	if err != nil {
		t.Fatalf("parseTask error: %v", err)
	}
	if task.Order != nil {
		t.Fatalf("Order = %v, want nil", task.Order)
	}

	writeTaskFile(t, dir, "zero_order.md", "---\nstatus: TODO\norder: 0\n---\nbody")
	task, err = ParseTask(filepath.Join(dir, "zero_order.md"))
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
func writeTaskFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
