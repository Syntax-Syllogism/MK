package board

import (
	"cmp"
	"math"
	"slices"
	"strings"
)

// ByOrder puts unset orders last. Filename ties use bytes rather than JS localeCompare.
func ByOrder(a, b Task) int {
	ao, bo := math.Inf(1), math.Inf(1)
	if a.Order != nil {
		ao = *a.Order
	}
	if b.Order != nil {
		bo = *b.Order
	}
	if c := cmp.Compare(ao, bo); c != 0 {
		return c
	}
	return strings.Compare(a.File, b.File)
}

// ComputeOrder interpolates between neighbours, or steps past a sole neighbour.
func ComputeOrder(prev, next *float64) float64 {
	if prev != nil && next != nil {
		return (*prev + *next) / 2
	}
	if prev != nil {
		return *prev + 1
	}
	if next != nil {
		return *next - 1
	}
	return 1
}

// DropOrder uses the full column, sorted by ByOrder, excluding the moving card.
// beforeFile is the visible card below the drop; lastVisibleFile is used for end drops.
// When an end drop follows an unordered card, updates assigns numeric orders to
// the unordered tail. Callers must apply these updates along with the returned
// order for the moving card. The input column is not modified.
func DropOrder(column []Task, beforeFile, lastVisibleFile string) (order float64, updates []UpdateRequest) {
	var prev, next *float64
	if beforeFile != "" {
		for i, t := range column {
			if t.File == beforeFile {
				next = t.Order
				if i > 0 {
					prev = column[i-1].Order
				}
				break
			}
		}
	} else if lastVisibleFile != "" {
		for i, t := range column {
			if t.File == lastVisibleFile {
				if t.Order == nil {
					column = cloneTasks(column)
					for j := range column {
						if column[j].Order == nil {
							var previous *float64
							if j > 0 {
								previous = column[j-1].Order
							}
							value := ComputeOrder(previous, nil)
							column[j].Order = &value
							updates = append(updates, UpdateRequest{File: column[j].File, Order: &value})
						}
					}
				}
				prev = column[i].Order
				if i+1 < len(column) {
					next = column[i+1].Order
				}
				break
			}
		}
	}
	return ComputeOrder(prev, next), updates
}

// Section returns CONFLICT, a configured column, or the diagnostic OTHER bucket.
func Section(t Task, columns []string) string {
	if t.Status == StatusConflict {
		return StatusConflict
	}
	if t.ParseError == "" && slices.Contains(columns, t.Status) {
		return t.Status
	}
	return StatusOther
}

type SectionInfo struct {
	Name      string
	Droppable bool
}

// Sections uses all tasks so filtering never removes diagnostic sections.
func Sections(s Snapshot) []SectionInfo {
	conflict, other := false, false
	for _, t := range s.Tasks {
		switch Section(t, s.Columns) {
		case StatusConflict:
			conflict = true
		case StatusOther:
			other = true
		}
	}
	var sections []SectionInfo
	if conflict {
		sections = append(sections, SectionInfo{StatusConflict, false})
	}
	for _, c := range s.Columns {
		sections = append(sections, SectionInfo{c, true})
	}
	if other {
		sections = append(sections, SectionInfo{StatusOther, false})
	}
	return sections
}

// SortForTable returns a copy ordered by displayed section, then ByOrder.
func SortForTable(tasks []Task, columns []string) []Task {
	ranks := map[string]int{StatusConflict: 0, StatusOther: len(columns) + 1}
	for i, c := range columns {
		ranks[c] = i + 1
	}
	result := cloneTasks(tasks)
	slices.SortFunc(result, func(a, b Task) int {
		if c := cmp.Compare(ranks[Section(a, columns)], ranks[Section(b, columns)]); c != 0 {
			return c
		}
		return ByOrder(a, b)
	})
	return result
}
