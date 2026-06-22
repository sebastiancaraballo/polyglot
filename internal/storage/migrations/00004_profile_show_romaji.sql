-- +goose Up
ALTER TABLE profiles ADD COLUMN show_romaji INTEGER NOT NULL DEFAULT 1;

-- +goose Down
ALTER TABLE profiles DROP COLUMN show_romaji;
