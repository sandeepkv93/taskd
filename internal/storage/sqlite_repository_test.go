package storage

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"
)

func setupRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "taskd-test.db")
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := MigrateUp(db); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	repo, err := NewSQLiteRepository(db)
	if err != nil {
		t.Fatalf("new repo: %v", err)
	}
	return repo
}

func parseRFC3339(t *testing.T, value string) time.Time {
	t.Helper()
	out, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return out
}

func TestTaskCRUDAndList(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	created := parseRFC3339(t, "2026-02-09T12:00:00Z")

	task := Task{
		ID:          "task-1",
		Title:       "Write schema",
		Description: "Design storage layout",
		State:       "Inbox",
		Priority:    "High",
		Energy:      "Deep",
		CreatedAt:   created,
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	got, err := repo.GetTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	if got.Title != task.Title || got.State != "Inbox" {
		t.Fatalf("unexpected task get result: %#v", got)
	}

	task.Title = "Write schema v2"
	task.State = "Planned"
	if err := repo.UpdateTask(ctx, task); err != nil {
		t.Fatalf("update task: %v", err)
	}

	planned, err := repo.ListTasks(ctx, TaskListFilter{State: "Planned"})
	if err != nil {
		t.Fatalf("list tasks: %v", err)
	}
	if len(planned) != 1 || planned[0].ID != task.ID {
		t.Fatalf("unexpected planned list: %#v", planned)
	}

	if err := repo.DeleteTask(ctx, task.ID); err != nil {
		t.Fatalf("delete task: %v", err)
	}
	_, err = repo.GetTask(ctx, task.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestReminderCRUDAndList(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	now := parseRFC3339(t, "2026-02-09T12:00:00Z")
	trigger := parseRFC3339(t, "2026-02-09T13:00:00Z")

	task := Task{
		ID:          "task-reminder",
		Title:       "Task",
		Description: "",
		State:       "Inbox",
		Priority:    "Medium",
		Energy:      "Light",
		CreatedAt:   now,
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	rem := Reminder{
		ID:         "rem-1",
		TaskID:     task.ID,
		TriggerAt:  trigger,
		Type:       "Hard",
		RepeatRule: "",
		Enabled:    true,
		CreatedAt:  now,
	}
	if err := repo.CreateReminder(ctx, rem); err != nil {
		t.Fatalf("create reminder: %v", err)
	}

	got, err := repo.GetReminder(ctx, rem.ID)
	if err != nil {
		t.Fatalf("get reminder: %v", err)
	}
	if got.Type != "Hard" || !got.Enabled {
		t.Fatalf("unexpected reminder: %#v", got)
	}

	rem.Type = "Soft"
	rem.Enabled = false
	if err := repo.UpdateReminder(ctx, rem); err != nil {
		t.Fatalf("update reminder: %v", err)
	}

	enabled := false
	items, err := repo.ListReminders(ctx, ReminderListFilter{TaskID: task.ID, Enabled: &enabled})
	if err != nil {
		t.Fatalf("list reminders: %v", err)
	}
	if len(items) != 1 || items[0].ID != rem.ID {
		t.Fatalf("unexpected reminder list: %#v", items)
	}

	if err := repo.DeleteReminder(ctx, rem.ID); err != nil {
		t.Fatalf("delete reminder: %v", err)
	}
	_, err = repo.GetReminder(ctx, rem.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestTagCRUDAndList(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	now := parseRFC3339(t, "2026-02-09T12:00:00Z")

	tag := Tag{
		ID:        "tag-1",
		Name:      "finance",
		CreatedAt: now,
	}
	if err := repo.CreateTag(ctx, tag); err != nil {
		t.Fatalf("create tag: %v", err)
	}

	got, err := repo.GetTag(ctx, tag.ID)
	if err != nil {
		t.Fatalf("get tag: %v", err)
	}
	if got.Name != "finance" {
		t.Fatalf("unexpected tag: %#v", got)
	}

	tag.Name = "money"
	if err := repo.UpdateTag(ctx, tag); err != nil {
		t.Fatalf("update tag: %v", err)
	}

	list, err := repo.ListTags(ctx, TagListFilter{})
	if err != nil {
		t.Fatalf("list tags: %v", err)
	}
	if len(list) != 1 || list[0].Name != "money" {
		t.Fatalf("unexpected tag list: %#v", list)
	}

	if err := repo.DeleteTag(ctx, tag.ID); err != nil {
		t.Fatalf("delete tag: %v", err)
	}
	_, err = repo.GetTag(ctx, tag.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}

func TestRecurrenceCRUDAndList(t *testing.T) {
	repo := setupRepo(t)
	ctx := context.Background()
	now := parseRFC3339(t, "2026-02-09T12:00:00Z")
	start := parseRFC3339(t, "2026-02-10T09:00:00Z")
	next := parseRFC3339(t, "2026-02-11T09:00:00Z")

	task := Task{
		ID:          "task-rec",
		Title:       "Recurring task",
		Description: "",
		State:       "Planned",
		Priority:    "Low",
		Energy:      "Low",
		CreatedAt:   now,
	}
	if err := repo.CreateTask(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}

	rec := RecurrenceRule{
		ID:            "rec-1",
		TaskID:        task.ID,
		RuleType:      "every_n_days",
		IntervalValue: 2,
		Timezone:      "UTC",
		StartAt:       start,
		NextAt:        &next,
		Enabled:       true,
		CreatedAt:     now,
	}
	if err := repo.CreateRecurrence(ctx, rec); err != nil {
		t.Fatalf("create recurrence: %v", err)
	}

	got, err := repo.GetRecurrence(ctx, rec.ID)
	if err != nil {
		t.Fatalf("get recurrence: %v", err)
	}
	if got.RuleType != "every_n_days" || got.IntervalValue != 2 {
		t.Fatalf("unexpected recurrence: %#v", got)
	}

	rec.IntervalValue = 3
	rec.Enabled = false
	if err := repo.UpdateRecurrence(ctx, rec); err != nil {
		t.Fatalf("update recurrence: %v", err)
	}

	enabled := false
	list, err := repo.ListRecurrences(ctx, RecurrenceListFilter{TaskID: task.ID, Enabled: &enabled})
	if err != nil {
		t.Fatalf("list recurrence: %v", err)
	}
	if len(list) != 1 || list[0].IntervalValue != 3 {
		t.Fatalf("unexpected recurrence list: %#v", list)
	}

	if err := repo.DeleteRecurrence(ctx, rec.ID); err != nil {
		t.Fatalf("delete recurrence: %v", err)
	}
	_, err = repo.GetRecurrence(ctx, rec.ID)
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got: %v", err)
	}
}
