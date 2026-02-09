package model

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

var ErrInvalidReminderType = errors.New("model: invalid reminder type")

type ReminderType string

const (
	ReminderTypeHard       ReminderType = "Hard"
	ReminderTypeSoft       ReminderType = "Soft"
	ReminderTypeNagging    ReminderType = "Nagging"
	ReminderTypeContextual ReminderType = "Contextual"
)

func (r ReminderType) IsValid() bool {
	switch r {
	case ReminderTypeHard, ReminderTypeSoft, ReminderTypeNagging, ReminderTypeContextual:
		return true
	default:
		return false
	}
}

type Reminder struct {
	ID          string
	TaskID      string
	TriggerTime time.Time
	Type        ReminderType
	RepeatRule  string
	LastFiredAt *time.Time
	Enabled     bool
}

func (r Reminder) Validate() error {
	if strings.TrimSpace(r.ID) == "" {
		return errors.New("model: reminder id is required")
	}
	if strings.TrimSpace(r.TaskID) == "" {
		return errors.New("model: reminder task_id is required")
	}
	if r.TriggerTime.IsZero() {
		return errors.New("model: reminder trigger_time is required")
	}
	if !r.Type.IsValid() {
		return fmt.Errorf("%w: %q", ErrInvalidReminderType, r.Type)
	}
	return nil
}
