-- +goose Up
CREATE TABLE pattern_progress (
    profile_id INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    pattern_id TEXT    NOT NULL,
    slot       TEXT    NOT NULL,
    streak     INTEGER NOT NULL DEFAULT 0,
    attempts   INTEGER NOT NULL DEFAULT 0,
    mastered   INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (profile_id, pattern_id, slot)
);

-- +goose Down
DROP TABLE pattern_progress;
