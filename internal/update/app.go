package update

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/scheduler"
	"github.com/sandeepkv93/taskd/internal/views"
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
	Today          TodayState
	Calendar       CalendarState
	Focus          FocusState
	Scheduler      *scheduler.Engine
	ReminderLog    []scheduler.ReminderEvent
	ReminderAck    map[string]bool
	SoftFollowedUp map[string]bool
	CompletedTasks map[string]bool
	Palette        CommandPaletteState
	HelpVisible    bool
	Notifications  []Notification
	DesktopEnabled bool
	notifier       DesktopNotifier
	Productivity   ProductivityState
	Status         StatusBar
	Keys           GlobalKeyMap
	Quitting       bool
	LastError      error
	// Bubble components used for rich TUI controls
	inboxList     list.Model
	todayList     list.Model
	calendarTable table.Model
	quickAddInput textinput.Model
	commandInput  textinput.Model
	notesArea     textarea.Model
	focusProgress progress.Model
	syncSpinner   spinner.Model
	helpModel     help.Model
	metaViewport  viewport.Model
	spinnerActive bool
	stateFilePath string
	// Recurrence editor (first-pass UI)
	recurrenceEditor RecurrenceEditorState
	todayCollapsed   map[TodayBucket]bool
	uiDensity        int
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

type TodayBucket string

const (
	TodayBucketScheduled TodayBucket = "Scheduled"
	TodayBucketAnytime   TodayBucket = "Anytime"
	TodayBucketOverdue   TodayBucket = "Overdue"
)

type TodayItem struct {
	ID          string
	Title       string
	Bucket      TodayBucket
	ScheduledAt string
	DueAt       string
	Priority    string
	Tags        []string
	Notes       string
}

type TodayState struct {
	Items  []TodayItem
	Cursor int
}

type CalendarMode string

const (
	CalendarModeDay   CalendarMode = "day"
	CalendarModeWeek  CalendarMode = "week"
	CalendarModeMonth CalendarMode = "month"
)

type AgendaItem struct {
	ID    string
	Title string
	Date  string
	Time  string
	Kind  string
}

type CalendarState struct {
	Mode      CalendarMode
	FocusDate time.Time
	Items     []AgendaItem
	Cursor    int
}

type FocusPhase string

const (
	FocusPhaseWork  FocusPhase = "work"
	FocusPhaseBreak FocusPhase = "break"
)

type FocusState struct {
	TaskID             string
	TaskTitle          string
	WorkDurationSec    int
	BreakDurationSec   int
	RemainingSec       int
	Running            bool
	Phase              FocusPhase
	CompletedPomodoros int
}

type CommandPaletteState struct {
	Active bool
	Input  string
}

type RecurrenceEditorState struct {
	Active       bool
	RuleType     string
	IntervalText string
	Preview      []string
	Err          string
}

type listItem struct {
	title       string
	description string
}

func (i listItem) FilterValue() string { return i.title + " " + i.description }
func (i listItem) Title() string       { return i.title }
func (i listItem) Description() string { return i.description }

type Notification struct {
	Title string
	Body  string
	Level string
	At    time.Time
}

type ProductivityState struct {
	AvailableMinutes int
	Signals          ProductivitySignals
}

type ProductivitySignals struct {
	TemporalDebtScore int
	TemporalDebtLabel string
	Suggestions       []Suggestion
}

type Suggestion struct {
	TaskID  string
	Title   string
	Reason  string
	Energy  string
	Minutes int
}

type DesktopNotifier interface {
	Send(Notification) error
}

type NoopDesktopNotifier struct{}

func (NoopDesktopNotifier) Send(Notification) error { return nil }

type ExecDesktopNotifier struct{}

func (ExecDesktopNotifier) Send(n Notification) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("notify-send", n.Title, n.Body).Run()
	case "darwin":
		script := fmt.Sprintf(`display notification "%s" with title "%s"`, escapeAppleScript(n.Body), escapeAppleScript(n.Title))
		return exec.Command("osascript", "-e", script).Run()
	default:
		return nil
	}
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

type SetTodayItemsMsg struct {
	Items []TodayItem
}

type SetCalendarItemsMsg struct {
	Items []AgendaItem
}

type FocusTickMsg struct{}

type ReminderDueMsg struct {
	Event scheduler.ReminderEvent
}

type AcknowledgeReminderMsg struct {
	ID string
}

