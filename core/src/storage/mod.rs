//! Persists learner profiles and their progress in a local SQLite database.
//!
//! Port of the Go `internal/storage` package. `modernc.org/sqlite` (pure Go)
//! becomes `rusqlite` (bundled SQLite); `goose` migrations become a small
//! `user_version`-tracked runner (see [`migrations`]). Timestamps are stored as
//! RFC 3339 text; a year-1 sentinel round-trips a `None` time for the NOT NULL
//! card-state columns, matching the Go zero-`time.Time` behavior.

mod migrations;

use std::fmt;
use std::path::{Path, PathBuf};

use chrono::{DateTime, Datelike, SecondsFormat, TimeZone, Utc};
use rusqlite::{params, Connection, OptionalExtension};

use crate::model::{
    AssessmentResult, CardState, Jlpt, KanaProgress, PatternProgress, Profile, Stats, StoryProgress,
};

/// An error from the storage layer.
#[derive(Debug)]
pub enum StorageError {
    /// A requested record does not exist.
    NotFound,
    /// An underlying SQLite error.
    Sqlite(rusqlite::Error),
    /// A stored timestamp could not be parsed.
    Time(String),
    /// A filesystem error resolving or preparing the database path.
    Io(std::io::Error),
}

impl fmt::Display for StorageError {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        match self {
            StorageError::NotFound => write!(f, "storage: not found"),
            StorageError::Sqlite(e) => write!(f, "storage: {e}"),
            StorageError::Time(m) => write!(f, "storage: {m}"),
            StorageError::Io(e) => write!(f, "storage: {e}"),
        }
    }
}

impl std::error::Error for StorageError {}

impl From<rusqlite::Error> for StorageError {
    fn from(e: rusqlite::Error) -> Self {
        StorageError::Sqlite(e)
    }
}

/// The app_meta key holding the active profile's id.
const ACTIVE_PROFILE_KEY: &str = "active_profile_id";

/// A [`Storage`]-equivalent backed by a local SQLite database. Safe for use by a
/// single running application instance.
pub struct SqliteStore {
    conn: Connection,
}

impl SqliteStore {
    /// Opens (creating if needed) the database at `path`, configures it, and
    /// applies all pending migrations.
    pub fn open(path: impl AsRef<Path>) -> Result<Self, StorageError> {
        let conn = Connection::open(path)?;
        conn.execute_batch(
            "PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON; PRAGMA busy_timeout = 5000;",
        )?;
        migrations::migrate(&conn)?;
        Ok(SqliteStore { conn })
    }

    /// Opens an in-memory database (used by tests).
    pub fn open_in_memory() -> Result<Self, StorageError> {
        let conn = Connection::open_in_memory()?;
        conn.execute_batch("PRAGMA foreign_keys = ON;")?;
        migrations::migrate(&conn)?;
        Ok(SqliteStore { conn })
    }

    // --- Profiles ---------------------------------------------------------

    /// Creates a new profile (with an empty stats row) and returns it.
    pub fn create_profile(&self, name: &str) -> Result<Profile, StorageError> {
        let now = Utc::now();
        let tx = self.conn.unchecked_transaction()?;
        tx.execute(
            "INSERT INTO profiles (name, onboarded, created_at) VALUES (?1, 0, ?2)",
            params![name, fmt_time(Some(now))],
        )?;
        let id = tx.last_insert_rowid();
        tx.execute(
            "INSERT INTO stats (profile_id, streak, best_streak, last_studied_at, xp) \
             VALUES (?1, 0, 0, NULL, 0)",
            params![id],
        )?;
        tx.commit()?;
        Ok(Profile {
            id,
            name: name.to_string(),
            onboarded: false,
            show_romaji: true,
            kana_onboarded: false,
            created_at: Some(now),
        })
    }

    /// Removes a profile and, via ON DELETE CASCADE, its stats and progress.
    pub fn delete_profile(&self, id: i64) -> Result<(), StorageError> {
        let n = self
            .conn
            .execute("DELETE FROM profiles WHERE id = ?1", params![id])?;
        require_affected(n)
    }

