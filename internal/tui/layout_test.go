package tui

import "testing"

func TestLayoutColumns(t *testing.T) {
	for _, tc := range []struct{ width, n, focus, first, count, col int }{{80, 4, 0, 0, 3, 26}, {80, 4, 3, 1, 3, 26}, {200, 4, 3, 0, 4, 49}, {80, 6, 0, 0, 3, 26}, {80, 6, 5, 3, 3, 26}, {20, 4, 3, 3, 1, 20}, {1, 2, 0, 0, 1, 1}, {80, 0, 0, 0, 0, 0}} {
		first, count, col := layoutColumns(tc.width, tc.n, tc.focus)
		if first != tc.first || count != tc.count || col != tc.col {
			t.Errorf("%+v: got %d,%d,%d", tc, first, count, col)
		}
	}
}
