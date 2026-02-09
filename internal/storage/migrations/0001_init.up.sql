PRAGMA foreign_keys = ON;

CREATE TABLE tasks (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    state TEXT NOT NULL CHECK (state IN ('Inbox', 'Planned', 'Done', 'Snoozed')),
    priority TEXT NOT NULL CHECK (priority IN ('Low', 'Medium', 'High', 'Critical')),
    energy TEXT NOT NULL CHECK (energy IN ('Deep', 'Light', 'Social', 'Low')),
    scheduled_at TEXT,
    due_at TEXT,
    created_at TEXT NOT NULL,
    completed_at TEXT
);

CREATE TABLE reminders (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    trigger_time TEXT NOT NULL,
    type TEXT NOT NULL CHECK (type IN ('Hard', 'Soft', 'Nagging', 'Contextual')),
    repeat_rule TEXT NOT NULL DEFAULT '',
    last_fired_at TEXT,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    created_at TEXT NOT NULL
);

CREATE TABLE task_tags (
    task_id TEXT NOT NULL,
    tag_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (task_id, tag_id),
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE,
    FOREIGN KEY (tag_id) REFERENCES tags (id) ON DELETE CASCADE
);

CREATE TABLE recurrence_rules (
    id TEXT PRIMARY KEY,
    task_id TEXT NOT NULL,
    rule_type TEXT NOT NULL CHECK (rule_type IN (
        'weekday',
        'every_n_days',
        'every_n_weeks',
        'last_day_of_month',
        'after_completion'
    )),
    interval_value INTEGER NOT NULL DEFAULT 1,
    timezone TEXT NOT NULL DEFAULT 'UTC',
    start_at TEXT NOT NULL,
    next_occurrence_at TEXT,
    enabled INTEGER NOT NULL DEFAULT 1 CHECK (enabled IN (0, 1)),
    created_at TEXT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks (id) ON DELETE CASCADE
);

CREATE TABLE scheduler_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    last_tick_at TEXT,
    checkpoint_cursor TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL
);

INSERT INTO scheduler_state (id, last_tick_at, checkpoint_cursor, updated_at)
VALUES (1, NULL, '', strftime('%Y-%m-%dT%H:%M:%fZ', 'now'));
