package update

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/commands"
	domainmodel "github.com/sandeepkv93/taskd/internal/model"
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

type KeyBinding struct {
	Key    string
	Action string
}

type helpKeyMap struct {
	short []key.Binding
	full  [][]key.Binding
}

func (k helpKeyMap) ShortHelp() []key.Binding  { return k.short }
func (k helpKeyMap) FullHelp() [][]key.Binding { return k.full }

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

func (m Model) handlePaletteKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.Palette.Active = false
		m.Palette.Input = ""
		m.commandInput.SetValue("")
		m.commandInput.Blur()
		m.Status = StatusBar{Text: "command palette closed", IsError: false}
	case "enter":
		m.Palette.Input = m.commandInput.Value()
		m = m.executePaletteCommand()
	default:
		if msg.Type == tea.KeyRunes {
			m.commandInput.SetValue(m.commandInput.Value() + string(msg.Runes))
			m.Palette.Input = m.commandInput.Value()
			return m
		}
		var cmd tea.Cmd
		m.commandInput, cmd = m.commandInput.Update(msg)
		_ = cmd
		m.Palette.Input = m.commandInput.Value()
	}
	return m
}

func (m Model) executePaletteCommand() Model {
	raw := strings.TrimSpace(m.Palette.Input)
	cmd, err := commands.Parse(raw)
	if err != nil {
		m.Status = StatusBar{Text: err.Error(), IsError: true}
		m.Palette.Active = false
		m.Palette.Input = ""
		return m
	}

	res, err := commands.Execute(cmd, commands.Handlers{
		Add: func(a commands.AddArgs) (commands.Result, error) {
			m.CurrentView = ViewInbox
			m.addInboxItem(a.Title)
			return commands.Result{Message: fmt.Sprintf("added inbox task: %s", a.Title)}, nil
		},
		Snooze: func(s commands.SnoozeArgs) (commands.Result, error) {
			applied := 0
			for i := range m.Today.Items {
				if strings.EqualFold(s.Target, "overdue") && m.Today.Items[i].Bucket == TodayBucketOverdue {
					m.Today.Items[i].Bucket = TodayBucketAnytime
					m.Today.Items[i].Notes = strings.TrimSpace(m.Today.Items[i].Notes + " | snoozed " + s.For)
					applied++
				}
			}
			if applied == 0 {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "no matching items for snooze target"}
			}
			return commands.Result{Message: fmt.Sprintf("snoozed %d task(s) for %s", applied, s.For)}, nil
		},
		Show: func(s commands.ShowArgs) (commands.Result, error) {
			if s.Tag != "" {
				m.Filter.Tag = s.Tag
				return commands.Result{Message: fmt.Sprintf("show filter applied: tag=%s", s.Tag)}, nil
			}
			return commands.Result{Message: fmt.Sprintf("show %s", s.Subject)}, nil
		},
		Reschedule: func(r commands.RescheduleArgs) (commands.Result, error) {
			if r.Target != "selected" {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "reschedule currently supports target: selected"}
			}
			applied := 0
			for i := range m.Inbox.Items {
				item := m.Inbox.Items[i]
				if m.Inbox.Selected[item.ID] {
					m.Inbox.Items[i].ScheduledFor = r.When
					applied++
				}
			}
			if applied == 0 {
				return commands.Result{}, &commands.CommandError{Code: commands.ErrCodeInvalidArgument, Message: "no selected inbox items to reschedule"}
			}
			return commands.Result{Message: fmt.Sprintf("rescheduled %d selected item(s) to %s", applied, r.When)}, nil
		},
	})
	if err != nil {
		m.Status = StatusBar{Text: err.Error(), IsError: true}
		m.notify("Command Failed", err.Error(), "error")
	} else {
		m.Status = StatusBar{Text: res.Message, IsError: false}
		m.notify("Command", res.Message, "info")
		m.refreshProductivitySignals()
	}

	m.Palette.Active = false
	m.Palette.Input = ""
	m.commandInput.SetValue("")
	return m
}

func (m Model) renderCommandPalette() string {
	return views.RenderCommandPalette(m.Palette.Active, m.Palette.Input)
}

func (m Model) renderNotificationsView() string {
	if len(m.Notifications) == 0 {
		return ""
	}
	n := m.Notifications[len(m.Notifications)-1]
	return views.RenderNotification(n.Level, n.Body)
}

