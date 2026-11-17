package tui

import (
	tea "charm.land/bubbletea/v2"
	"github.com/Syntax-Syllogism/mk/internal/board"
)

type rect struct{ x, y, w, h int }

func (r rect) contains(x, y int) bool { return x >= r.x && y >= r.y && x < r.x+r.w && y < r.y+r.h }

type hitKind int

const (
	hitColumn hitKind = iota
	hitCard
	hitChip
	hitFilterGroup
	hitFilters
	hitDependency
)

type hit struct {
	kind          hitKind
	section, file string
	row, chip     int
}
type region struct {
	r rect
	h hit
}
type layout struct{ hits []region }

func (m *Model) addHit(r rect, h hit) {
	// Only visible cells are interactive, including cards clipped at the bottom.
	r.w = min(r.w, m.width-r.x)
	r.h = min(r.h, m.height-1-r.y)
	if r.w > 0 && r.h > 0 {
		m.layout.hits = append(m.layout.hits, region{r, h})
	}
}
func (m *Model) openSelected() {
	if file := m.currentFile(); file != "" {
		m.prevMode = m.mode
		m.mode = modePreview
		m.preview.back = nil
		m.openPreview(file)
	}
}
func (m *Model) updateMouse(msg tea.MouseMsg) {
	if m.noMouse {
		return
	}
	mouse := msg.Mouse()
	if m.mode == modeHelp {
		if _, ok := msg.(tea.MouseWheelMsg); ok {
			m.scrollOverlay(mouse.Button)
		}
		return
	}
	if m.grab != nil || m.tags != nil || m.search != nil || m.filterFind != nil || m.preview.find != nil {
		return
	}
	if _, ok := msg.(tea.MouseWheelMsg); ok {
		if m.mode == modePreview {
			m.scrollPreview(mouse.Button)
			return
		}
		if m.mode == modeLogs {
			m.scrollOverlay(mouse.Button)
			return
		}
		for _, r := range m.layout.hits {
			if r.h.kind == hitColumn && m.collapsed(r.h.section) && r.r.contains(mouse.X, mouse.Y) {
				return
			}
		}
		if mouse.Button == tea.MouseWheelLeft || mouse.Button == tea.MouseWheelRight || mouse.Mod&tea.ModShift != 0 {
			if m.mode != modeBoard && !(m.mode == modeFilter && m.prevMode == modeBoard) {
				return
			}
			_, count, _ := layoutColumns(m.width, len(m.sections), m.focusSec, m.collapsedColumns())
			if count == len(m.sections) {
				return
			}
			step := 1
			if mouse.Button == tea.MouseWheelLeft || mouse.Button == tea.MouseWheelUp {
				step = -1
			}
			m.moveSection(step)
			return
		}
		step := 1
		if mouse.Button == tea.MouseWheelUp {
			step = -1
		}
		for _, r := range m.layout.hits {
			if r.h.kind == hitFilterGroup && r.r.contains(mouse.X, mouse.Y) {
				row := r.h.row
				g := m.filterLayout(row)
				m.filterScroll[row] = max(0, min(max(0, len(g.lines)-g.budget), m.filterScroll[row]+step))
				return
			}
		}
		if m.mode == modeTable || (m.mode == modeFilter && m.prevMode == modeTable) {
			tasks := m.tableTasks()
			h := max(1, m.contentHeight()-1)
			m.tableScroll = max(0, min(max(0, len(tasks)-h), m.tableScroll+3*step))
			m.tableCur = clampFile(tasks, m.tableCur, m.tableScroll, h)
			return
		}
		for _, r := range m.layout.hits {
			if r.h.kind == hitColumn && r.r.contains(mouse.X, mouse.Y) {
				section := r.h.section
				if m.collapsed(section) {
					return
				}
				tasks := m.visible(section)
				start := max(0, min(max(0, len(tasks)-1), m.scroll[section]+step))
				m.scroll[section] = start
				// Clamp selection to the cards that fit in this column's viewport.
				used := 0
				if start > 0 {
					used++
				}
				end := start
				for end < len(tasks) {
					size := lineCount(m.card(tasks[end], r.r.w, false))
					reserve := 0
					if end+1 < len(tasks) {
						reserve = 1
					}
					if used+size+reserve > r.r.h-1 && end > start {
						break
					}
					used += size
					end++
				}
				m.selected[section] = clampFile(tasks, m.selected[section], start, max(1, end-start))
				return
			}
		}
		return
	}
	if _, ok := msg.(tea.MouseClickMsg); !ok || mouse.Button != tea.MouseLeft || m.mode == modeLogs {
		return
	}
	for i := len(m.layout.hits) - 1; i >= 0; i-- {
		r := m.layout.hits[i]
		if !r.r.contains(mouse.X, mouse.Y) {
			continue
		}
		h := r.h
		switch h.kind {
		case hitFilters:
			m.prefs.FiltersCollapsed = false
			savePrefs(m.prefsFile, m.prefs)
		case hitChip:
			values := m.groupChips(h.row)
			if h.chip >= len(values) {
				return
			}
			m.toggleFilter(h.row, values[h.chip])
		case hitColumn, hitCard:
			if m.mode == modeFilter {
				m.mode = m.prevMode
			}
			if h.kind == hitCard && m.currentFile() == h.file {
				m.openSelected()
				return
			}
			if m.mode == modeTable {
				m.tableCur = h.file
				return
			}
			for j, s := range m.sections {
				if s.Name == h.section {
					m.focusSec = j
					if m.collapsed(s.Name) {
						m.toggleColumn()
					}
					break
				}
			}
			if h.kind == hitCard {
				m.selected[h.section] = h.file
			}
		case hitDependency:
			m.preview.depCursor = h.chip
			m.updatePreview(tea.KeyPressMsg{Code: tea.KeyEnter})
		}
		return
	}
}
func clampFile(tasks []board.Task, file string, start, height int) string {
	if len(tasks) == 0 {
		return ""
	}
	index := max(start, min(start+height-1, fileIndex(tasks, file)))
	return tasks[min(len(tasks)-1, index)].File
}
func (m *Model) scrollOverlay(button tea.MouseButton) {
	if button == tea.MouseWheelUp {
		m.overlay.ScrollUp(3)
	} else if button == tea.MouseWheelDown {
		m.overlay.ScrollDown(3)
	}
}
func (m *Model) scrollPreview(button tea.MouseButton) {
	if button == tea.MouseWheelUp {
		m.preview.viewport.ScrollUp(3)
	} else if button == tea.MouseWheelDown {
		m.preview.viewport.ScrollDown(3)
	}
}

func (m *Model) mouseMode() tea.MouseMode {
	if m.noMouse {
		return tea.MouseModeNone
	}
	return tea.MouseModeCellMotion
}
