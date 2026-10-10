package board

import (
	"reflect"
	"slices"
	"testing"
)

func TestComputeOrder(t *testing.T) {
	for _, tc := range []struct {
		prev, next *float64
		want       float64
	}{{number(2), number(4), 3}, {number(2), nil, 3}, {nil, number(4), 3}, {nil, nil, 1}} {
		if got := ComputeOrder(tc.prev, tc.next); got != tc.want {
			t.Fatalf("got %v want %v", got, tc.want)
		}
	}
}
func TestDropOrder(t *testing.T) {
	column := []Task{{File: "a", Order: number(1)}, {File: "hidden", Order: number(2)}, {File: "c", Order: number(3)}, {File: "nil"}}
	for _, tc := range []struct {
		name, before, last string
		column             []Task
		want               float64
	}{
		{"first", "a", "", column, 0}, {"between", "hidden", "", column, 1.5},
		{"hidden previous", "c", "", column, 2.5}, {"hidden next", "", "a", column, 1.5},
		{"end", "", "c", column[:3], 4}, {"empty", "", "", nil, 1},
		{"no visible", "", "", column, 1}, {"unset next", "nil", "", column, 4},
		{"unset previous", "", "nil", column, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := DropOrder(tc.column, tc.before, tc.last); got != tc.want {
				t.Fatalf("got %v want %v", got, tc.want)
			}
		})
	}
}
func TestDropOrderAfterUnorderedCard(t *testing.T) {
	for _, tc := range []struct {
		name   string
		column []Task
		last   string
		want   []string
	}{
		{"all unordered", []Task{{File: "b"}, {File: "c"}}, "c", []string{"b", "c", "a"}},
		{"mixed", []Task{{File: "ordered", Order: number(10)}, {File: "b"}, {File: "c"}}, "c", []string{"ordered", "b", "c", "a"}},
		{"hidden next", []Task{{File: "ordered", Order: number(10)}, {File: "b"}, {File: "hidden"}}, "b", []string{"ordered", "b", "a", "hidden"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			original := cloneTasks(tc.column)
			order, updates := DropOrder(tc.column, "", tc.last)
			if !reflect.DeepEqual(tc.column, original) {
				t.Fatal("drop mutated input")
			}
			column := cloneTasks(tc.column)
			for _, update := range updates {
				if update.Order == nil {
					t.Fatal("normalization update has no order")
				}
				index := slices.IndexFunc(column, func(task Task) bool { return task.File == update.File })
				if index < 0 || column[index].Order != nil {
					t.Fatalf("unexpected normalization update for %q", update.File)
				}
				column[index].Order = update.Order
			}
			column = append(column, Task{File: "a", Order: &order})
			slices.SortFunc(column, ByOrder)
			var files []string
			for _, task := range column {
				files = append(files, task.File)
			}
			if !reflect.DeepEqual(files, tc.want) {
				t.Fatalf("sorted drop=%v want %v", files, tc.want)
			}
		})
	}
}
func TestSectionsAndTableOrder(t *testing.T) {
	tasks := []Task{{File: "other", Status: "unknown"}, {File: "done", Status: "DONE"}, {File: "b", Status: "TODO", Order: number(2)}, {File: "conflict", Status: StatusConflict}, {File: "a", Status: "TODO", Order: number(2)}, {File: "parse", Status: "TODO", ParseError: "bad"}}
	columns := []string{"TODO", "REVIEW", "DONE"}
	want := []SectionInfo{{StatusConflict, false}, {"TODO", true}, {"REVIEW", true}, {"DONE", true}, {StatusOther, false}}
	if got := Sections(Snapshot{Columns: columns, Tasks: tasks}); !reflect.DeepEqual(got, want) {
		t.Fatalf("sections=%v", got)
	}
	sorted := SortForTable(tasks, columns)
	var files []string
	for _, task := range sorted {
		files = append(files, task.File)
	}
	if !reflect.DeepEqual(files, []string{"conflict", "a", "b", "done", "other", "parse"}) {
		t.Fatalf("order=%v", files)
	}
	if tasks[0].File != "other" {
		t.Fatal("sort mutated input")
	}
	if got := Sections(Snapshot{Columns: columns}); !reflect.DeepEqual(got, want[1:4]) {
		t.Fatalf("empty sections=%v", got)
	}
	if ByOrder(Task{File: "a"}, Task{File: "b", Order: number(100)}) <= 0 {
		t.Fatal("nil order should sort last")
	}
	if Section(Task{}, columns) != StatusOther {
		t.Fatal("empty status should render in OTHER")
	}
}