    /// Returns all profiles ordered by creation time then id.
    pub fn list_profiles(&self) -> Result<Vec<Profile>, StorageError> {
        let mut stmt = self.conn.prepare(
            "SELECT id, name, onboarded, show_romaji, kana_onboarded, created_at \
             FROM profiles ORDER BY created_at, id",
        )?;
        let rows = stmt.query_map([], profile_row)?;
        let mut out = Vec::new();
        for r in rows {
            out.push(build_profile(r?)?);
        }
        Ok(out)
    }

    /// Returns the profile with the given id, or [`StorageError::NotFound`].
    pub fn get_profile(&self, id: i64) -> Result<Profile, StorageError> {
        let raw = self
            .conn
            .query_row(
                "SELECT id, name, onboarded, show_romaji, kana_onboarded, created_at \
                 FROM profiles WHERE id = ?1",
                params![id],
                profile_row,
            )
            .optional()?;
        build_profile(raw.ok_or(StorageError::NotFound)?)
    }

    /// Marks a profile as having completed onboarding.
    pub fn set_onboarded(&self, profile_id: i64) -> Result<(), StorageError> {
        let n = self.conn.execute(
            "UPDATE profiles SET onboarded = 1 WHERE id = ?1",
            params![profile_id],
        )?;
        require_affected(n)
    }

    /// Sets whether a profile displays romaji alongside Japanese.
    pub fn set_show_romaji(&self, profile_id: i64, enabled: bool) -> Result<(), StorageError> {
        let n = self.conn.execute(
            "UPDATE profiles SET show_romaji = ?1 WHERE id = ?2",
            params![enabled, profile_id],
        )?;
        require_affected(n)
    }

    /// Marks a profile as having seen the kana trainer intro.
    pub fn set_kana_onboarded(&self, profile_id: i64) -> Result<(), StorageError> {
        let n = self.conn.execute(
            "UPDATE profiles SET kana_onboarded = 1 WHERE id = ?1",
            params![profile_id],
        )?;
        require_affected(n)
    }

    // --- Active profile ---------------------------------------------------

    /// Returns the persisted active profile id, or `None` when none is set.
    pub fn active_profile_id(&self) -> Result<Option<i64>, StorageError> {
        let value: Option<String> = self
            .conn
            .query_row(
                "SELECT value FROM app_meta WHERE key = ?1",
                params![ACTIVE_PROFILE_KEY],
                |row| row.get(0),
            )
            .optional()?;
        match value {
            None => Ok(None),
            Some(v) => v
                .parse::<i64>()
                .map(Some)
                .map_err(|e| StorageError::Time(format!("parse active profile id {v:?}: {e}"))),
        }
    }

    /// Persists which profile is active.
    pub fn set_active_profile_id(&self, id: i64) -> Result<(), StorageError> {
        self.conn.execute(
            "INSERT INTO app_meta (key, value) VALUES (?1, ?2) \
             ON CONFLICT (key) DO UPDATE SET value = excluded.value",
            params![ACTIVE_PROFILE_KEY, id.to_string()],
        )?;
        Ok(())
    }

    // --- Card states ------------------------------------------------------

    /// Returns the scheduling state for a card, or [`StorageError::NotFound`].
    pub fn get_card_state(
        &self,
        profile_id: i64,
        card_id: &str,
    ) -> Result<CardState, StorageError> {
        let raw = self
            .conn
            .query_row(
                "SELECT card_id, interval, ease, reps, lapses, due_at, last_reviewed_at \
                 FROM card_states WHERE profile_id = ?1 AND card_id = ?2",
                params![profile_id, card_id],
                card_state_row,
            )
            .optional()?;
        build_card_state(raw.ok_or(StorageError::NotFound)?)
    }

