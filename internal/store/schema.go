package store

const currentSchemaVersion = 1

const createSchemaVersionTable = `
CREATE TABLE IF NOT EXISTS schema_version (
    version     INTEGER NOT NULL,
    applied_at  TEXT NOT NULL
);`

// migrations holds ordered SQL DDL. Index 0 = version 1.
var migrations = []string{
	// version 1
	`
CREATE TABLE IF NOT EXISTS repos (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    owner       TEXT NOT NULL,
    name        TEXT NOT NULL,
    full_name   TEXT NOT NULL UNIQUE,
    packages    TEXT NOT NULL DEFAULT '{}',
    added_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_repos_full_name ON repos(full_name);

CREATE TABLE IF NOT EXISTS snapshots (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id     INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    stars       INTEGER NOT NULL DEFAULT 0,
    forks       INTEGER NOT NULL DEFAULT 0,
    open_prs    INTEGER NOT NULL DEFAULT 0,
    taken_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_snapshots_repo_id  ON snapshots(repo_id);
CREATE INDEX IF NOT EXISTS idx_snapshots_taken_at ON snapshots(taken_at);

CREATE TABLE IF NOT EXISTS events (
    id          TEXT PRIMARY KEY,
    repo_id     INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    kind        TEXT NOT NULL,
    title       TEXT NOT NULL DEFAULT '',
    body        TEXT NOT NULL DEFAULT '',
    url         TEXT NOT NULL DEFAULT '',
    actor       TEXT NOT NULL DEFAULT '',
    meta        TEXT NOT NULL DEFAULT '{}',
    occurred_at TEXT NOT NULL,
    created_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_repo_id     ON events(repo_id);
CREATE INDEX IF NOT EXISTS idx_events_occurred_at ON events(occurred_at);
CREATE INDEX IF NOT EXISTS idx_events_kind        ON events(kind);

CREATE TABLE IF NOT EXISTS milestones (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id       INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    kind          TEXT NOT NULL,
    threshold     INTEGER NOT NULL,
    actual_value  INTEGER NOT NULL DEFAULT 0,
    celebrated_at TEXT NOT NULL,
    UNIQUE(repo_id, kind, threshold)
);
CREATE INDEX IF NOT EXISTS idx_milestones_repo_id ON milestones(repo_id);
CREATE INDEX IF NOT EXISTS idx_milestones_kind    ON milestones(kind);

CREATE TABLE IF NOT EXISTS sessions (
    id      INTEGER PRIMARY KEY AUTOINCREMENT,
    ran_at  TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS streaks (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    repo_id      INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    metric       TEXT NOT NULL,
    current_days INTEGER NOT NULL DEFAULT 0,
    best_days    INTEGER NOT NULL DEFAULT 0,
    last_active  TEXT NOT NULL,
    UNIQUE(repo_id, metric)
);
CREATE INDEX IF NOT EXISTS idx_streaks_repo_id ON streaks(repo_id);

CREATE TABLE IF NOT EXISTS saved_moments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    event_id  TEXT NOT NULL,
    repo_id   INTEGER NOT NULL REFERENCES repos(id) ON DELETE CASCADE,
    kind      TEXT NOT NULL,
    title     TEXT NOT NULL DEFAULT '',
    body      TEXT NOT NULL DEFAULT '',
    actor     TEXT NOT NULL DEFAULT '',
    url       TEXT NOT NULL DEFAULT '',
    saved_at  TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_saved_moments_repo_id  ON saved_moments(repo_id);
CREATE INDEX IF NOT EXISTS idx_saved_moments_saved_at ON saved_moments(saved_at);
`,
}
