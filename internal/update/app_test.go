package update

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/scheduler"
)

type fakeNotifier struct {
	count int
	last  Notification
}

func (f *fakeNotifier) Send(n Notification) error {
	f.count++
	f.last = n
	return nil
}

func TestNewModelDefaults(t *testing.T) {
	m := NewModel()
	if m.CurrentView != ViewToday {
		t.Fatalf("expected default view %q, got %q", ViewToday, m.CurrentView)
	}
	if m.Sort != SortCreatedDesc {
		t.Fatalf("expected default sort %q, got %q", SortCreatedDesc, m.Sort)
	}
	if m.Keys.Quit != "q" {
		t.Fatalf("expected quit key q, got %q", m.Keys.Quit)
	}
}

func TestUpdateKeySwitchesView(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	next := updated.(Model)
	if next.CurrentView != ViewInbox {
		t.Fatalf("expected inbox view, got %q", next.CurrentView)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	next = updated.(Model)
	if next.CurrentView != ViewFocus {
		t.Fatalf("expected focus view, got %q", next.CurrentView)
	}
}

func TestUpdateSwitchViewMsg(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(SwitchViewMsg{View: ViewCalendar})
	next := updated.(Model)
	if next.CurrentView != ViewCalendar {
		t.Fatalf("expected calendar view, got %q", next.CurrentView)
	}

	updated, _ = next.Update(SwitchViewMsg{View: View("Unknown")})
	next = updated.(Model)
	if next.CurrentView != ViewCalendar {
		t.Fatalf("expected view unchanged for unknown view, got %q", next.CurrentView)
	}
}

func TestUpdateStatusAndError(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(SetStatusMsg{Text: "ready", IsError: false})
	next := updated.(Model)
	if next.Status.Text != "ready" || next.Status.IsError {
		t.Fatalf("unexpected status: %+v", next.Status)
	}

	errMsg := errors.New("boom")
	updated, _ = next.Update(AppErrorMsg{Err: errMsg})
	next = updated.(Model)
	if next.LastError == nil || next.LastError.Error() != "boom" {
		t.Fatalf("expected last error boom, got: %v", next.LastError)
	}
	if !next.Status.IsError || next.Status.Text != "boom" {
		t.Fatalf("unexpected error status: %+v", next.Status)
	}

	updated, _ = next.Update(ClearStatusMsg{})
	next = updated.(Model)
	if next.Status.Text != "" || next.Status.IsError {
		t.Fatalf("expected cleared status, got: %+v", next.Status)
	}
}

func TestUpdateQuitKey(t *testing.T) {
	m := NewModel()
	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	next := updated.(Model)
	if !next.Quitting {
		t.Fatal("expected quitting flag true")
	}
	if cmd == nil {
		t.Fatal("expected quit command")
	}
}

func TestViewContainsCoreState(t *testing.T) {
	m := NewModel()
	m.SelectedTaskID = "task-42"
	m.Status = StatusBar{Text: "all good"}
	out := m.View()
	if !strings.Contains(out, "view: Today") {
		t.Fatalf("expected view text in output: %q", out)
	}
	if !strings.Contains(out, "selected: task-42") {
		t.Fatalf("expected selected task in output: %q", out)
	}
	if !strings.Contains(out, "status: all good") {
		t.Fatalf("expected status in output: %q", out)
	}
}

func TestInboxQuickAddWithKeyboard(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(SwitchViewMsg{View: ViewInbox})
	next := updated.(Model)

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("write tests")})
	next = updated.(Model)
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next = updated.(Model)

	if len(next.Inbox.Items) != 1 {
		t.Fatalf("expected 1 inbox item, got %d", len(next.Inbox.Items))
	}
	if next.Inbox.Items[0].Title != "write tests" {
		t.Fatalf("unexpected inbox title: %q", next.Inbox.Items[0].Title)
	}
	if next.Inbox.Input != "" {
		t.Fatalf("expected empty input after capture, got %q", next.Inbox.Input)
	}
}

