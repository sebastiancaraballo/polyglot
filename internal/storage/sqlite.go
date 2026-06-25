package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite" // registers the "sqlite" driver (pure Go, no CGO)

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// timeLayout is the canonical textual representation for stored timestamps.
const timeLayout = time.RFC3339Nano

// SQLiteStore is a Storage backed by a local SQLite database. It is safe for use
// by a single running application instance.
type SQLiteStore struct {
	db *sql.DB
}

// compile-time assertion that SQLiteStore satisfies Storage.
var _ Storage = (*SQLiteStore)(nil)

// Open opens (creating if needed) the SQLite database at path, configures it, and
// applies all pending migrations.
func Open(path string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// A single connection avoids "database is locked" errors with SQLite while
	// keeping behavior simple for a single-user desktop app.
	db.SetMaxOpenConns(1)

	for _, pragma := range []string{
		"PRAGMA journal_mode = WAL;",
		"PRAGMA foreign_keys = ON;",
		"PRAGMA busy_timeout = 5000;",
	} {
		if _, err := db.Exec(pragma); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("apply %q: %w", pragma, err)
		}
	}

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

// migrate applies the embedded goose migrations to db.
func migrate(db *sql.DB) error {
	goose.SetBaseFS(migrationsFS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Close releases the underlying database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// CreateProfile inserts a new profile along with an empty stats row.
func (s *SQLiteStore) CreateProfile(ctx context.Context, name string) (model.Profile, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.Profile{}, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	now := time.Now().UTC()
	res, err := tx.ExecContext(ctx,
		`INSERT INTO profiles (name, onboarded, created_at) VALUES (?, 0, ?)`,
		name, now.Format(timeLayout),
	)
	if err != nil {
		return model.Profile{}, fmt.Errorf("insert profile: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return model.Profile{}, fmt.Errorf("read profile id: %w", err)
	}
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO stats (profile_id, streak, best_streak, last_studied_at, xp) VALUES (?, 0, 0, NULL, 0)`,
		id,
	); err != nil {
		return model.Profile{}, fmt.Errorf("init stats: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return model.Profile{}, fmt.Errorf("commit: %w", err)
	}

	return model.Profile{ID: id, Name: name, Onboarded: false, ShowRomaji: true, CreatedAt: now}, nil
}

// DeleteProfile removes a profile and, via ON DELETE CASCADE, its stats and card
// states.
func (s *SQLiteStore) DeleteProfile(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM profiles WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete profile: %w", err)
	}
	return requireAffected(res)
}

// ListProfiles returns all profiles ordered by creation time then id.
func (s *SQLiteStore) ListProfiles(ctx context.Context) ([]model.Profile, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, onboarded, show_romaji, created_at FROM profiles ORDER BY created_at, id`,
	)
	if err != nil {
		return nil, fmt.Errorf("query profiles: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var profiles []model.Profile
	for rows.Next() {
		p, err := scanProfile(rows)
		if err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate profiles: %w", err)
	}
	return profiles, nil
}

// GetProfile returns the profile with the given id, or ErrNotFound.
func (s *SQLiteStore) GetProfile(ctx context.Context, id int64) (model.Profile, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, name, onboarded, show_romaji, created_at FROM profiles WHERE id = ?`, id,
	)
	p, err := scanProfile(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Profile{}, ErrNotFound
	}
	if err != nil {
		return model.Profile{}, err
	}
	return p, nil
}

// SetOnboarded marks a profile as onboarded.
func (s *SQLiteStore) SetOnboarded(ctx context.Context, profileID int64) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE profiles SET onboarded = 1 WHERE id = ?`, profileID,
	)
	if err != nil {
		return fmt.Errorf("update onboarded: %w", err)
	}
	return requireAffected(res)
}

// SetShowRomaji sets whether a profile displays romaji alongside Japanese.
func (s *SQLiteStore) SetShowRomaji(ctx context.Context, profileID int64, enabled bool) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE profiles SET show_romaji = ? WHERE id = ?`, enabled, profileID,
	)
	if err != nil {
		return fmt.Errorf("update show_romaji: %w", err)
	}
	return requireAffected(res)
}

// GetCardState returns the scheduling state for a card, or ErrNotFound.
func (s *SQLiteStore) GetCardState(ctx context.Context, profileID int64, cardID string) (model.CardState, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT card_id, interval, ease, reps, lapses, due_at, last_reviewed_at
		   FROM card_states WHERE profile_id = ? AND card_id = ?`,
		profileID, cardID,
	)

	var (
		st             model.CardState
		dueAt          string
		lastReviewedAt string
	)
	err := row.Scan(&st.CardID, &st.Interval, &st.Ease, &st.Reps, &st.Lapses, &dueAt, &lastReviewedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return model.CardState{}, ErrNotFound
	}
	if err != nil {
		return model.CardState{}, fmt.Errorf("scan card state: %w", err)
	}
	if st.DueAt, err = parseTime(dueAt); err != nil {
		return model.CardState{}, err
	}
	if st.LastReviewedAt, err = parseTime(lastReviewedAt); err != nil {
		return model.CardState{}, err
	}
	return st, nil
}

// SaveCardState inserts or updates the scheduling state for a card.
func (s *SQLiteStore) SaveCardState(ctx context.Context, profileID int64, state model.CardState) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO card_states
		   (profile_id, card_id, interval, ease, reps, lapses, due_at, last_reviewed_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT (profile_id, card_id) DO UPDATE SET
		   interval = excluded.interval,
		   ease = excluded.ease,
		   reps = excluded.reps,
		   lapses = excluded.lapses,
		   due_at = excluded.due_at,
		   last_reviewed_at = excluded.last_reviewed_at`,
		profileID, state.CardID, state.Interval, state.Ease, state.Reps, state.Lapses,
		state.DueAt.UTC().Format(timeLayout), state.LastReviewedAt.UTC().Format(timeLayout),
	)
	if err != nil {
		return fmt.Errorf("save card state: %w", err)
	}
	return nil
}

// GetKanaProgress returns the profile's kana automaticity progress, keyed by
// kana character. Kana the profile has never practiced are absent from the map.
func (s *SQLiteStore) GetKanaProgress(ctx context.Context, profileID int64) (map[string]model.KanaProgress, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT char, streak, attempts, mastered, best_ms
		   FROM kana_progress WHERE profile_id = ?`,
		profileID,
	)
	if err != nil {
		return nil, fmt.Errorf("query kana progress: %w", err)
	}
	defer func() { _ = rows.Close() }()

	progress := make(map[string]model.KanaProgress)
	for rows.Next() {
		var p model.KanaProgress
		if err := rows.Scan(&p.Char, &p.Streak, &p.Attempts, &p.Mastered, &p.BestMs); err != nil {
			return nil, fmt.Errorf("scan kana progress: %w", err)
		}
		progress[p.Char] = p
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate kana progress: %w", err)
	}
	return progress, nil
}

