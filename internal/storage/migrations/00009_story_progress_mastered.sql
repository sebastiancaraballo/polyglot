-- +goose Up
ALTER TABLE story_progress ADD COLUMN mastered INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE story_progress DROP COLUMN mastered;
