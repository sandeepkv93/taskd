package model

import (
	"errors"
	"testing"
	"time"
)

func TestTaskValidateSuccess(t *testing.T) {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	task := Task{
		ID:        "task-1",
		Title:     "Implement model validation",
		State:     TaskStatePlanned,
		Priority:  PriorityHigh,
		Energy:    EnergyDeep,
		CreatedAt: now,
	}
	if err := task.Validate(); err != nil {
		t.Fatalf("expected valid task, got error: %v", err)
	}
}

func TestTaskValidateDoneRequiresCompletedAt(t *testing.T) {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	task := Task{
		ID:        "task-1",
		Title:     "Done task",
		State:     TaskStateDone,
		Priority:  PriorityMedium,
		Energy:    EnergyLight,
		CreatedAt: now,
	}
	err := task.Validate()
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "model: completed_at is required when task state is Done" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTaskValidateInvalidEnums(t *testing.T) {
	now := time.Date(2026, 2, 9, 12, 0, 0, 0, time.UTC)
	task := Task{
		ID:        "task-1",
		Title:     "Bad state",
		State:     TaskState("Invalid"),
		Priority:  PriorityLow,
		Energy:    EnergyLow,
		CreatedAt: now,
	}
	err := task.Validate()
	if err == nil || !errors.Is(err, ErrInvalidState) {
		t.Fatalf("expected ErrInvalidState, got: %v", err)
	}

	task.State = TaskStateInbox
	task.Priority = Priority("Bad")
	err = task.Validate()
	if err == nil || !errors.Is(err, ErrInvalidPriority) {
		t.Fatalf("expected ErrInvalidPriority, got: %v", err)
	}

	task.Priority = PriorityMedium
	task.Energy = Energy("Bad")
	err = task.Validate()
	if err == nil || !errors.Is(err, ErrInvalidEnergy) {
		t.Fatalf("expected ErrInvalidEnergy, got: %v", err)
	}
}
