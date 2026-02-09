package update

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type View string

const (
	ViewToday    View = "Today"
	ViewInbox    View = "Inbox"
	ViewCalendar View = "Calendar"
	ViewFocus    View = "Focus"
)

type SortOrder string

const (
	SortCreatedDesc SortOrder = "created_desc"
	SortCreatedAsc  SortOrder = "created_asc"
)

type FilterState struct {
	Tag      string
	State    string
	Priority string
}

type StatusBar struct {
	Text    string
	IsError bool
}

type GlobalKeyMap struct {
	Today    string
	Inbox    string
	Calendar string
	Focus    string
	Help     string
	Quit     string
}

type Model struct {
	CurrentView    View
	SelectedTaskID string
	Filter         FilterState
	Sort           SortOrder
	Inbox          InboxState
	Status         StatusBar
	Keys           GlobalKeyMap
	Quitting       bool
	LastError      error
}

type InboxItem struct {
	ID           string
	Title        string
	ScheduledFor string
	Tags         []string
}

type InboxState struct {
	Input    string
	Items    []InboxItem
	Cursor   int
	Selected map[string]bool
	NextID   int
}

type SwitchViewMsg struct {
	View View
}

type SetStatusMsg struct {
	Text    string
	IsError bool
}

type ClearStatusMsg struct{}

type AppErrorMsg struct {
	Err error
}

type QuickAddInboxTaskMsg struct {
	Title string
}

type BulkScheduleInboxMsg struct {
	When string
}

type BulkTagInboxMsg struct {
	Tag string
}

func NewModel() Model {
	return Model{
		CurrentView: ViewToday,
		Sort:        SortCreatedDesc,
		Inbox: InboxState{
			Selected: make(map[string]bool),
			NextID:   1,
		},
		Keys: GlobalKeyMap{
			Today:    "1",
			Inbox:    "2",
			Calendar: "3",
			Focus:    "4",
			Help:     "?",
			Quit:     "q",
		},
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		m.ensureInboxState()
		switch typed.String() {
		case m.Keys.Today:
			m.CurrentView = ViewToday
			return m, nil
		case m.Keys.Inbox:
			m.CurrentView = ViewInbox
			return m, nil
		case m.Keys.Calendar:
			m.CurrentView = ViewCalendar
			return m, nil
		case m.Keys.Focus:
			m.CurrentView = ViewFocus
			return m, nil
		case "ctrl+c", m.Keys.Quit:
			m.Quitting = true
			return m, tea.Quit
		}
		if m.CurrentView == ViewInbox {
			return m.handleInboxKey(typed), nil
		}
	case SwitchViewMsg:
		if isKnownView(typed.View) {
			m.CurrentView = typed.View
		}
		return m, nil
	case SetStatusMsg:
		m.Status = StatusBar{Text: typed.Text, IsError: typed.IsError}
		return m, nil
	case ClearStatusMsg:
		m.Status = StatusBar{}
		return m, nil
	case AppErrorMsg:
		m.LastError = typed.Err
		if typed.Err != nil {
			m.Status = StatusBar{Text: typed.Err.Error(), IsError: true}
		}
		return m, nil
	case QuickAddInboxTaskMsg:
		m.addInboxItem(typed.Title)
		return m, nil
	case BulkScheduleInboxMsg:
		m.bulkScheduleInbox(typed.When)
		return m, nil
	case BulkTagInboxMsg:
		m.bulkTagInbox(typed.Tag)
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	status := ""
	if m.Status.Text != "" {
		if m.Status.IsError {
			status = fmt.Sprintf("\nstatus: error: %s", m.Status.Text)
		} else {
			status = fmt.Sprintf("\nstatus: %s", m.Status.Text)
		}
	}
	inboxView := ""
	if m.CurrentView == ViewInbox {
		inboxView = m.renderInboxView()
	}
	return fmt.Sprintf(
		"taskd | view: %s | selected: %s\nkeys: [%s]today [%s]inbox [%s]calendar [%s]focus [%s]quit%s%s",
		m.CurrentView,
		m.SelectedTaskID,
		m.Keys.Today,
		m.Keys.Inbox,
		m.Keys.Calendar,
		m.Keys.Focus,
		m.Keys.Quit,
		status,
		inboxView,
	)
}

func isKnownView(v View) bool {
	switch v {
	case ViewToday, ViewInbox, ViewCalendar, ViewFocus:
		return true
	default:
		return false
	}
}

func (m *Model) ensureInboxState() {
	if m.Inbox.Selected == nil {
		m.Inbox.Selected = make(map[string]bool)
	}
	if m.Inbox.NextID <= 0 {
		m.Inbox.NextID = 1
	}
}

func (m Model) handleInboxKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "enter":
		m.addInboxItem(m.Inbox.Input)
	case "backspace":
		if len(m.Inbox.Input) > 0 {
			m.Inbox.Input = m.Inbox.Input[:len(m.Inbox.Input)-1]
		}
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
			m.Inbox.Input += string(msg.Runes)
		}
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

func (m Model) renderInboxView() string {
	var b strings.Builder
	b.WriteString("\ninbox:\n")
	b.WriteString(fmt.Sprintf("quick-add: %s\n", m.Inbox.Input))
	b.WriteString("actions: [enter]add [space]select [x]all [u]clear [s]schedule [g]tag\n")
	if len(m.Inbox.Items) == 0 {
		b.WriteString("(empty)")
		return b.String()
	}
	for i, item := range m.Inbox.Items {
		cursor := " "
		if i == m.Inbox.Cursor {
			cursor = ">"
		}
		selected := " "
		if m.Inbox.Selected[item.ID] {
			selected = "x"
		}
		tags := ""
		if len(item.Tags) > 0 {
			tags = " tags=" + strings.Join(item.Tags, ",")
		}
		scheduled := ""
		if item.ScheduledFor != "" {
			scheduled = " scheduled=" + item.ScheduledFor
		}
		b.WriteString(fmt.Sprintf("%s[%s] %s%s%s\n", cursor, selected, item.Title, scheduled, tags))
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