func TestInboxBulkSelectScheduleAndTag(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewInbox
	m.addInboxItem("task one")
	m.addInboxItem("task two")

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	next := updated.(Model)
	if len(next.Inbox.Selected) != 2 {
		t.Fatalf("expected 2 selected items, got %d", len(next.Inbox.Selected))
	}

	updated, _ = next.Update(BulkScheduleInboxMsg{When: "tomorrow 09:00"})
	next = updated.(Model)
	for _, item := range next.Inbox.Items {
		if item.ScheduledFor != "tomorrow 09:00" {
			t.Fatalf("expected scheduled value for %q, got %q", item.ID, item.ScheduledFor)
		}
	}

	updated, _ = next.Update(BulkTagInboxMsg{Tag: "triage"})
	next = updated.(Model)
	for _, item := range next.Inbox.Items {
		if len(item.Tags) != 1 || item.Tags[0] != "triage" {
			t.Fatalf("expected triage tag on %q, got %#v", item.ID, item.Tags)
		}
	}
}

func TestTodayViewRendersGroupedSectionsAndMetadata(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewToday
	m.Today.Items = []TodayItem{
		{ID: "a", Title: "sched", Bucket: TodayBucketScheduled, ScheduledAt: "10:00", Priority: "High", Tags: []string{"team"}, Notes: "note-a"},
		{ID: "b", Title: "any", Bucket: TodayBucketAnytime, Priority: "Low", Notes: "note-b"},
		{ID: "c", Title: "late", Bucket: TodayBucketOverdue, DueAt: "Yesterday", Priority: "Critical", Notes: "note-c"},
	}
	m.Today.Cursor = 2
	m.syncSelectedTaskToTodayCursor()

	out := m.View()
	if !strings.Contains(out, "Scheduled:") || !strings.Contains(out, "Anytime:") || !strings.Contains(out, "Overdue:") {
		t.Fatalf("missing grouped sections in today view: %q", out)
	}
	if !strings.Contains(out, "[YELLOW] sched") {
		t.Fatalf("missing scheduled urgency marker: %q", out)
	}
	if !strings.Contains(out, "[GREEN] any") {
		t.Fatalf("missing anytime urgency marker: %q", out)
	}
	if !strings.Contains(out, "[RED] late") {
		t.Fatalf("missing overdue urgency marker: %q", out)
	}
	if !strings.Contains(out, "metadata:") || !strings.Contains(out, "id: c") {
		t.Fatalf("missing metadata pane for selected item: %q", out)
	}
}

