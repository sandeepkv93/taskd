package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidState    = errors.New("model: invalid task state")
	ErrInvalidPriority = errors.New("model: invalid task priority")
	ErrInvalidEnergy   = errors.New("model: invalid task energy")
)

type TaskState string

const (
	TaskStateInbox   TaskState = "Inbox"
	TaskStatePlanned TaskState = "Planned"
	TaskStateDone    TaskState = "Done"
	TaskStateSnoozed TaskState = "Snoozed"
)

func (s TaskState) IsValid() bool {
	switch s {
	case TaskStateInbox, TaskStatePlanned, TaskStateDone, TaskStateSnoozed:
		return true
	default:
		return false
	}
}

type Priority string

const (
	PriorityLow      Priority = "Low"
	PriorityMedium   Priority = "Medium"
	PriorityHigh     Priority = "High"
	PriorityCritical Priority = "Critical"
)

func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh, PriorityCritical:
		return true
	default:
		return false
	}
}

type Energy string

const (
	EnergyDeep   Energy = "Deep"
	EnergyLight  Energy = "Light"
	EnergySocial Energy = "Social"
	EnergyLow    Energy = "Low"
)

func (e Energy) IsValid() bool {
	switch e {
	case EnergyDeep, EnergyLight, EnergySocial, EnergyLow:
		return true
	default:
		return false
	}
}

type Task struct {
	ID          string
	Title       string
	Description string
	State       TaskState
	Priority    Priority
	Energy      Energy
	Tags        []string
	CreatedAt   time.Time
	CompletedAt *time.Time
}

func (t Task) Validate() error {
	if strings.TrimSpace(t.ID) == "" {
		return errors.New("model: task id is required")
	}
	if strings.TrimSpace(t.Title) == "" {
		return errors.New("model: task title is required")
	}
	if !t.State.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidState, t.State)
	}
	if !t.Priority.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidPriority, t.Priority)
	}
	if !t.Energy.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidEnergy, t.Energy)
	}
	if t.CreatedAt.IsZero() {
		return errors.New("model: task created_at is required")
	}
	if t.State == TaskStateDone && t.CompletedAt == nil {
		return errors.New("model: completed_at is required when task state is Done")
	}
	if t.State != TaskStateDone && t.CompletedAt != nil {
		return errors.New("model: completed_at must be nil when task state is not Done")
	}
	return nil
}