func (m Model) renderProductivityView() string {
	s := m.Productivity.Signals
	suggestions := make([]views.SuggestionData, 0, len(s.Suggestions))
	if len(s.Suggestions) > 0 {
		for _, sg := range s.Suggestions {
			suggestions = append(suggestions, views.SuggestionData{
				Title:   sg.Title,
				Energy:  sg.Energy,
				Minutes: sg.Minutes,
				Reason:  sg.Reason,
			})
		}
	}
	return views.RenderProductivityPanel(views.ProductivityPanelData{
		TemporalDebtScore: s.TemporalDebtScore,
		TemporalDebtLabel: s.TemporalDebtLabel,
		Suggestions:       suggestions,
	})
}

func (m Model) renderHelpIfVisible() string {
	if !m.HelpVisible {
		return ""
	}
	return m.renderHelpView()
}

func (m Model) renderTodayMetadataPane() string {
	selected, ok := m.currentTodayItem()
	if !ok {
		return "metadata:\n(no selection)"
	}
	return views.RenderTodayMetadataPane(views.TodayMetadataData{
		SelectedID:       selected.ID,
		Priority:         selected.Priority,
		Tags:             selected.Tags,
		NotesEditorView:  m.notesArea.View(),
		MarkdownMetaView: m.metaViewport.View(),
	})
}

func (m Model) renderRecurrenceEditorIfVisible() string {
	return views.RenderRecurrenceEditor(views.RecurrenceEditorData{
		Active:       m.recurrenceEditor.Active,
		RuleType:     m.recurrenceEditor.RuleType,
		IntervalText: m.recurrenceEditor.IntervalText,
		ErrorText:    m.recurrenceEditor.Err,
		Preview:      m.recurrenceEditor.Preview,
	})
}

func (m Model) handleRecurrenceEditorKey(msg tea.KeyMsg) Model {
	switch msg.String() {
	case "esc":
		m.recurrenceEditor.Active = false
		return m
	case "tab":
		// Toggle between known presets to keep editor predictable in first pass.
		switch m.recurrenceEditor.RuleType {
		case "every_weekday":
			m.recurrenceEditor.RuleType = "every_n_days"
		case "every_n_days":
			m.recurrenceEditor.RuleType = "every_n_weeks"
		case "every_n_weeks":
			m.recurrenceEditor.RuleType = "last_day_of_month"
		case "last_day_of_month":
			m.recurrenceEditor.RuleType = "after_completion"
		default:
			m.recurrenceEditor.RuleType = "every_weekday"
		}
	case "enter":
		m.computeRecurrencePreview()
	case "backspace":
		if len(m.recurrenceEditor.IntervalText) > 0 {
			m.recurrenceEditor.IntervalText = m.recurrenceEditor.IntervalText[:len(m.recurrenceEditor.IntervalText)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.recurrenceEditor.IntervalText += string(msg.Runes)
		}
	}
	return m
}

func (m *Model) computeRecurrencePreview() {
	interval := 1
	if v := strings.TrimSpace(m.recurrenceEditor.IntervalText); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			interval = parsed
		}
	}
	rule := domainmodel.RecurrenceRule{
		Type:     domainmodel.RecurrenceType(m.recurrenceEditor.RuleType),
		Interval: interval,
		Anchor:   time.Now().UTC(),
	}
	preview, err := rule.Preview(time.Now().UTC(), nil, 5)
	if err != nil {
		m.recurrenceEditor.Err = err.Error()
		m.recurrenceEditor.Preview = nil
		return
	}
	m.recurrenceEditor.Err = ""
	m.recurrenceEditor.Preview = make([]string, 0, len(preview))
	for _, item := range preview {
		m.recurrenceEditor.Preview = append(m.recurrenceEditor.Preview, item.Format("2006-01-02 15:04"))
	}
}

func (m Model) renderHelpView() string {
	bindings := m.helpBindings()
	var plain []string
	for _, kb := range m.viewBindings() {
		plain = append(plain, fmt.Sprintf("- %s: %s", kb.Key, kb.Action))
	}
	return views.RenderHelpPanel(views.HelpPanelData{
		CurrentView: string(m.CurrentView),
		Bindings:    plain,
		HelpView: m.helpModel.View(helpKeyMap{
			short: bindings,
			full:  [][]key.Binding{bindings},
		}),
	})
}

