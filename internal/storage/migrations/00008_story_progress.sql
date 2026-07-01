-- +goose Up
CREATE TABLE story_progress (
    profile_id INTEGER NOT NULL REFERENCES profiles (id) ON DELETE CASCADE,
    chapter_id TEXT    NOT NULL,
    beat_index INTEGER NOT NULL DEFAULT 0,
    completed  INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (profile_id, chapter_id)
);

-- +goose Down
DROP TABLE story_progress;
