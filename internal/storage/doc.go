// Package storage persists learner profiles and progress. It defines the Storage
// interface and a SQLite-backed implementation (modernc.org/sqlite, no CGO) with
// goose-managed schema migrations.
package storage
