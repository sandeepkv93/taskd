package update

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/views"
)

func (m Model) Init() tea.Cmd {
	if m.Scheduler != nil {
		return waitForReminderCmd(m.Scheduler.C())
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	defer m.syncBubbleData()

	switch typed := msg.(type) {
	case tea.KeyMsg:
		m.ensureInboxState()
		m.ensureTodayState()
		m.ensureCalendarState()

		if m.Palette.Active {
			if typed.String() == m.Keys.Help {
				m.HelpVisible = !m.HelpVisible
				return m, nil
			}
			next := m.handlePaletteKey(typed)
			return next, nil
		}

		if m.recurrenceEditor.Active {
			next := m.handleRecurrenceEditorKey(typed)
			return next, nil
		}

		keyStr := typed.String()
		if m.CurrentView == ViewInbox && m.Inbox.CaptureMode && keyStr != "ctrl+c" &&
			keyStr != m.Keys.Today && keyStr != m.Keys.Inbox && keyStr != m.Keys.Calendar && keyStr != m.Keys.Focus &&
			keyStr != m.Keys.Help && keyStr != "/" && keyStr != m.Keys.Quit {
			return m.handleInboxKey(typed), nil
		}

		switch keyStr {
		case "/":
			m.Palette.Active = true
			m.Palette.Input = ""
			m.commandInput.Focus()
			m.commandInput.SetValue("")
			m.Status = StatusBar{Text: "command palette active", IsError: false}
			return m, nil
		case m.Keys.Today:
			m.CurrentView = ViewToday
			return m, nil
		case m.Keys.Inbox:
			m.CurrentView = ViewInbox
			m.Inbox.CaptureMode = true
			m.quickAddInput.Focus()
			return m, nil
		case m.Keys.Calendar:
			m.CurrentView = ViewCalendar
			return m, nil
		case m.Keys.Focus:
			m.CurrentView = ViewFocus
			m.bootstrapFocusTask()
			return m, nil
		case m.Keys.Help:
			m.HelpVisible = !m.HelpVisible
			if m.HelpVisible {
				m.Status = StatusBar{Text: "help shown", IsError: false}
			} else {
				m.Status = StatusBar{Text: "help hidden", IsError: false}
			}
			return m, nil
		case "S":
			if !m.spinnerActive {
				m.spinnerActive = true
				m.Status = StatusBar{Text: "sync started", IsError: false}
				return m, tea.Batch(m.syncSpinner.Tick, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return SetStatusMsg{Text: "sync complete", IsError: false} }))
			}
			return m, nil
		case "R":
			if m.CurrentView == ViewToday {
				m.recurrenceEditor.Active = true
				m.recurrenceEditor.Err = ""
				return m, nil
			}
		case "z":
			if m.CurrentView == ViewToday {
				m.toggleTodaySectionCollapse()
				return m, nil
			}
		case "D":
			m.cycleDensity()
			return m, nil
		case "ctrl+c", m.Keys.Quit:
			m.Quitting = true
			return m, tea.Quit
		}
		if m.CurrentView == ViewInbox {
			return m.handleInboxKey(typed), nil
		}
		if m.CurrentView == ViewToday {
			return m.handleTodayKey(typed), nil
		}
		if m.CurrentView == ViewCalendar {
			return m.handleCalendarKey(typed), nil
		}
		if m.CurrentView == ViewFocus {
			next, cmd := m.handleFocusKey(typed)
			return next, cmd
		}
	case spinner.TickMsg:
		if m.spinnerActive {
			var cmd tea.Cmd
			m.syncSpinner, cmd = m.syncSpinner.Update(typed)
			return m, cmd
		}
	case SwitchViewMsg:
		if isKnownView(typed.View) {
			m.CurrentView = typed.View
			if typed.View == ViewInbox {
				m.Inbox.CaptureMode = true
				m.quickAddInput.Focus()
			}
			if typed.View == ViewFocus {
				m.bootstrapFocusTask()
			}
		}
		return m, nil
	case SetStatusMsg:
		m.Status = StatusBar{Text: typed.Text, IsError: typed.IsError}
		if strings.Contains(strings.ToLower(typed.Text), "sync complete") {
			m.spinnerActive = false
		}
		m.notify("Status", typed.Text, levelFromError(typed.IsError))
		return m, nil
	case ClearStatusMsg:
		m.Status = StatusBar{}
		return m, nil
	case AppErrorMsg:
		m.LastError = typed.Err
		if typed.Err != nil {
			m.Status = StatusBar{Text: typed.Err.Error(), IsError: true}
			m.notify("Error", typed.Err.Error(), "error")
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
	case SetTodayItemsMsg:
		m.Today.Items = typed.Items
		m.Today.Cursor = 0
		m.syncSelectedTaskToTodayCursor()
		m.refreshProductivitySignals()
		return m, nil
	case SetCalendarItemsMsg:
		m.Calendar.Items = typed.Items
		m.Calendar.Cursor = 0
		m.syncSelectedTaskToCalendarCursor()
		return m, nil
	case FocusTickMsg:
		return m.onFocusTick()
	case ReminderDueMsg:
		m.ReminderLog = append(m.ReminderLog, typed.Event)
		if len(m.ReminderLog) > 20 {
			m.ReminderLog = m.ReminderLog[len(m.ReminderLog)-20:]
		}
		m.applyReminderBehavior(typed.Event, time.Now().UTC())
		m.notify("Reminder", m.Status.Text, levelFromError(m.Status.IsError))
		if m.Scheduler != nil {
			return m, waitForReminderCmd(m.Scheduler.C())
		}
		return m, nil
	case AcknowledgeReminderMsg:
		if typed.ID != "" {
			m.ReminderAck[typed.ID] = true
			m.Status = StatusBar{Text: fmt.Sprintf("reminder acknowledged: %s", typed.ID), IsError: false}
		}
		return m, nil
	}

	return m, nil
}

