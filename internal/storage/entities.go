package storage

import "time"

type Task struct {
	ID          string
	Title       string
	Description string
	State       string
	Priority    string
	Energy      string
	ScheduledAt *time.Time
	DueAt       *time.Time
	CreatedAt   time.Time
	CompletedAt *time.Time
}

type Reminder struct {
	ID         string
	TaskID     string
	TriggerAt  time.Time
	Type       string
	RepeatRule string
	LastFired  *time.Time
	Enabled    bool
	CreatedAt  time.Time
}

type Tag struct {
	ID        string
	Name      string
	CreatedAt time.Time
}

type RecurrenceRule struct {
	ID            string
	TaskID        string
	RuleType      string
	IntervalValue int
	Timezone      string
	StartAt       time.Time
	NextAt        *time.Time
	Enabled       bool
	CreatedAt     time.Time
}

type TaskListFilter struct {
	State  string
	Limit  int
	Offset int
}

type ReminderListFilter struct {
	TaskID  string
	Enabled *bool
	Limit   int
	Offset  int
}

type TagListFilter struct {
	Limit  int
	Offset int
}

type RecurrenceListFilter struct {
	TaskID  string
	Enabled *bool
	Limit   int
	Offset  int
}
