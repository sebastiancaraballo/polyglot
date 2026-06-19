-- +goose Up
CREATE TABLE profiles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    onboarded  INTEGER NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL
);

CREATE TABLE card_states (
    profile_id       INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    card_id          TEXT    NOT NULL,
    interval         INTEGER NOT NULL DEFAULT 0,
    ease             REAL    NOT NULL DEFAULT 2.5,
    reps             INTEGER NOT NULL DEFAULT 0,
    lapses           INTEGER NOT NULL DEFAULT 0,
    due_at           TEXT    NOT NULL,
    last_reviewed_at TEXT    NOT NULL,
    PRIMARY KEY (profile_id, card_id)
);

CREATE INDEX idx_card_states_due ON card_states (profile_id, due_at);

CREATE TABLE stats (
    profile_id      INTEGER PRIMARY KEY REFERENCES profiles (id) ON DELETE CASCADE,
    streak          INTEGER NOT NULL DEFAULT 0,
    best_streak     INTEGER NOT NULL DEFAULT 0,
    last_studied_at TEXT
);

-- +goose Down
DROP TABLE stats;
DROP TABLE card_states;
DROP TABLE profiles;