func (m Model) globalBindings() []KeyBinding {
	return []KeyBinding{
		{Key: m.Keys.Today, Action: "switch to Today"},
		{Key: m.Keys.Inbox, Action: "switch to Inbox"},
		{Key: m.Keys.Calendar, Action: "switch to Calendar"},
		{Key: m.Keys.Focus, Action: "switch to Focus"},
		{Key: "/", Action: "open command palette"},
		{Key: "D", Action: "cycle density"},
		{Key: m.Keys.Help, Action: "toggle help panel"},
		{Key: m.Keys.Quit, Action: "quit app"},
	}
}

func (m Model) viewBindings() []KeyBinding {
	switch m.CurrentView {
	case ViewInbox:
		return []KeyBinding{
			{Key: "enter", Action: "capture inbox item"},
			{Key: "j/k", Action: "move cursor"},
			{Key: "space", Action: "toggle select"},
			{Key: "x/u", Action: "select all / clear selection"},
			{Key: "s/g", Action: "bulk schedule / bulk tag"},
		}
	case ViewToday:
		return []KeyBinding{
			{Key: "j/k", Action: "move selection"},
			{Key: "z", Action: "collapse/expand selected section"},
		}
	case ViewCalendar:
		return []KeyBinding{
			{Key: "d/w/m", Action: "day/week/month mode"},
			{Key: "h/l", Action: "previous/next period"},
			{Key: "j/k", Action: "move agenda cursor"},
		}
	case ViewFocus:
		return []KeyBinding{
			{Key: "space", Action: "start/pause timer"},
			{Key: "r", Action: "reset timer"},
			{Key: "n", Action: "next focus phase"},
		}
	default:
		return []KeyBinding{{Key: "-", Action: "no contextual bindings"}}
	}
}

func (m Model) helpBindings() []key.Binding {
	out := make([]key.Binding, 0, len(m.globalBindings())+len(m.viewBindings()))
	for _, kb := range m.globalBindings() {
		out = append(out, key.NewBinding(key.WithKeys(kb.Key), key.WithHelp(kb.Key, kb.Action)))
	}
	for _, kb := range m.viewBindings() {
		out = append(out, key.NewBinding(key.WithKeys(kb.Key), key.WithHelp(kb.Key, kb.Action)))
	}
	return out
}

func (m *Model) notify(title, body, level string) {
	if strings.TrimSpace(body) == "" {
		return
	}
	n := Notification{
		Title: title,
		Body:  body,
		Level: level,
		At:    time.Now().UTC(),
	}
	m.Notifications = append(m.Notifications, n)
	if len(m.Notifications) > 40 {
		m.Notifications = m.Notifications[len(m.Notifications)-40:]
	}
	if m.DesktopEnabled && m.notifier != nil {
		_ = m.notifier.Send(n)
	}
}

func (m *Model) refreshProductivitySignals() {
	score := m.computeTemporalDebtScore()
	m.Productivity.Signals.TemporalDebtScore = score
	switch {
	case score >= 7:
		m.Productivity.Signals.TemporalDebtLabel = "high"
	case score >= 4:
		m.Productivity.Signals.TemporalDebtLabel = "medium"
	default:
		m.Productivity.Signals.TemporalDebtLabel = "low"
	}
	m.Productivity.Signals.Suggestions = m.computeEnergySuggestions(m.Productivity.AvailableMinutes, 3)
}

func (m Model) computeTemporalDebtScore() int {
	score := 0
	for _, item := range m.Today.Items {
		if item.Bucket == TodayBucketOverdue {
			score += 2
		}
		notes := strings.ToLower(item.Notes)
		if strings.Contains(notes, "snoozed") {
			score++
		}
		if item.Priority == "Critical" && item.Bucket == TodayBucketOverdue {
			score++
		}
	}
	if score > 10 {
		return 10
	}
	return score
}

func (m Model) computeEnergySuggestions(availableMinutes int, limit int) []Suggestion {
	if availableMinutes <= 0 || limit <= 0 {
		return nil
	}
	out := make([]Suggestion, 0, limit)
	for _, item := range m.Today.Items {
		energy := inferEnergyFromTodayItem(item)
		minutes := estimateMinutesForEnergy(energy)
		if minutes > availableMinutes {
			continue
		}
		reason := fmt.Sprintf("fits %d-minute window", availableMinutes)
		if item.Bucket == TodayBucketOverdue {
			reason = "overdue and still feasible now"
		}
		out = append(out, Suggestion{
			TaskID:  item.ID,
			Title:   item.Title,
			Reason:  reason,
			Energy:  energy,
			Minutes: minutes,
		})
		if len(out) >= limit {
			break
		}
	}
	return out
}