func (m Model) View() string {
	status := ""
	if m.Status.Text != "" {
		if m.Status.IsError {
			status = fmt.Sprintf("status: error: %s", m.Status.Text)
		} else {
			status = fmt.Sprintf("status: %s", m.Status.Text)
		}
	}
	leftPane := ""
	rightPane := ""
	switch m.CurrentView {
	case ViewInbox:
		leftPane = m.renderInboxView()
		rightPane = m.renderCommandPalette() + m.renderHelpIfVisible()
	case ViewToday:
		leftPane = m.renderTodayView()
		rightPane = m.renderTodayMetadataPane() + m.renderRecurrenceEditorIfVisible() + m.renderHelpIfVisible()
	case ViewCalendar:
		leftPane = m.renderCalendarView()
		rightPane = m.renderHelpIfVisible()
	case ViewFocus:
		leftPane = m.renderFocusView()
		rightPane = m.renderHelpIfVisible()
	}
	notificationView := ""
	if len(m.ReminderLog) > 0 {
		last := m.ReminderLog[len(m.ReminderLog)-1]
		notificationView = fmt.Sprintf("last-reminder: %s @ %s", last.ID, last.TriggerAt.Format("15:04:05"))
	}
	if m.spinnerActive {
		spin := m.syncSpinner.View()
		notificationView = strings.TrimSpace(strings.Join([]string{notificationView, "sync: " + spin + " running"}, "\n"))
	}
	notificationView = strings.TrimSpace(strings.Join([]string{
		notificationView,
		strings.TrimSpace(m.renderNotificationsView()),
		strings.TrimSpace(m.renderProductivityView()),
	}, "\n"))

	return views.RenderApp(views.AppData{
		Header:       fmt.Sprintf("taskd | view: %s | selected: %s", m.CurrentView, m.SelectedTaskID),
		LeftPane:     leftPane,
		RightPane:    rightPane,
		StatusLine:   status,
		Notification: notificationView,
		Footer:       fmt.Sprintf("keys: %s today | %s inbox | %s cal | %s focus | / cmd | %s help | %s quit", m.Keys.Today, m.Keys.Inbox, m.Keys.Calendar, m.Keys.Focus, m.Keys.Help, m.Keys.Quit),
	})
}

func isKnownView(v View) bool {
	switch v {
	case ViewToday, ViewInbox, ViewCalendar, ViewFocus:
		return true
	default:
		return false
	}
}
