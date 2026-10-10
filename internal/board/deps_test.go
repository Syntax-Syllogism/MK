package board

import (
	"reflect"
	"testing"
)

func TestAnalyzeDeps(t *testing.T) {
	for _, tc := range []struct {
		name   string
		tasks  []Task
		cycles []string
	}{
		{"none", []Task{{File: "a.md"}}, nil},
		{"missing", []Task{{File: "a.md", DependsOn: []string{"missing"}}}, nil},
		{"two", []Task{{File: "a.md", DependsOn: []string{"b"}}, {File: "b.md", DependsOn: []string{"a"}}}, []string{"a.md", "b.md"}},
		{"self", []Task{{File: "a.md", DependsOn: []string{"a"}}}, []string{"a.md"}},
		{"three and tail", []Task{{File: "a.md", DependsOn: []string{"b"}}, {File: "b.md", DependsOn: []string{"c"}}, {File: "c.md", DependsOn: []string{"a", "tail"}}, {File: "tail.md", DependsOn: []string{"leaf"}}, {File: "leaf.md"}, {File: "incoming.md", DependsOn: []string{"a"}}}, []string{"a.md", "b.md", "c.md"}},
		{"shared node", []Task{{File: "a.md", DependsOn: []string{"b", "c"}}, {File: "b.md", DependsOn: []string{"a"}}, {File: "c.md", DependsOn: []string{"a"}}}, []string{"a.md", "b.md", "c.md"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := AnalyzeDeps(tc.tasks)
			want := map[string]bool{}
			for _, file := range tc.cycles {
				want[file] = true
			}
			if !reflect.DeepEqual(got.Cycles, want) {
				t.Fatalf("cycles=%v want %v", got.Cycles, want)
			}
			if got.Missing("a") || !got.Missing("missing") {
				t.Fatalf("known=%v", got.Known)
			}
		})
	}
}
