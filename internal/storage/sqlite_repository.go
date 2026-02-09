package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

const sqliteTimeLayout = time.RFC3339Nano

type SQLiteRepository struct {
	db *sql.DB
}

func NewSQLiteRepository(db *sql.DB) (*SQLiteRepository, error) {
	if db == nil {
		return nil, errors.New("storage: nil db")
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	return &SQLiteRepository{db: db}, nil
}

func OpenSQLite(path string) (*SQLiteRepository, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	repo, err := NewSQLiteRepository(db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

func (r *SQLiteRepository) CreateTask(ctx context.Context, in Task) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tasks (id, title, description, state, priority, energy, scheduled_at, due_at, created_at, completed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.Title, in.Description, in.State, in.Priority, in.Energy,
		nullTime(in.ScheduledAt), nullTime(in.DueAt), mustTime(in.CreatedAt), nullTime(in.CompletedAt),
	)
	return err
}

func (r *SQLiteRepository) GetTask(ctx context.Context, id string) (Task, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, title, description, state, priority, energy, scheduled_at, due_at, created_at, completed_at
		FROM tasks WHERE id = ?`, id)
	task, err := scanTask(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Task{}, ErrNotFound
		}
		return Task{}, err
	}
	return task, nil
}

func (r *SQLiteRepository) UpdateTask(ctx context.Context, in Task) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, state = ?, priority = ?, energy = ?, scheduled_at = ?, due_at = ?, completed_at = ?
		WHERE id = ?`,
		in.Title, in.Description, in.State, in.Priority, in.Energy,
		nullTime(in.ScheduledAt), nullTime(in.DueAt), nullTime(in.CompletedAt), in.ID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) DeleteTask(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) ListTasks(ctx context.Context, filter TaskListFilter) ([]Task, error) {
	query := `SELECT id, title, description, state, priority, energy, scheduled_at, due_at, created_at, completed_at FROM tasks`
	args := make([]any, 0, 3)
	if filter.State != "" {
		query += ` WHERE state = ?`
		args = append(args, filter.State)
	}
	query += ` ORDER BY created_at DESC`
	query += applyPagination(&args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Task, 0)
	for rows.Next() {
		task, scanErr := scanTask(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, task)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateReminder(ctx context.Context, in Reminder) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO reminders (id, task_id, trigger_time, type, repeat_rule, last_fired_at, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.TaskID, mustTime(in.TriggerAt), in.Type, in.RepeatRule, nullTime(in.LastFired), boolInt(in.Enabled), mustTime(in.CreatedAt),
	)
	return err
}

func (r *SQLiteRepository) GetReminder(ctx context.Context, id string) (Reminder, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, task_id, trigger_time, type, repeat_rule, last_fired_at, enabled, created_at
		FROM reminders WHERE id = ?`, id)
	item, err := scanReminder(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Reminder{}, ErrNotFound
		}
		return Reminder{}, err
	}
	return item, nil
}

func (r *SQLiteRepository) UpdateReminder(ctx context.Context, in Reminder) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE reminders
		SET task_id = ?, trigger_time = ?, type = ?, repeat_rule = ?, last_fired_at = ?, enabled = ?
		WHERE id = ?`,
		in.TaskID, mustTime(in.TriggerAt), in.Type, in.RepeatRule, nullTime(in.LastFired), boolInt(in.Enabled), in.ID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) DeleteReminder(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM reminders WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) ListReminders(ctx context.Context, filter ReminderListFilter) ([]Reminder, error) {
	query := `SELECT id, task_id, trigger_time, type, repeat_rule, last_fired_at, enabled, created_at FROM reminders`
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if filter.TaskID != "" {
		clauses = append(clauses, "task_id = ?")
		args = append(args, filter.TaskID)
	}
	if filter.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, boolInt(*filter.Enabled))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY trigger_time ASC`
	query += applyPagination(&args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Reminder, 0)
	for rows.Next() {
		item, scanErr := scanReminder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateTag(ctx context.Context, in Tag) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tags (id, name, created_at)
		VALUES (?, ?, ?)`,
		in.ID, in.Name, mustTime(in.CreatedAt),
	)
	return err
}

func (r *SQLiteRepository) GetTag(ctx context.Context, id string) (Tag, error) {
	row := r.db.QueryRowContext(ctx, `SELECT id, name, created_at FROM tags WHERE id = ?`, id)
	item, err := scanTag(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Tag{}, ErrNotFound
		}
		return Tag{}, err
	}
	return item, nil
}

