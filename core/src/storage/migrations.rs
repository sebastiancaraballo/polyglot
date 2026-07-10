use rusqlite::Connection;

/// The ordered, forward-only schema migrations, embedded from `migrations/`.
/// Each is applied once, tracked via SQLite's `user_version` pragma (replacing
/// the Go original's `goose` migration table — this is a fresh database, so no
/// goose-table compatibility is needed).
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
];

/// Applies every pending migration, in order, each in its own transaction.
pub(super) fn migrate(conn: &Connection) -> rusqlite::Result<()> {
    let current: i64 = conn.query_row("PRAGMA user_version", [], |r| r.get(0))?;
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