// SaveKanaProgress inserts or updates the automaticity progress for one kana.
func (s *SQLiteStore) SaveKanaProgress(ctx context.Context, profileID int64, p model.KanaProgress) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO kana_progress (profile_id, char, streak, attempts, mastered, best_ms)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT (profile_id, char) DO UPDATE SET
		   streak = excluded.streak,
		   attempts = excluded.attempts,
		   mastered = excluded.mastered,
		   best_ms = excluded.best_ms`,
		profileID, p.Char, p.Streak, p.Attempts, p.Mastered, p.BestMs,
	)
	if err != nil {
		return fmt.Errorf("save kana progress: %w", err)
	}
	return nil
}

// GetStats returns the aggregate stats for a profile, or ErrNotFound.
func (s *SQLiteStore) GetStats(ctx context.Context, profileID int64) (model.Stats, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT streak, best_streak, last_studied_at, xp FROM stats WHERE profile_id = ?`,
		profileID,
	)

	var (
		stats         model.Stats
		lastStudiedAt sql.NullString
	)
	err := row.Scan(&stats.Streak, &stats.BestStreak, &lastStudiedAt, &stats.XP)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Stats{}, ErrNotFound
	}
	if err != nil {
		return model.Stats{}, fmt.Errorf("scan stats: %w", err)
	}
	if lastStudiedAt.Valid {
		if stats.LastStudiedAt, err = parseTime(lastStudiedAt.String); err != nil {
			return model.Stats{}, err
		}
	}
	return stats, nil
}

