package storage

import (
	"context"
	"errors"
)

var ErrNotFound = errors.New("storage: not found")

type Repository interface {
	CreateTask(ctx context.Context, in Task) error
	GetTask(ctx context.Context, id string) (Task, error)
	UpdateTask(ctx context.Context, in Task) error
	DeleteTask(ctx context.Context, id string) error
	ListTasks(ctx context.Context, filter TaskListFilter) ([]Task, error)

	CreateReminder(ctx context.Context, in Reminder) error
	GetReminder(ctx context.Context, id string) (Reminder, error)
	UpdateReminder(ctx context.Context, in Reminder) error
	DeleteReminder(ctx context.Context, id string) error
	ListReminders(ctx context.Context, filter ReminderListFilter) ([]Reminder, error)

	CreateTag(ctx context.Context, in Tag) error
	GetTag(ctx context.Context, id string) (Tag, error)
	UpdateTag(ctx context.Context, in Tag) error
	DeleteTag(ctx context.Context, id string) error
	ListTags(ctx context.Context, filter TagListFilter) ([]Tag, error)

	CreateRecurrence(ctx context.Context, in RecurrenceRule) error
	GetRecurrence(ctx context.Context, id string) (RecurrenceRule, error)
	UpdateRecurrence(ctx context.Context, in RecurrenceRule) error
	DeleteRecurrence(ctx context.Context, id string) error
	ListRecurrences(ctx context.Context, filter RecurrenceListFilter) ([]RecurrenceRule, error)
}