func TestTodayKeyNavigationUpdatesSelection(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewToday
	m.Today.Items = []TodayItem{
		{ID: "first", Title: "A", Bucket: TodayBucketAnytime, Priority: "Low"},
		{ID: "second", Title: "B", Bucket: TodayBucketScheduled, Priority: "High"},
	}
	m.Today.Cursor = 0
	m.syncSelectedTaskToTodayCursor()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(Model)
	if next.Today.Cursor != 1 {
		t.Fatalf("expected cursor at 1, got %d", next.Today.Cursor)
	}
	if next.SelectedTaskID != "second" {
		t.Fatalf("expected selected task second, got %q", next.SelectedTaskID)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	next = updated.(Model)
	if next.Today.Cursor != 0 {
		t.Fatalf("expected cursor at 0, got %d", next.Today.Cursor)
	}
	if next.SelectedTaskID != "first" {
		t.Fatalf("expected selected task first, got %q", next.SelectedTaskID)
	}
}

func TestCalendarModeSwitchAndPeriodNavigation(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewCalendar
	m.Calendar.FocusDate = time.Date(2026, 2, 9, 0, 0, 0, 0, time.UTC)

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	next := updated.(Model)
	if next.Calendar.Mode != CalendarModeDay {
		t.Fatalf("expected day mode, got %q", next.Calendar.Mode)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	next = updated.(Model)
	if next.Calendar.FocusDate.Format("2006-01-02") != "2026-02-10" {
		t.Fatalf("expected +1 day focus, got %s", next.Calendar.FocusDate.Format("2006-01-02"))
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	next = updated.(Model)
	if next.Calendar.Mode != CalendarModeWeek {
		t.Fatalf("expected week mode, got %q", next.Calendar.Mode)
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	next = updated.(Model)
	if next.Calendar.FocusDate.Format("2006-01-02") != "2026-02-03" {
		t.Fatalf("expected -7 day focus, got %s", next.Calendar.FocusDate.Format("2006-01-02"))
	}

	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
	next = updated.(Model)
	if next.Calendar.Mode != CalendarModeMonth {
		t.Fatalf("expected month mode, got %q", next.Calendar.Mode)
	}
}

func TestCalendarAgendaGroupingAndCursorSelection(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewCalendar
	m.Calendar.Items = []AgendaItem{
		{ID: "c1", Title: "A", Date: "2026-02-09", Time: "11:00", Kind: "task"},
		{ID: "c2", Title: "B", Date: "2026-02-10", Time: "09:00", Kind: "event"},
		{ID: "c3", Title: "C", Date: "2026-02-10", Time: "17:00", Kind: "task"},
	}
	m.Calendar.Cursor = 0
	m.syncSelectedTaskToCalendarCursor()

	out := m.View()
	if !strings.Contains(out, "2026-02-09:") || !strings.Contains(out, "2026-02-10:") {
		t.Fatalf("expected grouped agenda dates in output: %q", out)
	}
	if !strings.Contains(out, "agenda-metadata:") || !strings.Contains(out, "id: c1") {
		t.Fatalf("expected metadata for selected agenda item: %q", out)
	}

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	next := updated.(Model)
	if next.Calendar.Cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", next.Calendar.Cursor)
	}
	if next.SelectedTaskID != "c2" {
		t.Fatalf("expected selected c2, got %q", next.SelectedTaskID)
	}
}

func TestFocusStartPauseTickAndCompletion(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewFocus
	m.Focus.WorkDurationSec = 3
	m.Focus.BreakDurationSec = 2
	m.Focus.RemainingSec = 3
	m.Focus.Phase = FocusPhaseWork
	m.Focus.Running = false

	updated, cmd := m.Update(tea.KeyMsg{Type: tea.KeySpace})
	next := updated.(Model)
	if !next.Focus.Running {
		t.Fatal("expected focus running after start")
	}
	if cmd == nil {
		t.Fatal("expected tick cmd on focus start")
	}

	updated, cmd = next.Update(FocusTickMsg{})
	next = updated.(Model)
	if next.Focus.RemainingSec != 2 {
		t.Fatalf("expected remaining 2, got %d", next.Focus.RemainingSec)
	}
	if cmd == nil {
		t.Fatal("expected tick cmd while timer running")
	}

	updated, _ = next.Update(FocusTickMsg{})
	next = updated.(Model)
	updated, _ = next.Update(FocusTickMsg{})
	next = updated.(Model)
	if next.Focus.RemainingSec != 0 {
		t.Fatalf("expected remaining 0, got %d", next.Focus.RemainingSec)
	}
	if next.Focus.Running {
		t.Fatal("expected focus stopped on completion")
	}
	if !strings.Contains(next.Status.Text, "work session complete") {
		t.Fatalf("expected completion prompt, got %q", next.Status.Text)
	}
}

func TestFocusPhaseTransitionWithNextKey(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewFocus
	m.Focus.WorkDurationSec = 5
	m.Focus.BreakDurationSec = 2
	m.Focus.RemainingSec = 0
	m.Focus.Phase = FocusPhaseWork

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	next := updated.(Model)
	if next.Focus.Phase != FocusPhaseBreak {
		t.Fatalf("expected break phase, got %q", next.Focus.Phase)
	}
	if next.Focus.RemainingSec != 2 {
		t.Fatalf("expected break remaining 2, got %d", next.Focus.RemainingSec)
	}
	if next.Focus.CompletedPomodoros != 1 {
		t.Fatalf("expected completed pomodoros 1, got %d", next.Focus.CompletedPomodoros)
	}

	next.Focus.RemainingSec = 0
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	next = updated.(Model)
	if next.Focus.Phase != FocusPhaseWork {
		t.Fatalf("expected work phase, got %q", next.Focus.Phase)
	}
	if next.Focus.RemainingSec != 5 {
		t.Fatalf("expected work remaining 5, got %d", next.Focus.RemainingSec)
	}
}

func TestFocusViewRendering(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewFocus
	m.Focus.TaskTitle = "Write integration tests"
	m.Focus.Phase = FocusPhaseWork
	m.Focus.WorkDurationSec = 10
	m.Focus.RemainingSec = 5
	m.Focus.CompletedPomodoros = 2

	out := m.View()
	if !strings.Contains(out, "focus:") {
		t.Fatalf("expected focus section, got %q", out)
	}
	if !strings.Contains(out, "task: Write integration tests") {
		t.Fatalf("expected focus task title, got %q", out)
	}
	if !strings.Contains(out, "phase: WORK") {
		t.Fatalf("expected phase in output, got %q", out)
	}
	if !strings.Contains(out, "timer: 00:05") {
		t.Fatalf("expected timer in output, got %q", out)
	}
	if !strings.Contains(out, "pomodoros completed: 2") {
		t.Fatalf("expected pomodoro count in output, got %q", out)
	}
}

func TestInitWithSchedulerReturnsReminderCmd(t *testing.T) {
	engine := scheduler.NewEngine(1)
	m := NewModelWithScheduler(engine)
	if cmd := m.Init(); cmd == nil {
		t.Fatal("expected reminder wait cmd when scheduler is attached")
	}
}

func TestUpdateReminderDueMsgAppendsLogAndRearms(t *testing.T) {
	engine := scheduler.NewEngine(1)
	m := NewModelWithScheduler(engine)
	ev := scheduler.ReminderEvent{
		ID:        "rem-1",
		TriggerAt: time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC),
	}

	updated, cmd := m.Update(ReminderDueMsg{Event: ev})
	next := updated.(Model)
	if len(next.ReminderLog) != 1 || next.ReminderLog[0].ID != "rem-1" {
		t.Fatalf("unexpected reminder log: %#v", next.ReminderLog)
	}
	if cmd == nil {
		t.Fatal("expected reminder listener rearm cmd")
	}
	if !strings.Contains(next.Status.Text, "reminder fired") {
		t.Fatalf("expected reminder status text, got %q", next.Status.Text)
	}
}

func TestReminderBehaviorHard(t *testing.T) {
	m := NewModel()
	ev := scheduler.ReminderEvent{ID: "r-hard", Type: "Hard", TriggerAt: time.Now().UTC()}
	m.applyReminderBehavior(ev, time.Now().UTC())
	if !m.Status.IsError {
		t.Fatal("expected hard reminder status as error")
	}
	if !strings.Contains(m.Status.Text, "HARD reminder") {
		t.Fatalf("unexpected hard status: %q", m.Status.Text)
	}
}

func TestReminderBehaviorSoftFollowUpOnce(t *testing.T) {
	m := NewModel()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	ev := scheduler.ReminderEvent{ID: "r-soft", Type: "Soft", TriggerAt: now}

	m.applyReminderBehavior(ev, now)
	if !m.SoftFollowedUp["r-soft"] {
		t.Fatal("expected soft reminder to mark follow-up sent")
	}
	firstStatus := m.Status.Text
	m.applyReminderBehavior(ev, now.Add(time.Minute))
	if m.Status.Text == "" || firstStatus == "" {
		t.Fatalf("expected non-empty statuses, got first=%q second=%q", firstStatus, m.Status.Text)
	}
}

func TestReminderBehaviorNaggingAndAcknowledge(t *testing.T) {
	m := NewModel()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	ev := scheduler.ReminderEvent{ID: "r-nag", Type: "Nagging", TriggerAt: now}

	m.applyReminderBehavior(ev, now)
	if !strings.Contains(m.Status.Text, "nagging reminder") {
		t.Fatalf("unexpected nagging status: %q", m.Status.Text)
	}

	updated, _ := m.Update(AcknowledgeReminderMsg{ID: "r-nag"})
	next := updated.(Model)
	if !next.ReminderAck["r-nag"] {
		t.Fatal("expected reminder ack to be recorded")
	}
}

func TestReminderBehaviorContextualWindowAndDeferral(t *testing.T) {
	m := NewModel()
	inWindow := time.Date(2026, 2, 9, 19, 0, 0, 0, time.UTC)
	ev := scheduler.ReminderEvent{ID: "r-ctx", Type: "Contextual", TriggerAt: inWindow}
	m.applyReminderBehavior(ev, inWindow)
	if !strings.Contains(m.Status.Text, "contextual reminder") {
		t.Fatalf("expected contextual reminder status, got %q", m.Status.Text)
	}

	outWindow := time.Date(2026, 2, 9, 10, 0, 0, 0, time.UTC)
	m.applyReminderBehavior(ev, outWindow)
	if !strings.Contains(m.Status.Text, "contextual deferred") {
		t.Fatalf("expected contextual deferred status, got %q", m.Status.Text)
	}
	if nextContextualWindowStartForRule(outWindow, "evening").Hour() != 18 {
		t.Fatalf("expected next contextual start at 18:00, got %s", nextContextualWindowStartForRule(outWindow, "evening").Format("15:04"))
	}
}

func TestReminderBehaviorNaggingStopsWhenTaskCompleted(t *testing.T) {
	m := NewModel()
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	m.CompletedTasks["task-x"] = true
	ev := scheduler.ReminderEvent{ID: "r-nag2", TaskID: "task-x", Type: "Nagging", TriggerAt: now}
	m.applyReminderBehavior(ev, now)
	if strings.Contains(strings.ToLower(m.Status.Text), "failed") {
		t.Fatalf("unexpected reschedule failure for completed task: %q", m.Status.Text)
	}
}

func TestCommandPaletteAddCommand(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	next := updated.(Model)
	if !next.Palette.Active {
		t.Fatal("expected command palette active")
	}

	for _, r := range []rune("add buy groceries") {
		updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		next = updated.(Model)
	}
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next = updated.(Model)

	if next.Palette.Active {
		t.Fatal("expected command palette to close after execution")
	}
	if next.CurrentView != ViewInbox {
		t.Fatalf("expected view inbox after add command, got %q", next.CurrentView)
	}
	if len(next.Inbox.Items) == 0 || next.Inbox.Items[len(next.Inbox.Items)-1].Title != "buy groceries" {
		t.Fatalf("expected inbox item from add command, got %#v", next.Inbox.Items)
	}
}

func TestCommandPaletteRescheduleSelected(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewInbox
	m.addInboxItem("one")
	m.addInboxItem("two")
	m.Inbox.Selected[m.Inbox.Items[0].ID] = true

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	next := updated.(Model)
	for _, r := range []rune("reschedule selected next monday") {
		updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		next = updated.(Model)
	}
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next = updated.(Model)

	if next.Inbox.Items[0].ScheduledFor != "next monday" {
		t.Fatalf("expected selected item rescheduled, got %q", next.Inbox.Items[0].ScheduledFor)
	}
	if next.Inbox.Items[1].ScheduledFor != "" {
		t.Fatalf("expected unselected item unchanged, got %q", next.Inbox.Items[1].ScheduledFor)
	}
}

func TestCommandPaletteUnknownCommandSetsError(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	next := updated.(Model)
	for _, r := range []rune("unknown command") {
		updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		next = updated.(Model)
	}
	updated, _ = next.Update(tea.KeyMsg{Type: tea.KeyEnter})
	next = updated.(Model)

	if !next.Status.IsError {
		t.Fatal("expected error status for unknown command")
	}
	if !strings.Contains(next.Status.Text, "unknown_command") {
		t.Fatalf("expected structured error text, got %q", next.Status.Text)
	}
}

func TestHelpToggleAndContextualBindings(t *testing.T) {
	m := NewModel()
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	next := updated.(Model)
	if !next.HelpVisible {
		t.Fatal("expected help to be visible after toggle")
	}
	out := next.View()
	if !strings.Contains(out, "help:") || !strings.Contains(out, "global:") {
		t.Fatalf("expected help panel in view output, got %q", out)
	}
	if !strings.Contains(out, "today view:") {
		t.Fatalf("expected today contextual help section, got %q", out)
	}

	updated, _ = next.Update(SwitchViewMsg{View: ViewInbox})
	next = updated.(Model)
	out = next.View()
	if !strings.Contains(out, "inbox view:") {
		t.Fatalf("expected inbox contextual help section, got %q", out)
	}
	if !strings.Contains(out, "capture inbox item") {
		t.Fatalf("expected inbox-specific keybinding text, got %q", out)
	}
}

func TestViewBindingsExistForAllViews(t *testing.T) {
	m := NewModel()
	views := []View{ViewToday, ViewInbox, ViewCalendar, ViewFocus}
	for _, view := range views {
		m.CurrentView = view
		bindings := m.viewBindings()
		if len(bindings) == 0 {
			t.Fatalf("expected bindings for view %s", view)
		}
	}
}

func TestInTUINotificationOnReminderDue(t *testing.T) {
	m := NewModel()
	ev := scheduler.ReminderEvent{ID: "n-1", Type: "Soft", TriggerAt: time.Now().UTC()}

	updated, _ := m.Update(ReminderDueMsg{Event: ev})
	next := updated.(Model)
	if len(next.Notifications) == 0 {
		t.Fatal("expected in-TUI notification to be recorded")
	}
	last := next.Notifications[len(next.Notifications)-1]
	if !strings.Contains(last.Body, "soft reminder") {
		t.Fatalf("unexpected notification body: %q", last.Body)
	}
}

func TestDesktopNotificationOptional(t *testing.T) {
	f := &fakeNotifier{}
	m := NewModelWithRuntime(nil, true, f)
	updated, _ := m.Update(SetStatusMsg{Text: "hello", IsError: false})
	next := updated.(Model)
	if f.count == 0 {
		t.Fatal("expected desktop notifier to be called when enabled")
	}
	if !strings.Contains(f.last.Body, "hello") {
		t.Fatalf("unexpected desktop notification body: %q", f.last.Body)
	}

	f2 := &fakeNotifier{}
	m2 := NewModelWithRuntime(nil, false, f2)
	updated, _ = m2.Update(SetStatusMsg{Text: "hello", IsError: false})
	next = updated.(Model)
	_ = next
	if f2.count != 0 {
		t.Fatalf("expected desktop notifier not to be called when disabled, got %d", f2.count)
	}
}

func TestTemporalDebtScoring(t *testing.T) {
	m := NewModel()
	m.Today.Items = []TodayItem{
		{ID: "a", Title: "late critical", Bucket: TodayBucketOverdue, Priority: "Critical", Notes: "snoozed yesterday"},
		{ID: "b", Title: "late normal", Bucket: TodayBucketOverdue, Priority: "Medium"},
		{ID: "c", Title: "normal", Bucket: TodayBucketAnytime, Priority: "Low", Notes: "snoozed once"},
	}

	score := m.computeTemporalDebtScore()
	// a: overdue(2)+snoozed(1)+critical overdue(1)=4, b:2, c:1 => total 7
	if score != 7 {
		t.Fatalf("expected temporal debt score 7, got %d", score)
	}
}

func TestEnergyAwareSuggestions(t *testing.T) {
	m := NewModel()
	m.Today.Items = []TodayItem{
		{ID: "d1", Title: "deep architecture refactor", Bucket: TodayBucketAnytime, Notes: "complex module"},
		{ID: "l1", Title: "write release notes", Bucket: TodayBucketAnytime, Notes: "docs"},
		{ID: "s1", Title: "team standup call", Bucket: TodayBucketScheduled, Notes: "meeting"},
		{ID: "o1", Title: "submit tax forms", Bucket: TodayBucketOverdue, Notes: "finance"},
	}

	suggestions := m.computeEnergySuggestions(45, 5)
	if len(suggestions) == 0 {
		t.Fatal("expected suggestions for 45-minute window")
	}
	for _, s := range suggestions {
		if s.Minutes > 45 {
			t.Fatalf("suggestion exceeds window: %+v", s)
		}
	}
	foundOverdue := false
	for _, s := range suggestions {
		if s.TaskID == "o1" {
			foundOverdue = true
			if !strings.Contains(strings.ToLower(s.Reason), "overdue") {
				t.Fatalf("expected overdue reason for o1, got %q", s.Reason)
			}
		}
	}
	if !foundOverdue {
		t.Fatal("expected overdue task to appear in suggestions when feasible")
	}
}

func TestProductivityViewIncludesDebtAndSuggestions(t *testing.T) {
	m := NewModel()
	m.Today.Items = []TodayItem{
		{ID: "x", Title: "overdue email", Bucket: TodayBucketOverdue, Notes: "snoozed"},
		{ID: "y", Title: "write docs", Bucket: TodayBucketAnytime, Notes: "docs"},
	}
	m.Productivity.AvailableMinutes = 30
	m.refreshProductivitySignals()

	out := m.View()
	if !strings.Contains(out, "productivity:") {
		t.Fatalf("expected productivity panel in output: %q", out)
	}
	if !strings.Contains(out, "temporal-debt:") {
		t.Fatalf("expected temporal debt in output: %q", out)
	}
	if !strings.Contains(out, "suggestions:") {
		t.Fatalf("expected suggestions in output: %q", out)
	}
}

func TestTodaySectionCollapseToggle(t *testing.T) {
	m := NewModel()
	m.CurrentView = ViewToday
	m.Today.Items = []TodayItem{
		{ID: "a", Title: "sched", Bucket: TodayBucketScheduled, Priority: "High"},
		{ID: "b", Title: "any", Bucket: TodayBucketAnytime, Priority: "Low"},
	}
	m.Today.Cursor = 0
	m.syncSelectedTaskToTodayCursor()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	next := updated.(Model)
	if !next.todayCollapsed[TodayBucketScheduled] {
		t.Fatalf("expected scheduled section to be collapsed")
	}
	out := next.View()
	if !strings.Contains(out, "(collapsed)") {
		t.Fatalf("expected collapsed marker in today output: %q", out)
	}
}

func TestDensityCycleChangesStatus(t *testing.T) {
	m := NewModel()
	initial := m.uiDensity
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	next := updated.(Model)
	if next.uiDensity == initial {
		t.Fatalf("expected density to change from %d", initial)
	}
	if !strings.Contains(next.Status.Text, "density level") {
		t.Fatalf("expected density status text, got %q", next.Status.Text)
	}
}

func TestCompletedTaskStatePersistsAndReloads(t *testing.T) {
	statePath := filepath.Join(t.TempDir(), "state.json")
	cfg := DefaultRuntimeConfig()
	cfg.CompletionStatePath = statePath
	m := NewModelWithConfig(nil, nil, cfg)
	m.Focus.TaskID = "task-a"
	m.Focus.Phase = FocusPhaseWork
	m.completeFocusPhase()

	raw, err := os.ReadFile(statePath)
	if err != nil {
		t.Fatalf("expected state file to be written, got err: %v", err)
	}
	if !strings.Contains(string(raw), "task-a") {
		t.Fatalf("expected persisted task id in state file, got: %s", string(raw))
	}

	loaded := NewModelWithConfig(nil, nil, cfg)
	if !loaded.CompletedTasks["task-a"] {
		t.Fatalf("expected completed task to reload from state file, got %#v", loaded.CompletedTasks)
	}
}

func TestNaggingReminderSkipsRescheduleWhenCompletedTaskLoaded(t *testing.T) {
	engine := scheduler.NewEngine(4)
	engine.Start()
	defer engine.Stop()

	statePath := filepath.Join(t.TempDir(), "state.json")
	if err := os.WriteFile(statePath, []byte("{\"completed_task_ids\":[\"task-complete\"]}\n"), 0o644); err != nil {
		t.Fatalf("seed state write failed: %v", err)
	}

	cfg := DefaultRuntimeConfig()
	cfg.CompletionStatePath = statePath
	m := NewModelWithConfig(engine, nil, cfg)

	now := time.Now().UTC()
	ev := scheduler.ReminderEvent{
		ID:        "nag-1",
		TaskID:    "task-complete",
		Type:      "Nagging",
		TriggerAt: now,
	}
	m.applyReminderBehavior(ev, now)

	select {
	case got := <-engine.C():
		t.Fatalf("unexpected nagging reschedule for completed task: %#v", got)
	case <-time.After(250 * time.Millisecond):
	}
}
