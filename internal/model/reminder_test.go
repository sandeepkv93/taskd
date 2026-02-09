package model

import (
	"errors"
	"testing"
	"time"
)

func TestReminderValidateSuccess(t *testing.T) {
	rem := Reminder{
		ID:          "rem-1",
		TaskID:      "task-1",
		TriggerTime: time.Date(2026, 2, 9, 13, 0, 0, 0, time.UTC),
		Type:        ReminderTypeHard,
		Enabled:     true,
	}
	if err := rem.Validate(); err != nil {
		t.Fatalf("expected valid reminder, got error: %v", err)
	}
}

func TestReminderValidateInvalidType(t *testing.T) {
	rem := Reminder{
		ID:          "rem-1",
		TaskID:      "task-1",
		TriggerTime: time.Date(2026, 2, 9, 13, 0, 0, 0, time.UTC),
		Type:        ReminderType("invalid"),
	}
	err := rem.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidReminderType) {
		t.Fatalf("expected ErrInvalidReminderType, got: %v", err)
	}
}

func TestReminderTypeIsValid(t *testing.T) {
	valid := []ReminderType{
		ReminderTypeHard,
		ReminderTypeSoft,
		ReminderTypeNagging,
		ReminderTypeContextual,
	}
	for _, item := range valid {
		if !item.IsValid() {
			t.Fatalf("expected valid reminder type: %q", item)
		}
	}
	if ReminderType("other").IsValid() {
		t.Fatal("expected invalid type")
	}
}