// SaveStats replaces the aggregate stats for a profile.
func (s *SQLiteStore) SaveStats(ctx context.Context, profileID int64, stats model.Stats) error {
	var lastStudiedAt any
	if !stats.LastStudiedAt.IsZero() {
		lastStudiedAt = stats.LastStudiedAt.UTC().Format(timeLayout)
	}
	res, err := s.db.ExecContext(ctx,
		`UPDATE stats SET streak = ?, best_streak = ?, last_studied_at = ?, xp = ? WHERE profile_id = ?`,
		stats.Streak, stats.BestStreak, lastStudiedAt, stats.XP, profileID,
	)
	if err != nil {
		return fmt.Errorf("update stats: %w", err)
	}
	return requireAffected(res)
}

// AddXP atomically increments a profile's cumulative experience points.
func (s *SQLiteStore) AddXP(ctx context.Context, profileID int64, amount int) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE stats SET xp = xp + ? WHERE profile_id = ?`,
		amount, profileID,
	)
	if err != nil {
		return fmt.Errorf("add xp: %w", err)
	}
	return requireAffected(res)
}

// CountLearnedCards returns the number of cards the profile has reviewed at
// least once successfully (reps > 0).
func (s *SQLiteStore) CountLearnedCards(ctx context.Context, profileID int64) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM card_states WHERE profile_id = ? AND reps > 0`, profileID,
	).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count learned cards: %w", err)
	}
	return n, nil
}

// activeProfileKey is the app_meta key holding the active profile's id.
const activeProfileKey = "active_profile_id"

// GetActiveProfileID returns the persisted active profile id. ok is false when no
// active profile has been set yet.
func (s *SQLiteStore) GetActiveProfileID(ctx context.Context) (id int64, ok bool, err error) {
	var value string
	row := s.db.QueryRowContext(ctx, `SELECT value FROM app_meta WHERE key = ?`, activeProfileKey)
	if err := row.Scan(&value); errors.Is(err, sql.ErrNoRows) {
		return 0, false, nil
	} else if err != nil {
		return 0, false, fmt.Errorf("scan active profile: %w", err)
	}
	id, err = strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, false, fmt.Errorf("parse active profile id %q: %w", value, err)
	}
	return id, true, nil
}

// SetActiveProfileID persists which profile is active.
func (s *SQLiteStore) SetActiveProfileID(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO app_meta (key, value) VALUES (?, ?)
		 ON CONFLICT (key) DO UPDATE SET value = excluded.value`,
		activeProfileKey, strconv.FormatInt(id, 10),
	)
	if err != nil {
		return fmt.Errorf("set active profile: %w", err)
	}
	return nil
}

// rowScanner is satisfied by both *sql.Row and *sql.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanProfile(s rowScanner) (model.Profile, error) {
	var (
		p         model.Profile
		createdAt string
	)
	if err := s.Scan(&p.ID, &p.Name, &p.Onboarded, &p.ShowRomaji, &createdAt); err != nil {
		return model.Profile{}, err
	}
	parsed, err := parseTime(createdAt)
	if err != nil {
		return model.Profile{}, err
	}
	p.CreatedAt = parsed
	return p, nil
}

func parseTime(s string) (time.Time, error) {
	t, err := time.Parse(timeLayout, s)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse timestamp %q: %w", s, err)
	}
	return t, nil
}

// requireAffected returns ErrNotFound when an UPDATE matched no rows.
func requireAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