    /// Inserts or updates the scheduling state for a card.
    pub fn save_card_state(&self, profile_id: i64, state: &CardState) -> Result<(), StorageError> {
        self.conn.execute(
            "INSERT INTO card_states \
               (profile_id, card_id, interval, ease, reps, lapses, due_at, last_reviewed_at) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6, ?7, ?8) \
             ON CONFLICT (profile_id, card_id) DO UPDATE SET \
               interval = excluded.interval, ease = excluded.ease, reps = excluded.reps, \
               lapses = excluded.lapses, due_at = excluded.due_at, \
               last_reviewed_at = excluded.last_reviewed_at",
            params![
                profile_id,
                state.card_id,
                state.interval,
                state.ease,
                state.reps,
                state.lapses,
                fmt_time(state.due_at),
                fmt_time(state.last_reviewed_at),
            ],
        )?;
        Ok(())
    }

    /// Returns every card-scheduling state the profile has, keyed by card ID.
    pub fn get_card_states(
        &self,
        profile_id: i64,
    ) -> Result<std::collections::HashMap<String, CardState>, StorageError> {
        let mut stmt = self.conn.prepare(
            "SELECT card_id, interval, ease, reps, lapses, due_at, last_reviewed_at \
             FROM card_states WHERE profile_id = ?1",
        )?;
        let rows = stmt.query_map(params![profile_id], card_state_row)?;
        let mut map = std::collections::HashMap::new();
        for r in rows {
            let st = build_card_state(r?)?;
            map.insert(st.card_id.clone(), st);
        }
        Ok(map)
    }

    // --- Kana progress ----------------------------------------------------

    /// Returns the profile's kana automaticity progress, keyed by character.
    pub fn get_kana_progress(
        &self,
        profile_id: i64,
    ) -> Result<std::collections::HashMap<String, KanaProgress>, StorageError> {
        let mut stmt = self.conn.prepare(
            "SELECT char, streak, attempts, mastered, best_ms FROM kana_progress WHERE profile_id = ?1",
        )?;
        let rows = stmt.query_map(params![profile_id], |row| {
            Ok(KanaProgress {
                char: row.get(0)?,
                streak: row.get(1)?,
                attempts: row.get(2)?,
                mastered: row.get(3)?,
                best_ms: row.get(4)?,
            })
        })?;
        let mut map = std::collections::HashMap::new();
        for r in rows {
            let p = r?;
            map.insert(p.char.clone(), p);
        }
        Ok(map)
    }

    /// Inserts or updates the automaticity progress for one kana.
    pub fn save_kana_progress(
        &self,
        profile_id: i64,
        p: &KanaProgress,
    ) -> Result<(), StorageError> {
        self.conn.execute(
            "INSERT INTO kana_progress (profile_id, char, streak, attempts, mastered, best_ms) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6) \
             ON CONFLICT (profile_id, char) DO UPDATE SET \
               streak = excluded.streak, attempts = excluded.attempts, \
               mastered = excluded.mastered, best_ms = excluded.best_ms",
            params![profile_id, p.char, p.streak, p.attempts, p.mastered, p.best_ms],
        )?;
        Ok(())
    }

    // --- Pattern progress -------------------------------------------------

    /// Returns the profile's grammar-pattern drill progress, keyed by
    /// `"<pattern_id>:<slot>"`.
    pub fn get_pattern_progress(
        &self,
        profile_id: i64,
    ) -> Result<std::collections::HashMap<String, PatternProgress>, StorageError> {
        let mut stmt = self.conn.prepare(
            "SELECT pattern_id, slot, streak, attempts, mastered FROM pattern_progress WHERE profile_id = ?1",
        )?;
        let rows = stmt.query_map(params![profile_id], |row| {
            Ok(PatternProgress {
                pattern_id: row.get(0)?,
                slot: row.get(1)?,
                streak: row.get(2)?,
                attempts: row.get(3)?,
                mastered: row.get(4)?,
            })
        })?;
        let mut map = std::collections::HashMap::new();
        for r in rows {
            let p = r?;
            map.insert(format!("{}:{}", p.pattern_id, p.slot), p);
        }
        Ok(map)
    }

