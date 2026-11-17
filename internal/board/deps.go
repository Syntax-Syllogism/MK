package board

// DepInfo describes known dependency targets and files belonging to cycles.
type DepInfo struct{ Known, Cycles, Archived map[string]bool }

func (d DepInfo) Missing(slug string) bool    { return !d.Known[slug+".md"] && !d.IsArchived(slug) }
func (d DepInfo) IsArchived(slug string) bool { return !d.Known[slug+".md"] && d.Archived[slug+".md"] }

// AnalyzeDeps uses Tarjan's strongly connected components algorithm. Missing edges are ignored.
func AnalyzeDeps(tasks []Task, archived ...map[string]bool) DepInfo {
	result := DepInfo{Known: map[string]bool{}, Cycles: map[string]bool{}}
	if len(archived) > 0 {
		result.Archived = archived[0]
	}
	byFile := map[string]Task{}
	for _, t := range tasks {
		result.Known[t.File] = true
		byFile[t.File] = t
	}
	indices, low := map[string]int{}, map[string]int{}
	onStack := map[string]bool{}
	var stack []string
	index := 0
	var visit func(string)
	visit = func(file string) {
		index++
		indices[file] = index
		low[file] = index
		stack = append(stack, file)
		onStack[file] = true
		self := false
		for _, slug := range byFile[file].DependsOn {
			next := slug + ".md"
			if !result.Known[next] {
				continue
			}
			if next == file {
				self = true
			}
			if indices[next] == 0 {
				visit(next)
				low[file] = min(low[file], low[next])
			} else if onStack[next] {
				low[file] = min(low[file], indices[next])
			}
		}
		if low[file] != indices[file] {
			return
		}
		var component []string
		for {
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			onStack[last] = false
			component = append(component, last)
			if last == file {
				break
			}
		}
		if len(component) > 1 || self {
			for _, member := range component {
				result.Cycles[member] = true
			}
		}
	}
	for _, t := range tasks {
		if indices[t.File] == 0 {
			visit(t.File)
		}
	}
	return result
}
