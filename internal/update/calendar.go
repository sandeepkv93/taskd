package update

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleCalendarKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "d":
		m.Calendar.Mode = CalendarModeDay
		m.Status = StatusBar{Text: "calendar mode: day", IsError: false}
	case "w":
		m.Calendar.Mode = CalendarModeWeek
		m.Status = StatusBar{Text: "calendar mode: week", IsError: false}
	case "m":
		m.Calendar.Mode = CalendarModeMonth
		m.Status = StatusBar{Text: "calendar mode: month", IsError: false}
	case "h", "left":
		m.shiftCalendarFocus(-1)
	case "l", "right":
		m.shiftCalendarFocus(1)
	case "up", "k":
		if m.Calendar.Cursor > 0 {
			m.Calendar.Cursor--
		}
		m.syncSelectedTaskToCalendarCursor()
	case "down", "j":
		if m.Calendar.Cursor < len(m.Calendar.Items)-1 {
			m.Calendar.Cursor++
		}
		m.syncSelectedTaskToCalendarCursor()
	}
	return m
}

func (m *Model) shiftCalendarFocus(delta int) {
	switch m.Calendar.Mode {
	case CalendarModeDay:
		m.Calendar.FocusDate = m.Calendar.FocusDate.AddDate(0, 0, delta)
	case CalendarModeMonth:
		m.Calendar.FocusDate = m.Calendar.FocusDate.AddDate(0, delta, 0)
	default:
		m.Calendar.FocusDate = m.Calendar.FocusDate.AddDate(0, 0, 7*delta)
	}
	m.Status = StatusBar{
		Text:    fmt.Sprintf("calendar focus: %s", m.Calendar.FocusDate.Format("2006-01-02")),
		IsError: false,
	}
}

func (m *Model) syncSelectedTaskToCalendarCursor() {
	if selected, ok := m.currentAgendaItem(); ok {
		m.SelectedTaskID = selected.ID
	}
}

func (m Model) currentAgendaItem() (AgendaItem, bool) {
	if len(m.Calendar.Items) == 0 {
		return AgendaItem{}, false
	}
	if m.Calendar.Cursor < 0 || m.Calendar.Cursor >= len(m.Calendar.Items) {
		return AgendaItem{}, false
	}
	return m.Calendar.Items[m.Calendar.Cursor], true
}

func (m *Model) ensureCalendarState() {
	if m.Calendar.Mode == "" {
		m.Calendar.Mode = CalendarModeWeek
	}
	if m.Calendar.FocusDate.IsZero() {
		m.Calendar.FocusDate = time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)
	}
	if m.Calendar.Cursor < 0 {
		m.Calendar.Cursor = 0
	}
	if m.Calendar.Cursor >= len(m.Calendar.Items) && len(m.Calendar.Items) > 0 {
		m.Calendar.Cursor = len(m.Calendar.Items) - 1
	}
	if len(m.Calendar.Items) > 0 && m.SelectedTaskID == "" {
		m.syncSelectedTaskToCalendarCursor()
	}
}