func inferEnergyFromTodayItem(item TodayItem) string {
	combined := strings.ToLower(item.Title + " " + item.Notes + " " + strings.Join(item.Tags, " "))
	switch {
	case strings.Contains(combined, "call"), strings.Contains(combined, "meeting"), strings.Contains(combined, "standup"):
		return "Social"
	case strings.Contains(combined, "review"), strings.Contains(combined, "docs"), strings.Contains(combined, "write"):
		return "Light"
	case strings.Contains(combined, "tax"), strings.Contains(combined, "budget"), strings.Contains(combined, "email"):
		return "Low"
	default:
		return "Deep"
	}
}

func estimateMinutesForEnergy(energy string) int {
	switch strings.ToLower(energy) {
	case "deep":
		return 90
	case "social":
		return 45
	case "light":
		return 30
	default:
		return 20
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

func (m *Model) applyReminderBehavior(ev scheduler.ReminderEvent, now time.Time) {
	t := strings.ToLower(strings.TrimSpace(ev.Type))
	switch t {
	case "hard":
		m.Status = StatusBar{Text: fmt.Sprintf("HARD reminder: %s", ev.ID), IsError: true}
	case "soft":
		m.Status = StatusBar{Text: fmt.Sprintf("soft reminder: %s", ev.ID), IsError: false}
		if !m.SoftFollowedUp[ev.ID] {
			m.SoftFollowedUp[ev.ID] = true
			m.rescheduleReminder(ev, now.Add(10*time.Minute))
		}
	case "nagging":
		m.Status = StatusBar{Text: fmt.Sprintf("nagging reminder: %s", ev.ID), IsError: false}
		if !m.ReminderAck[ev.ID] && !m.isTaskCompleted(ev.TaskID) {
			m.rescheduleReminder(ev, now.Add(2*time.Minute))
		}
	case "contextual":
		if inContextualWindowForRule(now, ev.RepeatRule) {
			m.Status = StatusBar{Text: fmt.Sprintf("contextual reminder: %s", ev.ID), IsError: false}
		} else {
			next := nextContextualWindowStartForRule(now, ev.RepeatRule)
			m.Status = StatusBar{Text: fmt.Sprintf("contextual deferred: %s -> %s", ev.ID, next.Format("15:04")), IsError: false}
			m.rescheduleReminder(ev, next)
		}
	default:
		m.Status = StatusBar{Text: fmt.Sprintf("reminder fired: %s", ev.ID), IsError: false}
	}
}

func (m *Model) rescheduleReminder(ev scheduler.ReminderEvent, next time.Time) {
	if m.Scheduler == nil {
		return
	}
	nextEv := ev
	nextEv.TriggerAt = next
	if err := m.Scheduler.Schedule(nextEv); err != nil {
		m.Status = StatusBar{Text: fmt.Sprintf("reminder reschedule failed: %v", err), IsError: true}
	}
}

func inContextualWindowForRule(now time.Time, rule string) bool {
	cfg := parseContextualRule(rule)
	if !cfg.allowsWeekday(now.Weekday()) {
		return false
	}
	return cfg.inWindow(now)
}

func nextContextualWindowStartForRule(now time.Time, rule string) time.Time {
	cfg := parseContextualRule(rule)
	loc := now.Location()
	for i := 0; i < 30; i++ {
		day := now.AddDate(0, 0, i)
		if !cfg.allowsWeekday(day.Weekday()) {
			continue
		}
		y, mo, d := day.Date()
		for _, w := range cfg.windows {
			candidate := time.Date(y, mo, d, w.StartHour, 0, 0, 0, loc)
			if candidate.After(now) {
				return candidate
			}
		}
	}
	return now.Add(24 * time.Hour)
}

type contextualWindow struct {
	StartHour int
	EndHour   int
}

type contextualRule struct {
	windows []contextualWindow
	days    map[time.Weekday]bool
}

func parseContextualRule(raw string) contextualRule {
	rule := contextualRule{
		windows: []contextualWindow{{StartHour: 18, EndHour: 22}}, // default: evening
	}
	normalized := strings.ToLower(strings.TrimSpace(raw))
	if normalized == "" {
		return rule
	}

	parts := strings.FieldsFunc(normalized, func(r rune) bool {
		return r == ';' || r == '|'
	})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "window=") {
			val := strings.TrimSpace(strings.TrimPrefix(part, "window="))
			if parsed := parseContextualWindows(val); len(parsed) > 0 {
				rule.windows = parsed
			}
			continue
		}
		if strings.HasPrefix(part, "days=") {
			val := strings.TrimSpace(strings.TrimPrefix(part, "days="))
			if parsed := parseContextualDays(val); len(parsed) > 0 {
				rule.days = parsed
			}
			continue
		}
	}

	// Backward-compatible keyword parsing for simpler rules.
	if strings.Contains(normalized, "morning") {
		rule.windows = []contextualWindow{{StartHour: 8, EndHour: 12}}
	}
	if strings.Contains(normalized, "afternoon") {
		rule.windows = []contextualWindow{{StartHour: 12, EndHour: 17}}
	}
	if strings.Contains(normalized, "evening") {
		rule.windows = []contextualWindow{{StartHour: 18, EndHour: 22}}
	}
	if strings.Contains(normalized, "weekend") {
		rule.days = map[time.Weekday]bool{time.Saturday: true, time.Sunday: true}
	}
	if strings.Contains(normalized, "weekday") {
		rule.days = map[time.Weekday]bool{
			time.Monday: true, time.Tuesday: true, time.Wednesday: true, time.Thursday: true, time.Friday: true,
		}
	}

	return rule
}

