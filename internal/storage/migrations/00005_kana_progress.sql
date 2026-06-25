-- +goose Up
CREATE TABLE kana_progress (
    profile_id INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    char       TEXT    NOT NULL,
    streak     INTEGER NOT NULL DEFAULT 0,
    attempts   INTEGER NOT NULL DEFAULT 0,
    mastered   INTEGER NOT NULL DEFAULT 0,
    best_ms    INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (profile_id, char)
);

-- +goose Down
DROP TABLE kana_progress;