func NewModel() Model {
	m := Model{
		CurrentView: ViewToday,
		Sort:        SortCreatedDesc,
		Inbox: InboxState{
			Selected: make(map[string]bool),
			NextID:   1,
		},
		Today: TodayState{
			Items: []TodayItem{
				{
					ID:          "today-1",
					Title:       "Daily standup",
					Bucket:      TodayBucketScheduled,
					ScheduledAt: "09:30",
					Priority:    "High",
					Tags:        []string{"team"},
					Notes:       "Share blockers and plan.",
				},
				{
					ID:       "today-2",
					Title:    "Review pull request",
					Bucket:   TodayBucketAnytime,
					Priority: "Medium",
					Tags:     []string{"code"},
					Notes:    "Check tests and architecture changes.",
				},
				{
					ID:       "today-3",
					Title:    "Submit tax docs",
					Bucket:   TodayBucketOverdue,
					DueAt:    "Yesterday",
					Priority: "Critical",
					Tags:     []string{"finance"},
					Notes:    "Overdue since yesterday evening.",
				},
			},
		},
		Calendar: CalendarState{
			Mode:      CalendarModeWeek,
			FocusDate: time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC),
			Items: []AgendaItem{
				{ID: "ag-1", Title: "Design review", Date: "2026-02-09", Time: "11:00", Kind: "event"},
				{ID: "ag-2", Title: "Write migration docs", Date: "2026-02-09", Time: "14:00", Kind: "task"},
				{ID: "ag-3", Title: "Gym", Date: "2026-02-10", Time: "18:30", Kind: "event"},
			},
		},
		Focus: FocusState{
			WorkDurationSec:  25 * 60,
			BreakDurationSec: 5 * 60,
			RemainingSec:     25 * 60,
			Phase:            FocusPhaseWork,
		},
		ReminderAck:    make(map[string]bool),
		SoftFollowedUp: make(map[string]bool),
		CompletedTasks: make(map[string]bool),
		DesktopEnabled: false,
		notifier:       NoopDesktopNotifier{},
		Productivity: ProductivityState{
			AvailableMinutes: 60,
		},
		Keys: GlobalKeyMap{
			Today:    "1",
			Inbox:    "2",
			Calendar: "3",
			Focus:    "4",
			Help:     "?",
			Quit:     "q",
		},
		recurrenceEditor: RecurrenceEditorState{
			RuleType:     "every_n_days",
			IntervalText: "1",
		},
		todayCollapsed: map[TodayBucket]bool{
			TodayBucketScheduled: false,
			TodayBucketAnytime:   false,
			TodayBucketOverdue:   false,
		},
		uiDensity: 1,
	}
	m.initBubbleComponents()
	m.syncBubbleData()
	m.refreshProductivitySignals()
	return m
}

func NewModelWithScheduler(engine *scheduler.Engine) Model {
	m := NewModel()
	m.Scheduler = engine
	return m
}

func NewModelWithRuntime(engine *scheduler.Engine, desktopEnabled bool, notifier DesktopNotifier) Model {
	cfg := DefaultRuntimeConfig()
	cfg.DesktopNotifications = desktopEnabled
	return NewModelWithConfig(engine, notifier, cfg)
}

