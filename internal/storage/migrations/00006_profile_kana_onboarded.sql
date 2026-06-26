-- +goose Up
ALTER TABLE profiles ADD COLUMN kana_onboarded INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE profiles DROP COLUMN kana_onboarded;
