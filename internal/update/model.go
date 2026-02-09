package update

import (
	"fmt"
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
	"github.com/sandeepkv93/taskd/internal/scheduler"
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
