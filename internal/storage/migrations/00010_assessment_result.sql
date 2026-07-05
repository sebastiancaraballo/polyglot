-- +goose Up
CREATE TABLE assessment_result (
    profile_id   INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    level        TEXT    NOT NULL,
    passed       INTEGER NOT NULL DEFAULT 0,
    best_correct INTEGER NOT NULL DEFAULT 0,
    total        INTEGER NOT NULL DEFAULT 0,
    taken_at     TEXT,
    PRIMARY KEY (profile_id, level)
);

-- +goose Down
DROP TABLE assessment_result;
