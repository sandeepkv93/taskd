package update

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sandeepkv93/taskd/internal/scheduler"
)

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
	if nextContextualWindowStart(outWindow).Hour() != 18 {
		t.Fatalf("expected next contextual start at 18:00, got %s", nextContextualWindowStart(outWindow).Format("15:04"))
	}
}