    /// Inserts or updates progress for one pattern slot.
    pub fn save_pattern_progress(
        &self,
        profile_id: i64,
        p: &PatternProgress,
    ) -> Result<(), StorageError> {
        self.conn.execute(
            "INSERT INTO pattern_progress (profile_id, pattern_id, slot, streak, attempts, mastered) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6) \
             ON CONFLICT (profile_id, pattern_id, slot) DO UPDATE SET \
               streak = excluded.streak, attempts = excluded.attempts, mastered = excluded.mastered",
            params![profile_id, p.pattern_id, p.slot, p.streak, p.attempts, p.mastered],
        )?;
        Ok(())
    }

    // --- Story progress ---------------------------------------------------

    /// Returns the profile's Katsudoo chapter progress, keyed by chapter ID.
    pub fn get_story_progress(
        &self,
        profile_id: i64,
    ) -> Result<std::collections::HashMap<String, StoryProgress>, StorageError> {
        let mut stmt = self.conn.prepare(
            "SELECT chapter_id, beat_index, completed, mastered FROM story_progress WHERE profile_id = ?1",
        )?;
        let rows = stmt.query_map(params![profile_id], |row| {
            Ok(StoryProgress {
                chapter_id: row.get(0)?,
                beat_index: row.get(1)?,
                completed: row.get(2)?,
                mastered: row.get(3)?,
            })
        })?;
        let mut map = std::collections::HashMap::new();
        for r in rows {
            let p = r?;
            map.insert(p.chapter_id.clone(), p);
        }
        Ok(map)
    }

    /// Inserts or updates progress for one chapter.
    pub fn save_story_progress(
        &self,
        profile_id: i64,
        p: &StoryProgress,
    ) -> Result<(), StorageError> {
        self.conn.execute(
            "INSERT INTO story_progress (profile_id, chapter_id, beat_index, completed, mastered) \
             VALUES (?1, ?2, ?3, ?4, ?5) \
             ON CONFLICT (profile_id, chapter_id) DO UPDATE SET \
               beat_index = excluded.beat_index, completed = excluded.completed, \
               mastered = excluded.mastered",
            params![
                profile_id,
                p.chapter_id,
                p.beat_index,
                p.completed,
                p.mastered
            ],
        )?;
        Ok(())
    }

    // --- Assessment -------------------------------------------------------

    /// Returns the profile's result for a level's mock assessment. A level
    /// never assessed yields a zero-value (not passed) result, not an error.
    pub fn get_assessment_result(
        &self,
        profile_id: i64,
        level: Jlpt,
    ) -> Result<AssessmentResult, StorageError> {
        let raw = self
            .conn
            .query_row(
                "SELECT passed, best_correct, total, taken_at FROM assessment_result \
                 WHERE profile_id = ?1 AND level = ?2",
                params![profile_id, level.as_str()],
                |row| {
                    Ok((
                        row.get::<_, bool>(0)?,
                        row.get::<_, i64>(1)?,
                        row.get::<_, i64>(2)?,
                        row.get::<_, Option<String>>(3)?,
                    ))
                },
            )
            .optional()?;
        match raw {
            None => Ok(AssessmentResult {
                level,
                passed: false,
                best_correct: 0,
                total: 0,
                taken_at: None,
            }),
            Some((passed, best_correct, total, taken_at)) => Ok(AssessmentResult {
                level,
                passed,
                best_correct,
                total,
                taken_at: parse_time_opt(taken_at)?,
            }),
        }
    }

    /// Inserts or updates the result for one level.
    pub fn save_assessment_result(
        &self,
        profile_id: i64,
        r: &AssessmentResult,
    ) -> Result<(), StorageError> {
        let taken_at = r.taken_at.map(|t| fmt_time(Some(t)));
        self.conn.execute(
            "INSERT INTO assessment_result (profile_id, level, passed, best_correct, total, taken_at) \
             VALUES (?1, ?2, ?3, ?4, ?5, ?6) \
             ON CONFLICT (profile_id, level) DO UPDATE SET \
               passed = excluded.passed, best_correct = excluded.best_correct, \
               total = excluded.total, taken_at = excluded.taken_at",
            params![profile_id, r.level.as_str(), r.passed, r.best_correct, r.total, taken_at],
        )?;
        Ok(())
    }

