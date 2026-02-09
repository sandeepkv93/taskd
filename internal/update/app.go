package update

import (
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/commands"
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
	Palette        CommandPaletteState
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
	return Model{
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

func NewModelWithScheduler(engine *scheduler.Engine) Model {
	m := NewModel()
	m.Scheduler = engine
	return m
}

func (m Model) Init() tea.Cmd {
	if m.Scheduler != nil {
		return waitForReminderCmd(m.Scheduler.C())
	}
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch typed := msg.(type) {
	case tea.KeyMsg:
		m.ensureInboxState()
		m.ensureTodayState()
		m.ensureCalendarState()

		if m.Palette.Active {
			next := m.handlePaletteKey(typed)
			return next, nil
		}

		switch typed.String() {
		case "/":
			m.Palette.Active = true
			m.Palette.Input = ""
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
	case SetTodayItemsMsg:
		m.Today.Items = typed.Items
		m.Today.Cursor = 0
		m.syncSelectedTaskToTodayCursor()
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
			status = fmt.Sprintf("\nstatus: error: %s", m.Status.Text)
		} else {
			status = fmt.Sprintf("\nstatus: %s", m.Status.Text)
		}
	}
	inboxView := ""
	if m.CurrentView == ViewInbox {
		inboxView = m.renderInboxView()
	}
	todayView := ""
	if m.CurrentView == ViewToday {
		todayView = m.renderTodayView()
	}
	calendarView := ""
	if m.CurrentView == ViewCalendar {
		calendarView = m.renderCalendarView()
	}
	focusView := ""
	if m.CurrentView == ViewFocus {
		focusView = m.renderFocusView()
	}
	reminderView := ""
	if len(m.ReminderLog) > 0 {
		last := m.ReminderLog[len(m.ReminderLog)-1]
		reminderView = fmt.Sprintf("\nlast-reminder: %s @ %s", last.ID, last.TriggerAt.Format("15:04:05"))
	}
	paletteView := ""
	if m.Palette.Active {
		paletteView = m.renderCommandPalette()
	}
	return fmt.Sprintf(
		"taskd | view: %s | selected: %s\nkeys: [%s]today [%s]inbox [%s]calendar [%s]focus [%s]quit%s%s%s%s%s%s%s",
		m.CurrentView,
		m.SelectedTaskID,
		m.Keys.Today,
		m.Keys.Inbox,
		m.Keys.Calendar,
		m.Keys.Focus,
		m.Keys.Quit,
		status,
		inboxView,
		todayView,
		calendarView,
		focusView,
		reminderView,
		paletteView,
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

func (m *Model) ensureTodayState() {
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

func (m Model) handleFocusKey(msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case " ":
		if m.Focus.Running {
			m.Focus.Running = false
			m.Status = StatusBar{Text: "focus paused", IsError: false}
			return m, nil
		}
		if m.Focus.RemainingSec <= 0 {
			m.Focus.RemainingSec = m.currentFocusTotal()
		}
		m.Focus.Running = true
		m.Status = StatusBar{Text: "focus running", IsError: false}
		return m, focusTickCmd()
	case "r":
		m.Focus.Running = false
		m.Focus.RemainingSec = m.currentFocusTotal()
		m.Status = StatusBar{Text: "focus reset", IsError: false}
		return m, nil
	case "n":
		m.completeFocusPhase()
		return m, nil
	}
	return m, nil
}

func (m Model) onFocusTick() (tea.Model, tea.Cmd) {
	if !m.Focus.Running {
		return m, nil
	}
	if m.Focus.RemainingSec > 0 {
		m.Focus.RemainingSec--
	}
	if m.Focus.RemainingSec == 0 {
		m.Focus.Running = false
		if m.Focus.Phase == FocusPhaseWork {
			m.Status = StatusBar{Text: "work session complete; press n to start break", IsError: false}
		} else {
			m.Status = StatusBar{Text: "break complete; press n for next focus block", IsError: false}
		}
		return m, nil
	}
	return m, focusTickCmd()
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

func (m Model) renderTodayView() string {
	scheduled := make([]TodayItem, 0)
	anytime := make([]TodayItem, 0)
	overdue := make([]TodayItem, 0)
	for _, item := range m.Today.Items {
		switch item.Bucket {
		case TodayBucketScheduled:
			scheduled = append(scheduled, item)
		case TodayBucketOverdue:
			overdue = append(overdue, item)
		default:
			anytime = append(anytime, item)
		}
	}

	var b strings.Builder
	b.WriteString("\ntoday:\n")
	b.WriteString("actions: [j/k]move [1]today [2]inbox [3]calendar [4]focus\n")
	renderTodaySection(&b, "Scheduled", scheduled, m.Today)
	renderTodaySection(&b, "Anytime", anytime, m.Today)
	renderTodaySection(&b, "Overdue", overdue, m.Today)

	if selected, ok := m.currentTodayItem(); ok {
		b.WriteString("\nmetadata:\n")
		b.WriteString(fmt.Sprintf("id: %s\n", selected.ID))
		b.WriteString(fmt.Sprintf("priority: %s\n", selected.Priority))
		if len(selected.Tags) > 0 {
			b.WriteString(fmt.Sprintf("tags: %s\n", strings.Join(selected.Tags, ",")))
		} else {
			b.WriteString("tags: -\n")
		}
		if selected.ScheduledAt != "" {
			b.WriteString(fmt.Sprintf("scheduled: %s\n", selected.ScheduledAt))
		}
		if selected.DueAt != "" {
			b.WriteString(fmt.Sprintf("due: %s\n", selected.DueAt))
		}
		if selected.Notes != "" {
			b.WriteString(fmt.Sprintf("notes: %s\n", selected.Notes))
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func (m Model) renderCalendarView() string {
	var b strings.Builder
	b.WriteString("\ncalendar:\n")
	b.WriteString(fmt.Sprintf("mode: %s | focus: %s\n", m.Calendar.Mode, m.Calendar.FocusDate.Format("2006-01-02")))
	b.WriteString("actions: [d]day [w]week [m]month [h/l]period [j/k]agenda\n")

	grouped := make(map[string][]AgendaItem)
	keys := make([]string, 0)
	for _, item := range m.Calendar.Items {
		if _, ok := grouped[item.Date]; !ok {
			keys = append(keys, item.Date)
		}
		grouped[item.Date] = append(grouped[item.Date], item)
	}
	sort.Strings(keys)

	if len(keys) == 0 {
		b.WriteString("(agenda empty)")
		return b.String()
	}

	for _, day := range keys {
		b.WriteString(fmt.Sprintf("\n%s:\n", day))
		items := grouped[day]
		sort.SliceStable(items, func(i, j int) bool { return items[i].Time < items[j].Time })
		for _, item := range items {
			cursor := " "
			if calendarCursorItem(m.Calendar, item.ID) {
				cursor = ">"
			}
			b.WriteString(fmt.Sprintf("%s [%s] %s %s\n", cursor, strings.ToUpper(item.Kind), item.Time, item.Title))
		}
	}

	if selected, ok := m.currentAgendaItem(); ok {
		b.WriteString("\nagenda-metadata:\n")
		b.WriteString(fmt.Sprintf("id: %s\n", selected.ID))
		b.WriteString(fmt.Sprintf("kind: %s\n", selected.Kind))
		b.WriteString(fmt.Sprintf("when: %s %s\n", selected.Date, selected.Time))
	}

	return strings.TrimSuffix(b.String(), "\n")
}

func (m Model) renderFocusView() string {
	total := m.currentFocusTotal()
	progress := 0.0
	if total > 0 {
		progress = float64(total-m.Focus.RemainingSec) / float64(total)
	}

	var b strings.Builder
	b.WriteString("\nfocus:\n")
	if m.Focus.TaskTitle != "" {
		b.WriteString(fmt.Sprintf("task: %s\n", m.Focus.TaskTitle))
	} else {
		b.WriteString("task: (none selected)\n")
	}
	b.WriteString(fmt.Sprintf("phase: %s\n", strings.ToUpper(string(m.Focus.Phase))))
	b.WriteString(fmt.Sprintf("timer: %s\n", formatDuration(m.Focus.RemainingSec)))
	b.WriteString(fmt.Sprintf("progress: %s %.0f%%\n", progressBar(progress, 20), progress*100))
	b.WriteString(fmt.Sprintf("pomodoros completed: %d\n", m.Focus.CompletedPomodoros))
	b.WriteString("actions: [space]start/pause [r]reset [n]next-phase\n")
	if m.Focus.RemainingSec == 0 {
		b.WriteString("prompt: session ended, press [n] to continue")
	}
	return strings.TrimSuffix(b.String(), "\n")
}

func renderTodaySection(b *strings.Builder, title string, items []TodayItem, state TodayState) {
	b.WriteString(fmt.Sprintf("\n%s:\n", title))
	if len(items) == 0 {
		b.WriteString("  (none)\n")
		return
	}
	for _, item := range items {
		cursor := " "
		if stateCursorItem(state, item.ID) {
			cursor = ">"
		}
		b.WriteString(fmt.Sprintf("%s %s %s", cursor, urgencyBadge(item), item.Title))
		if item.ScheduledAt != "" {
			b.WriteString(fmt.Sprintf(" @%s", item.ScheduledAt))
		}
		if item.DueAt != "" {
			b.WriteString(fmt.Sprintf(" due:%s", item.DueAt))
		}
		b.WriteString("\n")
	}
}

func urgencyBadge(item TodayItem) string {
	if item.Bucket == TodayBucketOverdue || item.Priority == "Critical" {
		return "[RED]"
	}
	if item.Bucket == TodayBucketScheduled || item.Priority == "High" {
		return "[YELLOW]"
	}
	return "[GREEN]"
}

func stateCursorItem(state TodayState, id string) bool {
	if state.Cursor < 0 || state.Cursor >= len(state.Items) {
		return false
	}
	return state.Items[state.Cursor].ID == id
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

func (m *Model) bootstrapFocusTask() {
	if m.Focus.TaskID != "" {
		return
	}
	m.Focus.TaskID = m.SelectedTaskID
	if item, ok := m.currentTodayItem(); ok {
		m.Focus.TaskID = item.ID
		m.Focus.TaskTitle = item.Title
		return
	}
	if m.Focus.TaskID != "" {
		m.Focus.TaskTitle = m.Focus.TaskID
	}
}

func (m *Model) completeFocusPhase() {
	if m.Focus.Phase == FocusPhaseWork {
		m.Focus.CompletedPomodoros++
		m.Focus.Phase = FocusPhaseBreak
		m.Focus.RemainingSec = m.Focus.BreakDurationSec
		m.Focus.Running = false
		m.Status = StatusBar{Text: "break ready", IsError: false}
		return
	}
	m.Focus.Phase = FocusPhaseWork
	m.Focus.RemainingSec = m.Focus.WorkDurationSec
	m.Focus.Running = false
	m.Status = StatusBar{Text: "focus block ready", IsError: false}
}

func (m Model) currentFocusTotal() int {
	if m.Focus.Phase == FocusPhaseBreak {
		return m.Focus.BreakDurationSec
	}
	return m.Focus.WorkDurationSec
}

func focusTickCmd() tea.Cmd {
	return tea.Tick(time.Second, func(time.Time) tea.Msg { return FocusTickMsg{} })
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
		m.Status = StatusBar{Text: "command palette closed", IsError: false}
	case "enter":
		m = m.executePaletteCommand()
	case "backspace":
		if len(m.Palette.Input) > 0 {
			m.Palette.Input = m.Palette.Input[:len(m.Palette.Input)-1]
		}
	default:
		if msg.Type == tea.KeyRunes {
			m.Palette.Input += string(msg.Runes)
		}
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
	} else {
		m.Status = StatusBar{Text: res.Message, IsError: false}
	}

	m.Palette.Active = false
	m.Palette.Input = ""
	return m
}

func (m Model) renderCommandPalette() string {
	return fmt.Sprintf("\ncommand: /%s", m.Palette.Input)
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
		if !m.ReminderAck[ev.ID] {
			m.rescheduleReminder(ev, now.Add(2*time.Minute))
		}
	case "contextual":
		if inContextualWindow(now) {
			m.Status = StatusBar{Text: fmt.Sprintf("contextual reminder: %s", ev.ID), IsError: false}
		} else {
			next := nextContextualWindowStart(now)
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

func inContextualWindow(now time.Time) bool {
	h := now.Hour()
	return h >= 18 && h < 22
}

func nextContextualWindowStart(now time.Time) time.Time {
	y, mo, d := now.Date()
	loc := now.Location()
	todayStart := time.Date(y, mo, d, 18, 0, 0, 0, loc)
	if now.Before(todayStart) {
		return todayStart
	}
	return todayStart.AddDate(0, 0, 1)
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

func calendarCursorItem(state CalendarState, id string) bool {
	if state.Cursor < 0 || state.Cursor >= len(state.Items) {
		return false
	}
	return state.Items[state.Cursor].ID == id
}

func contains(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
