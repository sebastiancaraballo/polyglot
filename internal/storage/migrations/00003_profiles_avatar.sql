-- +goose Up
ALTER TABLE profiles ADD COLUMN avatar TEXT NOT NULL DEFAULT '';

CREATE TABLE app_meta (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- +goose Down
DROP TABLE app_meta;
ALTER TABLE profiles DROP COLUMN avatar;