    // --- Stats ------------------------------------------------------------

    /// Returns the aggregate stats for a profile, or [`StorageError::NotFound`].
    pub fn get_stats(&self, profile_id: i64) -> Result<Stats, StorageError> {
        let raw = self
            .conn
            .query_row(
                "SELECT streak, best_streak, last_studied_at, xp FROM stats WHERE profile_id = ?1",
                params![profile_id],
                |row| {
                    Ok((
                        row.get::<_, i64>(0)?,
                        row.get::<_, i64>(1)?,
                        row.get::<_, Option<String>>(2)?,
                        row.get::<_, i64>(3)?,
                    ))
                },
            )
            .optional()?;
        let (streak, best_streak, last, xp) = raw.ok_or(StorageError::NotFound)?;
        Ok(Stats {
            streak,
            best_streak,
            last_studied_at: parse_time_opt(last)?,
            xp,
        })
    }

    /// Replaces the aggregate stats for a profile.
    pub fn save_stats(&self, profile_id: i64, stats: &Stats) -> Result<(), StorageError> {
        let last = stats.last_studied_at.map(|t| fmt_time(Some(t)));
        let n = self.conn.execute(
            "UPDATE stats SET streak = ?1, best_streak = ?2, last_studied_at = ?3, xp = ?4 \
             WHERE profile_id = ?5",
            params![stats.streak, stats.best_streak, last, stats.xp, profile_id],
        )?;
        require_affected(n)
    }

    /// Atomically increments a profile's cumulative experience points.
    pub fn add_xp(&self, profile_id: i64, amount: i64) -> Result<(), StorageError> {
        let n = self.conn.execute(
            "UPDATE stats SET xp = xp + ?1 WHERE profile_id = ?2",
            params![amount, profile_id],
        )?;
        require_affected(n)
    }

    /// Returns how many cards the profile has reviewed at least once
    /// successfully (reps > 0).
    pub fn count_learned_cards(&self, profile_id: i64) -> Result<i64, StorageError> {
        let n = self.conn.query_row(
            "SELECT COUNT(*) FROM card_states WHERE profile_id = ?1 AND reps > 0",
            params![profile_id],
            |row| row.get(0),
        )?;
        Ok(n)
    }
}

// --- Path helpers ---------------------------------------------------------

/// Returns the path to the application's database file, creating the enclosing
/// directory if necessary. Uses the OS-appropriate user config directory.
pub fn default_path() -> Result<PathBuf, StorageError> {
    let base = dirs::config_dir().ok_or_else(|| {
        StorageError::Io(std::io::Error::new(
            std::io::ErrorKind::NotFound,
            "locate user config dir",
        ))
    })?;
    let app_dir = base.join("polyglot");
    std::fs::create_dir_all(&app_dir).map_err(StorageError::Io)?;
    Ok(app_dir.join("polyglot.db"))
}

/// Deletes the database at `path` along with its WAL and shared-memory sidecar
/// files. Missing files are not an error.
pub fn remove(path: impl AsRef<Path>) -> Result<(), StorageError> {
    let path = path.as_ref();
    for suffix in ["", "-wal", "-shm"] {
        let p = if suffix.is_empty() {
            path.to_path_buf()
        } else {
            let mut s = path.as_os_str().to_os_string();
            s.push(suffix);
            PathBuf::from(s)
        };
        match std::fs::remove_file(&p) {
            Ok(()) => {}
            Err(e) if e.kind() == std::io::ErrorKind::NotFound => {}
            Err(e) => return Err(StorageError::Io(e)),
        }
    }
    Ok(())
}

// --- Row helpers ----------------------------------------------------------

type ProfileRow = (i64, String, bool, bool, bool, String);

fn profile_row(row: &rusqlite::Row<'_>) -> rusqlite::Result<ProfileRow> {
    Ok((
        row.get(0)?,
        row.get(1)?,
        row.get(2)?,
        row.get(3)?,
        row.get(4)?,
        row.get(5)?,
    ))
}

