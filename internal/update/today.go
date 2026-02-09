package update

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleTodayKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "up", "k":
		if m.Today.Cursor > 0 {
			m.Today.Cursor--
		}
		m.syncSelectedTaskToTodayCursor()
	case "down", "j":
		if m.Today.Cursor < len(m.Today.Items)-1 {
			m.Today.Cursor++
		}
		m.syncSelectedTaskToTodayCursor()
	}
	return m
}

func (m *Model) syncSelectedTaskToTodayCursor() {
	if selected, ok := m.currentTodayItem(); ok {
		m.SelectedTaskID = selected.ID
	}
}

func (m Model) currentTodayItem() (TodayItem, bool) {
	if len(m.Today.Items) == 0 {
		return TodayItem{}, false
	}
	if m.Today.Cursor < 0 || m.Today.Cursor >= len(m.Today.Items) {
		return TodayItem{}, false
	}
	return m.Today.Items[m.Today.Cursor], true
}
