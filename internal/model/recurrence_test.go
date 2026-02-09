package model

import (
	"errors"
	"testing"
	"time"
)

func TestRecurrenceEveryWeekday(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceEveryWeekday,
		Interval: 1,
		Anchor:   time.Date(2026, 2, 9, 9, 0, 0, 0, time.UTC), // Monday
	}
	from := time.Date(2026, 2, 13, 10, 0, 0, 0, time.UTC) // Friday

	next, err := rule.NextAfter(from, nil)
	if err != nil {
		t.Fatalf("next weekday failed: %v", err)
	}
	if next.Weekday() != time.Monday || next.Format("2006-01-02 15:04") != "2026-02-16 09:00" {
		t.Fatalf("unexpected next weekday: %s", next.Format(time.RFC3339))
	}
}

func TestRecurrenceEveryNDays(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceEveryNDays,
		Interval: 2,
		Anchor:   time.Date(2026, 2, 1, 8, 0, 0, 0, time.UTC),
	}
	from := time.Date(2026, 2, 5, 9, 0, 0, 0, time.UTC)
	next, err := rule.NextAfter(from, nil)
	if err != nil {
		t.Fatalf("next every n days failed: %v", err)
	}
	if next.Format("2006-01-02 15:04") != "2026-02-07 08:00" {
		t.Fatalf("unexpected next occurrence: %s", next.Format(time.RFC3339))
	}
}

func TestRecurrenceEveryNWeeks(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceEveryNWeeks,
		Interval: 2,
		Anchor:   time.Date(2026, 2, 2, 10, 30, 0, 0, time.UTC),
	}
	from := time.Date(2026, 2, 10, 11, 0, 0, 0, time.UTC)
	next, err := rule.NextAfter(from, nil)
	if err != nil {
		t.Fatalf("next every n weeks failed: %v", err)
	}
	if next.Format("2006-01-02 15:04") != "2026-02-16 10:30" {
		t.Fatalf("unexpected next occurrence: %s", next.Format(time.RFC3339))
	}
}

func TestRecurrenceLastDayOfMonth(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceLastDayOfMonth,
		Interval: 1,
		Anchor:   time.Date(2026, 1, 31, 17, 0, 0, 0, time.UTC),
	}
	from := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	next, err := rule.NextAfter(from, nil)
	if err != nil {
		t.Fatalf("next last day failed: %v", err)
	}
	if next.Format("2006-01-02 15:04") != "2026-02-28 17:00" {
		t.Fatalf("unexpected next occurrence: %s", next.Format(time.RFC3339))
	}
}

func TestRecurrenceAfterCompletion(t *testing.T) {
	rule := RecurrenceRule{
		Type:            RecurrenceAfterComplete,
		Interval:        1,
		Anchor:          time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
		AfterCompleteIn: 6 * time.Hour,
	}
	done := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	next, err := rule.NextAfter(time.Date(2026, 2, 9, 13, 0, 0, 0, time.UTC), &done)
	if err != nil {
		t.Fatalf("next after completion failed: %v", err)
	}
	if next.Format("2006-01-02 15:04") != "2026-02-09 18:00" {
		t.Fatalf("unexpected next occurrence: %s", next.Format(time.RFC3339))
	}
}

func TestRecurrencePreview(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceEveryNDays,
		Interval: 3,
		Anchor:   time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
	}
	list, err := rule.Preview(time.Date(2026, 2, 5, 0, 0, 0, 0, time.UTC), nil, 3)
	if err != nil {
		t.Fatalf("preview failed: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("expected 3 preview items, got %d", len(list))
	}
	want := []string{"2026-02-07 09:00", "2026-02-10 09:00", "2026-02-13 09:00"}
	for i := range list {
		if got := list[i].Format("2006-01-02 15:04"); got != want[i] {
			t.Fatalf("preview[%d] got %s want %s", i, got, want[i])
		}
	}
}

func TestRecurrenceAfterCompletionRequiresCompletedAt(t *testing.T) {
	rule := RecurrenceRule{
		Type:     RecurrenceAfterComplete,
		Interval: 1,
		Anchor:   time.Date(2026, 2, 1, 9, 0, 0, 0, time.UTC),
	}
	_, err := rule.NextAfter(time.Date(2026, 2, 2, 9, 0, 0, 0, time.UTC), nil)
	if !errors.Is(err, ErrCompletionRequired) {
		t.Fatalf("expected ErrCompletionRequired, got %v", err)
	}
}
