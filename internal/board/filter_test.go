package board

import (
	"reflect"
	"testing"
)

func TestFilterVisible(t *testing.T) {
	f := Filter{Projects: map[string]bool{"mk": true, "other": true}, Epics: map[string]bool{NoEpic: true}, Tags: map[string]bool{"a": true, "b": true}}
	if f.Active() != 5 {
		t.Fatalf("active=%d", f.Active())
	}
	for _, tc := range []struct {
		name string
		task Task
		want bool
	}{
		{"all groups", Task{Project: "mk", Tags: []string{"b", "unselected"}}, true},
		{"project OR", Task{Project: "other", Tags: []string{"a"}}, true},
		{"no project", Task{Tags: []string{"a"}}, false},
		{"epic AND", Task{Project: "mk", Epic: "some", Tags: []string{"a"}}, false},
		{"tags AND", Task{Project: "mk", Tags: []string{"unselected"}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := f.Visible(tc.task); got != tc.want {
				t.Fatalf("got=%v", got)
			}
		})
	}
	if !(Filter{}).Visible(Task{}) {
		t.Fatal("empty groups should match")
	}
	if !(Filter{Projects: map[string]bool{"mk": false}}).Visible(Task{}) {
		t.Fatal("false chip is not selected")
	}
}
func TestFacets(t *testing.T) {
	projects, epics, tags := Facets([]Task{{Project: "z", Tags: []string{"b", "a"}}, {Project: "a", Epic: "work", Tags: []string{"a"}}})
	if !reflect.DeepEqual(projects, []string{"a", "z"}) || !reflect.DeepEqual(epics, []string{NoEpic, "work"}) || !reflect.DeepEqual(tags, []string{"a", "b"}) {
		t.Fatalf("facets=%v %v %v", projects, epics, tags)
	}
}

func TestFilterQuery(t *testing.T) {
	task := Task{Title: "mk-tui-1-core-store-extraction", Project: "Mavmark", Epic: "Terminal UI", Tags: []string{"bubbletea"}, DependsOn: []string{"core-store"}, Content: "Only the body mentions needle."}
	for _, query := range []string{"tuicore", "TUICORE", "mav", "terminal", "bubble", "core-store", "needle", "tuicore needle", "  "} {
		t.Run(query, func(t *testing.T) {
			f := Filter{Query: query}
			if !f.Visible(task) {
				t.Fatal("expected query hit")
			}
		})
	}
	for _, query := range []string{"absent", "tuicore absent", "nede"} {
		if (Filter{Query: query}).Visible(task) {
			t.Fatalf("unexpected hit: %q", query)
		}
	}
	f := Filter{Query: "needle", Projects: map[string]bool{"other": true}}
	if f.Visible(task) {
		t.Fatal("query must AND with chips")
	}
	f = Filter{Query: "needle", Tags: map[string]bool{"bubbletea": true}}
	if !f.Visible(task) {
		t.Fatal("tag and body query must match")
	}
	if (Filter{Query: "needle"}).Active() != 1 || (Filter{Query: "  "}).Active() != 0 {
		t.Fatal("query active count")
	}
	if (Filter{Query: "needle"}).Score(task) != 0 {
		t.Fatal("body must not contribute score")
	}
}
