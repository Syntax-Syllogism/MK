package tui

// layoutColumns keeps a contiguous window containing focus. Collapsed strips
// reserve five cells; expanded columns share the rest of the available width.
func layoutColumns(width, n, focus int, collapsed ...[]bool) (first, count, colWidth int) {
	if n <= 0 {
		return 0, 0, 0
	}
	width = max(1, width)
	focus = max(0, min(focus, n-1))
	isCollapsed := func(i int) bool { return len(collapsed) > 0 && i < len(collapsed[0]) && collapsed[0][i] }
	cost := func(i int) int {
		if isCollapsed(i) {
			return 5
		}
		return 26
	}
	first, end := focus, focus+1
	used := cost(focus)
	for first > 0 && used+1+cost(first-1) <= width {
		first--
		used += 1 + cost(first)
	}
	for end < n && used+1+cost(end) <= width {
		used += 1 + cost(end)
		end++
	}
	count = end - first
	strips := 0
	for i := first; i < end; i++ {
		if isCollapsed(i) {
			strips++
		}
	}
	expanded := count - strips
	colWidth = 5
	if expanded > 0 {
		colWidth = max(1, (width-(count-1)-5*strips)/expanded)
	}
	return
}