fn build_profile(r: ProfileRow) -> Result<Profile, StorageError> {
    let (id, name, onboarded, show_romaji, kana_onboarded, created_at) = r;
    Ok(Profile {
        id,
        name,
        onboarded,
        show_romaji,
        kana_onboarded,
        created_at: Some(parse_dt(&created_at)?),
    })
}

type CardStateRow = (String, i64, f64, i64, i64, String, String);

fn card_state_row(row: &rusqlite::Row<'_>) -> rusqlite::Result<CardStateRow> {
    Ok((
        row.get(0)?,
        row.get(1)?,
        row.get(2)?,
        row.get(3)?,
        row.get(4)?,
        row.get(5)?,
        row.get(6)?,
    ))
}

fn build_card_state(r: CardStateRow) -> Result<CardState, StorageError> {
    let (card_id, interval, ease, reps, lapses, due_at, last_reviewed_at) = r;
    Ok(CardState {
        card_id,
        interval,
        ease,
        reps,
        lapses,
        due_at: parse_time_req(&due_at)?,
        last_reviewed_at: parse_time_req(&last_reviewed_at)?,
    })
}

fn require_affected(n: usize) -> Result<(), StorageError> {
    if n == 0 {
        Err(StorageError::NotFound)
    } else {
        Ok(())
    }
}

// --- Time helpers ---------------------------------------------------------

/// The year-1 sentinel used to round-trip a `None` time through a NOT NULL
/// column (mirrors the Go zero `time.Time`).
fn zero_time() -> DateTime<Utc> {
    Utc.with_ymd_and_hms(1, 1, 1, 0, 0, 0).unwrap()
}

/// Formats an optional time for a NOT NULL column: `None` becomes the sentinel.
fn fmt_time(t: Option<DateTime<Utc>>) -> String {
    t.unwrap_or_else(zero_time)
        .to_rfc3339_opts(SecondsFormat::AutoSi, true)
}

fn parse_dt(s: &str) -> Result<DateTime<Utc>, StorageError> {
    DateTime::parse_from_rfc3339(s)
        .map(|dt| dt.with_timezone(&Utc))
        .map_err(|e| StorageError::Time(format!("parse timestamp {s:?}: {e}")))
}

/// Parses a NOT NULL time column, mapping the year-1 sentinel back to `None`.
fn parse_time_req(s: &str) -> Result<Option<DateTime<Utc>>, StorageError> {
    let dt = parse_dt(s)?;
    Ok(if dt.year() <= 1 { None } else { Some(dt) })
}

