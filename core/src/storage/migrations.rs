use rusqlite::{Connection, OptionalExtension};

/// The ordered, forward-only schema migrations, embedded from `migrations/`.
/// Each is applied once, tracked via SQLite's `user_version` pragma. A database
/// created by the original Go app (via `goose`) has the full schema but
/// `user_version` 0; [`migrate`] detects that and adopts goose's applied
/// version instead of trying to recreate existing tables, so the two clients
/// interoperate on the same database file.
const MIGRATIONS: &[&str] = &[
    include_str!("migrations/00001_init.sql"),
    include_str!("migrations/00002_add_xp.sql"),
    include_str!("migrations/00003_app_meta.sql"),
    include_str!("migrations/00004_profile_show_romaji.sql"),
    include_str!("migrations/00005_kana_progress.sql"),
    include_str!("migrations/00006_profile_kana_onboarded.sql"),
    include_str!("migrations/00007_pattern_progress.sql"),
    include_str!("migrations/00008_story_progress.sql"),
    include_str!("migrations/00009_story_progress_mastered.sql"),
    include_str!("migrations/00010_assessment_result.sql"),
    include_str!("migrations/00011_kanji_progress.sql"),
];

/// Applies every pending migration, in order, each in its own transaction.
pub(super) fn migrate(conn: &Connection) -> rusqlite::Result<()> {
    let mut current: i64 = conn.query_row("PRAGMA user_version", [], |r| r.get(0))?;

    // Interop with a Go/goose-created database: it holds the full schema but
    // leaves `user_version` at 0. Adopt goose's applied migration count so we
    // skip the migrations whose tables already exist.
    if current == 0 && table_exists(conn, "profiles")? {
        current = goose_version(conn).unwrap_or(MIGRATIONS.len() as i64);
        conn.pragma_update(None, "user_version", current)?;
    }

    for (i, sql) in MIGRATIONS.iter().enumerate() {
        let version = (i + 1) as i64;
        if version <= current {
            continue;
        }
        let tx = conn.unchecked_transaction()?;
        tx.execute_batch(sql)?;
        tx.pragma_update(None, "user_version", version)?;
        tx.commit()?;
    }
    Ok(())
}

fn table_exists(conn: &Connection, name: &str) -> rusqlite::Result<bool> {
    Ok(conn
        .query_row(
            "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?1",
            [name],
            |_| Ok(()),
        )
        .optional()?
        .is_some())
}

/// Returns goose's highest applied migration version, if its bookkeeping table
/// is present. Goose numbers migrations by their filename prefix (`00001` → 1),
/// so the maximum applied `version_id` is the schema version.
fn goose_version(conn: &Connection) -> Option<i64> {
    if !table_exists(conn, "goose_db_version").ok()? {
        return None;
    }
    conn.query_row(
        "SELECT MAX(version_id) FROM goose_db_version WHERE is_applied = 1",
        [],
        |r| r.get::<_, Option<i64>>(0),
    )
    .ok()
    .flatten()
}

#[cfg(test)]
mod tests {
    use super::*;

    /// The real interop case: a Go/goose database stops at the last migration
    /// goose knew about, and the Rust runner must carry it forward from there —
    /// this is what happens to an existing learner's database on upgrade.
    #[test]
    fn applies_new_migrations_on_top_of_a_goose_database() {
        const GOOSE_VERSION: usize = 10; // the last migration the Go app shipped
        let conn = Connection::open_in_memory().unwrap();
        for sql in MIGRATIONS.iter().take(GOOSE_VERSION) {
            conn.execute_batch(sql).unwrap();
        }
        conn.execute_batch(&format!(
            "CREATE TABLE goose_db_version (\
               id INTEGER PRIMARY KEY, version_id INTEGER, is_applied INTEGER, tstamp TEXT);\
             INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1), ({GOOSE_VERSION}, 1);"
        ))
        .unwrap();
        conn.pragma_update(None, "user_version", 0i64).unwrap();
        assert!(!table_exists(&conn, "kanji_progress").unwrap());

        migrate(&conn).unwrap();

        assert!(
            table_exists(&conn, "kanji_progress").unwrap(),
            "the post-goose migrations must be applied"
        );
        let v: i64 = conn
            .query_row("PRAGMA user_version", [], |r| r.get(0))
            .unwrap();
        assert_eq!(v, MIGRATIONS.len() as i64);
    }

    #[test]
    fn adopts_a_goose_created_schema() {
        let conn = Connection::open_in_memory().unwrap();
        // Simulate the Go/goose database: full schema, a goose bookkeeping table
        // recording all migrations applied, but user_version left at 0.
        for sql in MIGRATIONS {
            conn.execute_batch(sql).unwrap();
        }
        conn.execute_batch(
            "CREATE TABLE goose_db_version (\
               id INTEGER PRIMARY KEY, version_id INTEGER, is_applied INTEGER, tstamp TEXT);\
             INSERT INTO goose_db_version (version_id, is_applied) VALUES (0, 1), (10, 1);",
        )
        .unwrap();
        conn.pragma_update(None, "user_version", 0i64).unwrap();

        // migrate must NOT try to recreate the existing tables.
        migrate(&conn).unwrap();
        let v: i64 = conn
            .query_row("PRAGMA user_version", [], |r| r.get(0))
            .unwrap();
        assert_eq!(v, MIGRATIONS.len() as i64);
    }
}
