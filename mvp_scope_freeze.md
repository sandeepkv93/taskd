# taskd - MVP Scope Freeze (E01)

Date: 2026-02-09  
Board item: `E01` - Freeze MVP + non-goals from requirements

## Outcome

MVP scope is frozen for initial delivery. The implementation phase proceeds with `E02` next.

## In Scope (MVP)

- Today view (primary screen) with clear sections:
  - Scheduled
  - Anytime today
  - Overdue
- Inbox view with fast capture (title-first flow) and later triage.
- Calendar/Agenda view with day/week/month navigation.
- Focus mode with single-task timer and progress display.
- Reminder system with types:
  - Hard
  - Soft
  - Nagging
  - Contextual
- Recurrence basics:
  - Every weekday
  - Every N days/weeks
  - Last day of month
  - After completion
- Command palette core (`/`) with initial commands:
  - `add`
  - `snooze`
  - `show`
  - `reschedule`
- Keyboard-first operation with contextual help.
- Local-first persistence using SQLite and migrations.

## Out of Scope (for MVP)

- Team collaboration.
- Cloud sync.
- Mobile or web UI.
- Feature parity with large task managers.
- Webhook delivery channel.
- LLM-powered command parsing and local LLM assistance.
- External imports (GitHub/Jira).
- Time-tracking analytics.

## Acceptance Checklist for E01

- [x] MVP features are explicitly listed and bounded.
- [x] Non-goals/stretch items are explicitly deferred.
- [x] Scope aligns with:
  - `taskd/tui_todo_reminders_calendar_requirements.md`
  - `taskd/detailed_step_by_step_plan.md`
  - `taskd/execution_board.md`
- [x] Next executable board item is `E02`.

## Notes for E02

- `E02` will create the project skeleton without adding feature logic.
- Keep architecture aligned with:
  - `cmd/`
  - `internal/model`
  - `internal/views`
  - `internal/update`
  - `internal/scheduler`
  - `internal/storage`
  - `internal/commands`