func NewModelWithConfig(engine *scheduler.Engine, notifier DesktopNotifier, cfg RuntimeConfig) Model {
	m := NewModel()
	m.Scheduler = engine
	m.DesktopEnabled = cfg.DesktopNotifications
	m.stateFilePath = strings.TrimSpace(cfg.CompletionStatePath)
	if notifier != nil {
		m.notifier = notifier
	}
	if cfg.FocusWorkMinutes > 0 {
		m.Focus.WorkDurationSec = cfg.FocusWorkMinutes * 60
	}
	if cfg.FocusBreakMinutes > 0 {
		m.Focus.BreakDurationSec = cfg.FocusBreakMinutes * 60
	}
	m.Focus.RemainingSec = m.Focus.WorkDurationSec
	if cfg.ProductivityAvailableMins > 0 {
		m.Productivity.AvailableMinutes = cfg.ProductivityAvailableMins
	}
	if m.stateFilePath != "" {
		if completed, err := loadCompletedTaskState(m.stateFilePath); err == nil {
			m.CompletedTasks = completed
		}
	}
	m.refreshProductivitySignals()
	return m
}

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

		switch typed.String() {
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
		Footer:       fmt.Sprintf("keys: %s today | %s inbox | %s calendar | %s focus | %s help | %s quit", m.Keys.Today, m.Keys.Inbox, m.Keys.Calendar, m.Keys.Focus, m.Keys.Help, m.Keys.Quit),
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

func (m *Model) initBubbleComponents() {
	m.inboxList = list.New([]list.Item{}, list.NewDefaultDelegate(), 56, 12)
	m.inboxList.Title = "Inbox (list)"
	m.inboxList.SetShowHelp(false)
	m.inboxList.SetFilteringEnabled(false)

	m.todayList = list.New([]list.Item{}, list.NewDefaultDelegate(), 56, 12)
	m.todayList.Title = "Today (list)"
	m.todayList.SetShowHelp(false)
	m.todayList.SetFilteringEnabled(false)

	cols := []table.Column{
		{Title: "Date", Width: 12},
		{Title: "Time", Width: 7},
		{Title: "Kind", Width: 8},
		{Title: "Title", Width: 24},
	}
	m.calendarTable = table.New(table.WithColumns(cols), table.WithRows([]table.Row{}), table.WithFocused(true), table.WithHeight(10))

	m.quickAddInput = textinput.New()
	m.quickAddInput.Prompt = "add> "
	m.quickAddInput.CharLimit = 256
	m.quickAddInput.Width = 42

	m.commandInput = textinput.New()
	m.commandInput.Prompt = "/"
	m.commandInput.CharLimit = 256
	m.commandInput.Width = 48

	m.notesArea = textarea.New()
	m.notesArea.SetWidth(54)
	m.notesArea.SetHeight(8)
	m.notesArea.ShowLineNumbers = false
	m.notesArea.Placeholder = "Task notes (markdown)"

	m.focusProgress = progress.New(progress.WithDefaultGradient())

	m.syncSpinner = spinner.New()
	m.syncSpinner.Spinner = spinner.Dot

	m.helpModel = help.New()
	m.metaViewport = viewport.New(54, 12)
}

func (m *Model) syncBubbleData() {
	listWidth, listHeight, tableHeight, notesHeight, viewportHeight := densityDimensions(m.uiDensity)
	m.inboxList.SetSize(listWidth, listHeight)
	m.todayList.SetSize(listWidth, listHeight)
	m.calendarTable.SetHeight(tableHeight)
	m.notesArea.SetHeight(notesHeight)
	m.metaViewport.Height = viewportHeight

	inboxItems := make([]list.Item, 0, len(m.Inbox.Items))
	for _, item := range m.Inbox.Items {
		desc := item.ScheduledFor
		if desc == "" {
			desc = strings.Join(item.Tags, ",")
		}
		inboxItems = append(inboxItems, listItem{title: item.Title, description: desc})
	}
	m.inboxList.SetItems(inboxItems)
	if len(inboxItems) > 0 {
		m.inboxList.Select(m.Inbox.Cursor)
	}

	todayItems := make([]list.Item, 0, len(m.Today.Items))
	for _, item := range m.Today.Items {
		desc := fmt.Sprintf("%s | %s", item.Bucket, item.Priority)
		todayItems = append(todayItems, listItem{title: item.Title, description: desc})
	}
	m.todayList.SetItems(todayItems)
	if len(todayItems) > 0 {
		m.todayList.Select(m.Today.Cursor)
	}

	rows := make([]table.Row, 0, len(m.Calendar.Items))
	for _, item := range m.Calendar.Items {
		rows = append(rows, table.Row{item.Date, item.Time, strings.ToUpper(item.Kind), item.Title})
	}
	m.calendarTable.SetRows(rows)
	if len(rows) > 0 && m.Calendar.Cursor < len(rows) {
		m.calendarTable.SetCursor(m.Calendar.Cursor)
	}

	m.quickAddInput.SetValue(m.Inbox.Input)
	m.commandInput.SetValue(m.Palette.Input)
	if m.CurrentView == ViewInbox {
		m.quickAddInput.Focus()
	}
	if m.Palette.Active {
		m.commandInput.Focus()
	}

	if sel, ok := m.currentTodayItem(); ok {
		md := sel.Notes
		if strings.TrimSpace(md) == "" {
			md = "_No notes_"
		}
		m.notesArea.SetValue(md)
		m.metaViewport.SetContent(views.RenderMarkdown(md))
	}

	total := m.currentFocusTotal()
	pct := 0.0
	if total > 0 {
		pct = float64(total-m.Focus.RemainingSec) / float64(total)
	}
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	_ = m.focusProgress.SetPercent(pct)
}

func densityDimensions(level int) (listWidth int, listHeight int, tableHeight int, notesHeight int, viewportHeight int) {
	switch level {
	case 2:
		return 60, 14, 12, 10, 14
	case 3:
		return 64, 16, 14, 12, 16
	default:
		return 56, 12, 10, 8, 12
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

func (m *Model) ensureTodayState() {
	if m.todayCollapsed == nil {
		m.todayCollapsed = map[TodayBucket]bool{
			TodayBucketScheduled: false,
			TodayBucketAnytime:   false,
			TodayBucketOverdue:   false,
		}
	}
	if m.Today.Cursor < 0 {
		m.Today.Cursor = 0
	}
	if m.Today.Cursor >= len(m.Today.Items) && len(m.Today.Items) > 0 {
		m.Today.Cursor = len(m.Today.Items) - 1
	}
	if len(m.Today.Items) > 0 && m.SelectedTaskID == "" {
		m.syncSelectedTaskToTodayCursor()
	}
}

func (m *Model) toggleTodaySectionCollapse() {
	selected, ok := m.currentTodayItem()
	if !ok {
		return
	}
	bucket := selected.Bucket
	m.todayCollapsed[bucket] = !m.todayCollapsed[bucket]
	state := "expanded"
	if m.todayCollapsed[bucket] {
		state = "collapsed"
	}
	m.Status = StatusBar{
		Text:    fmt.Sprintf("%s section %s", strings.ToLower(string(bucket)), state),
		IsError: false,
	}
}

func (m *Model) cycleDensity() {
	m.uiDensity++
	if m.uiDensity > 3 {
		m.uiDensity = 1
	}
	m.Status = StatusBar{
		Text:    fmt.Sprintf("density level: %d", m.uiDensity),
		IsError: false,
	}
}

func (m Model) renderInboxView() string {
	return views.RenderInboxPanel(views.InboxPanelData{
		QuickAddView: m.quickAddInput.View(),
		ListView:     m.inboxList.View(),
	})
}

func (m Model) renderTodayView() string {
	items := make([]views.TodayItemData, 0, len(m.Today.Items))
	for _, item := range m.Today.Items {
		items = append(items, views.TodayItemData{
			ID:          item.ID,
			Title:       item.Title,
			Bucket:      string(item.Bucket),
			ScheduledAt: item.ScheduledAt,
			DueAt:       item.DueAt,
			Priority:    item.Priority,
		})
	}
	return views.RenderTodayPanel(views.TodayPanelData{
		ListView:   m.todayList.View(),
		Items:      items,
		SelectedID: m.SelectedTaskID,
		Collapsed: map[string]bool{
			string(TodayBucketScheduled): m.todayCollapsed[TodayBucketScheduled],
			string(TodayBucketAnytime):   m.todayCollapsed[TodayBucketAnytime],
			string(TodayBucketOverdue):   m.todayCollapsed[TodayBucketOverdue],
		},
	})
}

func (m Model) renderCalendarView() string {
	items := make([]views.CalendarAgendaItemData, 0, len(m.Calendar.Items))
	for _, item := range m.Calendar.Items {
		items = append(items, views.CalendarAgendaItemData{
			ID:    item.ID,
			Title: item.Title,
			Date:  item.Date,
			Time:  item.Time,
			Kind:  item.Kind,
		})
	}
	var selected *views.CalendarAgendaItemData
	if s, ok := m.currentAgendaItem(); ok {
		selected = &views.CalendarAgendaItemData{
			ID:    s.ID,
			Title: s.Title,
			Date:  s.Date,
			Time:  s.Time,
			Kind:  s.Kind,
		}
	}
	return views.RenderCalendarPanel(views.CalendarPanelData{
		Mode:      string(m.Calendar.Mode),
		FocusDate: m.Calendar.FocusDate.Format("2006-01-02"),
		TableView: m.calendarTable.View(),
		Items:     items,
		Selected:  selected,
	})
}

func (m Model) renderFocusView() string {
	total := m.currentFocusTotal()
	progress := 0.0
	if total > 0 {
		progress = float64(total-m.Focus.RemainingSec) / float64(total)
	}

	return views.RenderFocusPanel(views.FocusPanelData{
		TaskTitle:          m.Focus.TaskTitle,
		Phase:              string(m.Focus.Phase),
		Timer:              formatDuration(m.Focus.RemainingSec),
		ProgressView:       m.focusProgress.ViewAs(progress),
		ProgressPct:        int(progress * 100),
		CompletedPomodoros: m.Focus.CompletedPomodoros,
		ShowEndPrompt:      m.Focus.RemainingSec == 0,
	})
}

func waitForReminderCmd(ch <-chan scheduler.ReminderEvent) tea.Cmd {
	if ch == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return ReminderDueMsg{Event: ev}
	}
}

func levelFromError(isErr bool) string {
	if isErr {
		return "error"
	}
	return "info"
}

func escapeAppleScript(s string) string {
	return strings.ReplaceAll(s, `"`, `\"`)
}

func DesktopNotificationsEnabledFromEnv() bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv("TASKD_DESKTOP_NOTIFICATIONS")))
	return v == "1" || v == "true" || v == "yes"
}

func formatDuration(totalSec int) string {
	if totalSec < 0 {
		totalSec = 0
	}
	min := totalSec / 60
	sec := totalSec % 60
	return fmt.Sprintf("%02d:%02d", min, sec)
}

func progressBar(progress float64, width int) string {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("#", filled) + strings.Repeat("-", width-filled) + "]"
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
