# Storage Migrations

This folder stores ordered SQL migrations for `taskd` SQLite storage.

## Current migration set

- `0001_init.up.sql`: creates the baseline schema.
- `0001_init.down.sql`: drops the baseline schema.

## Baseline schema coverage

- `tasks`
- `reminders`
- `tags`
- `task_tags`
- `recurrence_rules`
- `scheduler_state`