/// Parses a nullable time column: SQL NULL is `None`.
fn parse_time_opt(s: Option<String>) -> Result<Option<DateTime<Utc>>, StorageError> {
    match s {
        None => Ok(None),
        Some(s) => Ok(Some(parse_dt(&s)?)),
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::srs;

    fn store() -> SqliteStore {
        SqliteStore::open_in_memory().expect("open in-memory db")
    }

    #[test]
    fn profile_lifecycle_and_active() {
        let s = store();
        let p = s.create_profile("Yui").unwrap();
        assert_eq!(p.name, "Yui");
        assert!(p.show_romaji, "new profiles default to showing romaji");
        assert!(!p.onboarded);

        let got = s.get_profile(p.id).unwrap();
        assert_eq!(got, p);

        s.set_onboarded(p.id).unwrap();
        s.set_show_romaji(p.id, false).unwrap();
        s.set_kana_onboarded(p.id).unwrap();
        let got = s.get_profile(p.id).unwrap();
        assert!(got.onboarded && got.kana_onboarded && !got.show_romaji);

        assert_eq!(s.active_profile_id().unwrap(), None);
        s.set_active_profile_id(p.id).unwrap();
        assert_eq!(s.active_profile_id().unwrap(), Some(p.id));

        assert_eq!(s.list_profiles().unwrap().len(), 1);
    }

    #[test]
    fn get_missing_profile_is_not_found() {
        let s = store();
        assert!(matches!(s.get_profile(999), Err(StorageError::NotFound)));
    }

    #[test]
    fn card_state_round_trip_and_upsert() {
        let s = store();
        let p = s.create_profile("A").unwrap();
        let now = Utc::now();
        let state = srs::review(&srs::new_card("greetings:1"), srs::Grade::Good, now);

        s.save_card_state(p.id, &state).unwrap();
        let got = s.get_card_state(p.id, "greetings:1").unwrap();
        assert_eq!(got.reps, 1);
        assert_eq!(got.card_id, "greetings:1");
        assert!(got.due_at.is_some());

        // Upsert: a second review updates in place.
        let state2 = srs::review(&got, srs::Grade::Good, now);
        s.save_card_state(p.id, &state2).unwrap();
        assert_eq!(s.get_card_states(p.id).unwrap().len(), 1);
        assert_eq!(s.count_learned_cards(p.id).unwrap(), 1);
    }

    #[test]
    fn delete_profile_cascades() {
        let s = store();
        let p = s.create_profile("A").unwrap();
        let state = srs::review(&srs::new_card("c1"), srs::Grade::Good, Utc::now());
        s.save_card_state(p.id, &state).unwrap();

        s.delete_profile(p.id).unwrap();
        assert!(matches!(s.get_profile(p.id), Err(StorageError::NotFound)));
        // Cascade removed the card state too.
        assert!(matches!(
            s.get_card_state(p.id, "c1"),
            Err(StorageError::NotFound)
        ));
    }

    #[test]
    fn stats_and_xp() {
        let s = store();
        let p = s.create_profile("A").unwrap();
        let mut stats = s.get_stats(p.id).unwrap();
        assert_eq!(stats.xp, 0);
        assert_eq!(stats.last_studied_at, None);

        stats.streak = 3;
        stats.best_streak = 3;
        stats.last_studied_at = Some(Utc::now());
        s.save_stats(p.id, &stats).unwrap();
        s.add_xp(p.id, 25).unwrap();

        let got = s.get_stats(p.id).unwrap();
        assert_eq!(got.streak, 3);
        assert_eq!(got.xp, 25);
        assert!(got.last_studied_at.is_some());
    }

    #[test]
    fn assessment_absent_is_zero_value() {
        let s = store();
        let p = s.create_profile("A").unwrap();
        let r = s.get_assessment_result(p.id, Jlpt::N5).unwrap();
        assert!(!r.passed);
        assert_eq!(r.level, Jlpt::N5);

        s.save_assessment_result(
            p.id,
            &AssessmentResult {
                level: Jlpt::N5,
                passed: true,
                best_correct: 13,
                total: 15,
                taken_at: Some(Utc::now()),
            },
        )
        .unwrap();
        let r = s.get_assessment_result(p.id, Jlpt::N5).unwrap();
        assert!(r.passed);
        assert_eq!(r.best_correct, 13);
        assert!(r.taken_at.is_some());
    }

    #[test]
    fn kana_pattern_story_progress_round_trip() {
        let s = store();
        let p = s.create_profile("A").unwrap();

        s.save_kana_progress(
            p.id,
            &KanaProgress {
                char: "あ".to_string(),
                streak: 3,
                attempts: 4,
                mastered: true,
                best_ms: 420,
            },
        )
        .unwrap();
        assert_eq!(s.get_kana_progress(p.id).unwrap()["あ"].best_ms, 420);

        s.save_pattern_progress(
            p.id,
            &PatternProgress {
                pattern_id: "copula".to_string(),
                slot: "X".to_string(),
                streak: 2,
                attempts: 2,
                mastered: false,
            },
        )
        .unwrap();
        assert!(s
            .get_pattern_progress(p.id)
            .unwrap()
            .contains_key("copula:X"));

        s.save_story_progress(
            p.id,
            &StoryProgress {
                chapter_id: "ch1".to_string(),
                beat_index: 4,
                completed: true,
                mastered: true,
            },
        )
        .unwrap();
        assert!(s.get_story_progress(p.id).unwrap()["ch1"].mastered);
    }
}
