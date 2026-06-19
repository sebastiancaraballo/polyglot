package storage

import "embed"

// migrationsFS holds the goose SQL migrations embedded into the binary.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS
