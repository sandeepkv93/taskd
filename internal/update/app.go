package update

import (
	"fmt"

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
	Status         StatusBar
	Keys           GlobalKeyMap
	Quitting       bool
	LastError      error
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

func NewModel() Model {
	return Model{
		CurrentView: ViewToday,
		Sort:        SortCreatedDesc,
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
	return fmt.Sprintf(
		"taskd | view: %s | selected: %s\nkeys: [%s]today [%s]inbox [%s]calendar [%s]focus [%s]quit%s",
		m.CurrentView,
		m.SelectedTaskID,
		m.Keys.Today,
		m.Keys.Inbox,
		m.Keys.Calendar,
		m.Keys.Focus,
		m.Keys.Quit,
		status,
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
