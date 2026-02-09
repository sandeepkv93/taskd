package update

import (
	"errors"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
