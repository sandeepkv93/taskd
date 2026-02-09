package update

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleInboxKey(msg tea.KeyMsg) Model {
	if m.Inbox.CaptureMode {
		switch msg.String() {
		case "esc":
			m.Inbox.CaptureMode = false
			m.quickAddInput.Blur()
			m.Status = StatusBar{Text: "inbox list mode", IsError: false}
			return m
		case "enter":
			m.addInboxItem(m.quickAddInput.Value())
			m.quickAddInput.SetValue("")
			m.Inbox.Input = ""
			return m
		}
		var cmd tea.Cmd
		m.quickAddInput, cmd = m.quickAddInput.Update(msg)
		_ = cmd
		m.Inbox.Input = m.quickAddInput.Value()
		return m
	}

	switch msg.String() {
	case "i":
		m.Inbox.CaptureMode = true
		m.quickAddInput.Focus()
		m.Status = StatusBar{Text: "inbox capture mode", IsError: false}
	case "enter":
		m.Inbox.CaptureMode = true
		m.quickAddInput.Focus()
		m.Status = StatusBar{Text: "inbox capture mode", IsError: false}
	case "up", "k":
		if m.Inbox.Cursor > 0 {
			m.Inbox.Cursor--
		}
	case "down", "j":
		if m.Inbox.Cursor < len(m.Inbox.Items)-1 {
			m.Inbox.Cursor++
		}
	case " ":
		m.toggleSelectedAtCursor()
	case "x":
		m.selectAllInboxItems()
	case "u":
		m.clearInboxSelection()
	case "s":
		m.bulkScheduleInbox("tomorrow 09:00")
	case "g":
		m.bulkTagInbox("triage")
	default:
		if msg.Type == tea.KeyRunes {
			m.Inbox.CaptureMode = true
			m.quickAddInput.Focus()
			m.quickAddInput.SetValue(string(msg.Runes))
			m.Inbox.Input = m.quickAddInput.Value()
			return m
		}
		var cmd tea.Cmd
		m.quickAddInput, cmd = m.quickAddInput.Update(msg)
		_ = cmd
		m.Inbox.Input = m.quickAddInput.Value()
	}
	return m
}

func (m *Model) addInboxItem(title string) {
	m.ensureInboxState()
	trimmed := strings.TrimSpace(title)
	if trimmed == "" {
		return
	}
	item := InboxItem{
		ID:    fmt.Sprintf("inbox-%d", m.Inbox.NextID),
		Title: trimmed,
	}
	m.Inbox.NextID++
	m.Inbox.Items = append(m.Inbox.Items, item)
	m.Inbox.Input = ""
	m.Inbox.Cursor = len(m.Inbox.Items) - 1
	m.Status = StatusBar{Text: "inbox item captured", IsError: false}
}

func (m *Model) toggleSelectedAtCursor() {
	if len(m.Inbox.Items) == 0 {
		return
	}
	item := m.Inbox.Items[m.Inbox.Cursor]
	if m.Inbox.Selected[item.ID] {
		delete(m.Inbox.Selected, item.ID)
		return
	}
	m.Inbox.Selected[item.ID] = true
}

func (m *Model) selectAllInboxItems() {
	m.ensureInboxState()
	for _, item := range m.Inbox.Items {
		m.Inbox.Selected[item.ID] = true
	}
	if len(m.Inbox.Items) > 0 {
		m.Status = StatusBar{Text: fmt.Sprintf("%d items selected", len(m.Inbox.Items)), IsError: false}
	}
}

func (m *Model) clearInboxSelection() {
	m.Inbox.Selected = make(map[string]bool)
}

func (m *Model) bulkScheduleInbox(when string) {
	m.ensureInboxState()
	applied := 0
	for i := range m.Inbox.Items {
		item := m.Inbox.Items[i]
		if m.Inbox.Selected[item.ID] {
			m.Inbox.Items[i].ScheduledFor = when
			applied++
		}
	}
	if applied > 0 {
		m.Status = StatusBar{Text: fmt.Sprintf("scheduled %d inbox items", applied), IsError: false}
	}
}

func (m *Model) bulkTagInbox(tag string) {
	m.ensureInboxState()
	applied := 0
	for i := range m.Inbox.Items {
		item := m.Inbox.Items[i]
		if m.Inbox.Selected[item.ID] {
			if !contains(item.Tags, tag) {
				m.Inbox.Items[i].Tags = append(m.Inbox.Items[i].Tags, tag)
			}
			applied++
		}
	}
	if applied > 0 {
		m.Status = StatusBar{Text: fmt.Sprintf("tagged %d inbox items", applied), IsError: false}
	}
}