func (r *SQLiteRepository) UpdateTag(ctx context.Context, in Tag) error {
	res, err := r.db.ExecContext(ctx, `UPDATE tags SET name = ? WHERE id = ?`, in.Name, in.ID)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) DeleteTag(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tags WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) ListTags(ctx context.Context, filter TagListFilter) ([]Tag, error) {
	args := make([]any, 0, 2)
	query := `SELECT id, name, created_at FROM tags ORDER BY name ASC` + applyPagination(&args, filter.Limit, filter.Offset)
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]Tag, 0)
	for rows.Next() {
		item, scanErr := scanTag(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (r *SQLiteRepository) CreateRecurrence(ctx context.Context, in RecurrenceRule) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO recurrence_rules (id, task_id, rule_type, interval_value, timezone, start_at, next_occurrence_at, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.TaskID, in.RuleType, in.IntervalValue, in.Timezone, mustTime(in.StartAt), nullTime(in.NextAt), boolInt(in.Enabled), mustTime(in.CreatedAt),
	)
	return err
}

func (r *SQLiteRepository) GetRecurrence(ctx context.Context, id string) (RecurrenceRule, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, task_id, rule_type, interval_value, timezone, start_at, next_occurrence_at, enabled, created_at
		FROM recurrence_rules WHERE id = ?`, id)
	item, err := scanRecurrence(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return RecurrenceRule{}, ErrNotFound
		}
		return RecurrenceRule{}, err
	}
	return item, nil
}

func (r *SQLiteRepository) UpdateRecurrence(ctx context.Context, in RecurrenceRule) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE recurrence_rules
		SET task_id = ?, rule_type = ?, interval_value = ?, timezone = ?, start_at = ?, next_occurrence_at = ?, enabled = ?
		WHERE id = ?`,
		in.TaskID, in.RuleType, in.IntervalValue, in.Timezone, mustTime(in.StartAt), nullTime(in.NextAt), boolInt(in.Enabled), in.ID,
	)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) DeleteRecurrence(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM recurrence_rules WHERE id = ?`, id)
	if err != nil {
		return err
	}
	return checkRowsAffected(res)
}

func (r *SQLiteRepository) ListRecurrences(ctx context.Context, filter RecurrenceListFilter) ([]RecurrenceRule, error) {
	query := `SELECT id, task_id, rule_type, interval_value, timezone, start_at, next_occurrence_at, enabled, created_at FROM recurrence_rules`
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 4)
	if filter.TaskID != "" {
		clauses = append(clauses, "task_id = ?")
		args = append(args, filter.TaskID)
	}
	if filter.Enabled != nil {
		clauses = append(clauses, "enabled = ?")
		args = append(args, boolInt(*filter.Enabled))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += ` ORDER BY start_at ASC`
	query += applyPagination(&args, filter.Limit, filter.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]RecurrenceRule, 0)
	for rows.Next() {
		item, scanErr := scanRecurrence(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func nullTime(v *time.Time) any {
	if v == nil {
		return nil
	}
	return v.UTC().Format(sqliteTimeLayout)
}

func mustTime(v time.Time) string {
	return v.UTC().Format(sqliteTimeLayout)
}

func parseNullableTime(v sql.NullString) (*time.Time, error) {
	if !v.Valid || v.String == "" {
		return nil, nil
	}
	tm, err := time.Parse(sqliteTimeLayout, v.String)
	if err != nil {
		return nil, err
	}
	return &tm, nil
}

func parseRequiredTime(v string) (time.Time, error) {
	return time.Parse(sqliteTimeLayout, v)
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}

func applyPagination(args *[]any, limit, offset int) string {
	sql := ""
	if limit > 0 {
		sql += " LIMIT ?"
		*args = append(*args, limit)
	}
	if offset > 0 {
		sql += " OFFSET ?"
		*args = append(*args, offset)
	}
	return sql
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(s scanner) (Task, error) {
	var out Task
	var scheduled sql.NullString
	var due sql.NullString
	var created string
	var completed sql.NullString
	if err := s.Scan(&out.ID, &out.Title, &out.Description, &out.State, &out.Priority, &out.Energy, &scheduled, &due, &created, &completed); err != nil {
		return Task{}, err
	}
	createdAt, err := parseRequiredTime(created)
	if err != nil {
		return Task{}, err
	}
	scheduledAt, err := parseNullableTime(scheduled)
	if err != nil {
		return Task{}, err
	}
	dueAt, err := parseNullableTime(due)
	if err != nil {
		return Task{}, err
	}
	completedAt, err := parseNullableTime(completed)
	if err != nil {
		return Task{}, err
	}
	out.CreatedAt = createdAt
	out.ScheduledAt = scheduledAt
	out.DueAt = dueAt
	out.CompletedAt = completedAt
	return out, nil
}

func scanReminder(s scanner) (Reminder, error) {
	var out Reminder
	var trigger string
	var fired sql.NullString
	var enabled int
	var created string
	if err := s.Scan(&out.ID, &out.TaskID, &trigger, &out.Type, &out.RepeatRule, &fired, &enabled, &created); err != nil {
		return Reminder{}, err
	}
	triggerAt, err := parseRequiredTime(trigger)
	if err != nil {
		return Reminder{}, err
	}
	lastFired, err := parseNullableTime(fired)
	if err != nil {
		return Reminder{}, err
	}
	createdAt, err := parseRequiredTime(created)
	if err != nil {
		return Reminder{}, err
	}
	out.TriggerAt = triggerAt
	out.LastFired = lastFired
	out.Enabled = enabled == 1
	out.CreatedAt = createdAt
	return out, nil
}

func scanTag(s scanner) (Tag, error) {
	var out Tag
	var created string
	if err := s.Scan(&out.ID, &out.Name, &created); err != nil {
		return Tag{}, err
	}
	createdAt, err := parseRequiredTime(created)
	if err != nil {
		return Tag{}, err
	}
	out.CreatedAt = createdAt
	return out, nil
}

func scanRecurrence(s scanner) (RecurrenceRule, error) {
	var out RecurrenceRule
	var start string
	var next sql.NullString
	var enabled int
	var created string
	if err := s.Scan(&out.ID, &out.TaskID, &out.RuleType, &out.IntervalValue, &out.Timezone, &start, &next, &enabled, &created); err != nil {
		return RecurrenceRule{}, err
	}
	startAt, err := parseRequiredTime(start)
	if err != nil {
		return RecurrenceRule{}, err
	}
	nextAt, err := parseNullableTime(next)
	if err != nil {
		return RecurrenceRule{}, err
	}
	createdAt, err := parseRequiredTime(created)
	if err != nil {
		return RecurrenceRule{}, err
	}
	out.StartAt = startAt
	out.NextAt = nextAt
	out.Enabled = enabled == 1
	out.CreatedAt = createdAt
	return out, nil
}

func checkRowsAffected(res sql.Result) error {
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrNotFound
	}
	return nil
}
