# taskd

`taskd` is a keyboard-first terminal productivity app built with Bubble Tea.

## Current Features

- Multi-view TUI core: Today, Inbox, Calendar/Agenda, Focus
- Reminder scheduler engine with type-specific behavior
- Recurrence rule engine with preview support
- Command palette (`/`) with `add`, `snooze`, `show`, `reschedule`
- Contextual help and keybinding panel
- In-TUI notifications + optional desktop notifications
- Productivity signals: temporal debt + energy-aware suggestions

## Run

```bash
go run ./cmd/taskd
```

## Runtime Config (Environment)

- `TASKD_DESKTOP_NOTIFICATIONS` (`true|false|1|0`)
- `TASKD_FOCUS_WORK_MINUTES` (default `25`)
- `TASKD_FOCUS_BREAK_MINUTES` (default `5`)
- `TASKD_PRODUCTIVITY_AVAILABLE_MINUTES` (default `60`)
- `TASKD_SCHEDULER_BUFFER` (default `64`)

See `taskd.example.env` for examples.

## Validation

```bash
go test ./...
go test -race ./...
```
