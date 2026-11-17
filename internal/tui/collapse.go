package tui

import (
	"fmt"
	"slices"
)

func (m *Model) collapsed(name string) bool { return slices.Contains(m.prefs.Collapsed, name) }
func (m *Model) collapsedColumns() []bool {
	result := make([]bool, len(m.sections))
	for i, s := range m.sections {
		result[i] = m.collapsed(s.Name)
	}
	return result
}
func (m *Model) moveSection(step int) {
	for i := m.focusSec + step; i >= 0 && i < len(m.sections); i += step {
		if !m.collapsed(m.sections[i].Name) {
			m.focusSec = i
			return
		}
	}
}
func (m *Model) toggleColumn() {
	if len(m.sections) == 0 {
		return
	}
	name := m.sections[m.focusSec].Name
	if i := slices.Index(m.prefs.Collapsed, name); i >= 0 {
		m.prefs.Collapsed = slices.Delete(m.prefs.Collapsed, i, i+1)
	} else {
		expanded := 0
		for _, s := range m.sections {
			if !m.collapsed(s.Name) {
				expanded++
			}
		}
		if expanded <= 1 {
			m.actionError(fmt.Errorf("can't collapse every column"))
			return
		}
		m.prefs.Collapsed = append(m.prefs.Collapsed, name)
		old := m.focusSec
		m.moveSection(1)
		if m.focusSec == old {
			m.moveSection(-1)
		}
	}
	savePrefs(m.prefsFile, m.prefs)
}