func parseContextualWindows(raw string) []contextualWindow {
	out := make([]contextualWindow, 0, 2)
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "morning":
			out = append(out, contextualWindow{StartHour: 8, EndHour: 12})
		case "afternoon":
			out = append(out, contextualWindow{StartHour: 12, EndHour: 17})
		case "evening":
			out = append(out, contextualWindow{StartHour: 18, EndHour: 22})
		}
	}
	return out
}

func parseContextualDays(raw string) map[time.Weekday]bool {
	parsed := make(map[time.Weekday]bool)
	for _, token := range strings.Split(raw, ",") {
		switch strings.TrimSpace(token) {
		case "weekdays", "weekday":
			parsed[time.Monday] = true
			parsed[time.Tuesday] = true
			parsed[time.Wednesday] = true
			parsed[time.Thursday] = true
			parsed[time.Friday] = true
		case "weekends", "weekend":
			parsed[time.Saturday] = true
			parsed[time.Sunday] = true
		case "mon", "monday":
			parsed[time.Monday] = true
		case "tue", "tuesday":
			parsed[time.Tuesday] = true
		case "wed", "wednesday":
			parsed[time.Wednesday] = true
		case "thu", "thursday":
			parsed[time.Thursday] = true
		case "fri", "friday":
			parsed[time.Friday] = true
		case "sat", "saturday":
			parsed[time.Saturday] = true
		case "sun", "sunday":
			parsed[time.Sunday] = true
		}
	}
	return parsed
}

func (r contextualRule) allowsWeekday(d time.Weekday) bool {
	if len(r.days) == 0 {
		return true
	}
	return r.days[d]
}

func (r contextualRule) inWindow(now time.Time) bool {
	h := now.Hour()
	for _, w := range r.windows {
		if h >= w.StartHour && h < w.EndHour {
			return true
		}
	}
	return false
}

func (m Model) isTaskCompleted(taskID string) bool {
	if strings.TrimSpace(taskID) == "" {
		return false
	}
	return m.CompletedTasks[taskID]
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

type completionState struct {
	CompletedTaskIDs []string `json:"completed_task_ids"`
}

func (m *Model) persistCompletedTaskState() error {
	if strings.TrimSpace(m.stateFilePath) == "" {
		return nil
	}
	dir := filepath.Dir(m.stateFilePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	ids := make([]string, 0, len(m.CompletedTasks))
	for id, done := range m.CompletedTasks {
		if done && strings.TrimSpace(id) != "" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	payload, err := json.MarshalIndent(completionState{CompletedTaskIDs: ids}, "", "  ")
	if err != nil {
		return err
	}
	tmp := m.stateFilePath + ".tmp"
	if err := os.WriteFile(tmp, append(payload, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, m.stateFilePath)
}

func loadCompletedTaskState(path string) (map[string]bool, error) {
	out := make(map[string]bool)
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return out, nil
	}
	raw, err := os.ReadFile(trimmed)
	if err != nil {
		if os.IsNotExist(err) {
			return out, nil
		}
		return nil, err
	}
	if strings.TrimSpace(string(raw)) == "" {
		return out, nil
	}
	var state completionState
	if err := json.Unmarshal(raw, &state); err != nil {
		return nil, err
	}
	for _, id := range state.CompletedTaskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		out[id] = true
	}
	return out, nil
}
