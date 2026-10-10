package tui

func layoutColumns(width, n, focus int) (first, count, colWidth int) {
	if n <= 0 {
		return 0, 0, 0
	}
	width = max(1, width)
	count = min(n, max(1, (width+1)/27))
	first = max(0, min(focus-count+1, n-count))
	colWidth = max(1, (width-count+1)/count)
	return
}
