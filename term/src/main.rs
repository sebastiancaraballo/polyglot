//! Polyglot terminal frontend (ratatui).
//!
//! Wires storage + content + the ratatui router, mirroring the Go
//! `cmd/polyglot` + `internal/app`. During the TUI port the menu is real and
//! other destinations render a placeholder.

mod app;
mod frame;
mod screens;
mod theme;

use std::collections::HashSet;
use std::error::Error;

use polyglot_core::content::{self, Course};
use polyglot_core::model::{Jlpt, Profile};
use polyglot_core::storage::{SqliteStore, StorageError};
use polyglot_core::{i18n, study};

use app::App;
use screens::menu::Summary;
use theme::Theme;

fn main() {
    if let Err(e) = real_main() {
        eprintln!("error: {e}");
        std::process::exit(1);
    }
}

fn real_main() -> Result<(), Box<dyn Error>> {
    let course = content::load_embedded(content::DEFAULT_PAIR)?;
    let db_path = polyglot_core::storage::default_path()?;
    let store = SqliteStore::open(&db_path)?;

    let summary = build_summary(&store, &course)?;
    let app = App::new(
        Theme::default_theme(),
        i18n::default(),
        summary,
        env!("CARGO_PKG_VERSION").to_string(),
    );
    app::run(app)?;
    Ok(())
}

/// Returns the persisted active profile, the first existing profile, or `None`
/// on a first run with no profiles yet.
fn resolve_profile(store: &SqliteStore) -> Result<Option<Profile>, StorageError> {
    if let Some(id) = store.active_profile_id()? {
        match store.get_profile(id) {
            Ok(p) => return Ok(Some(p)),
            Err(StorageError::NotFound) => {}
            Err(e) => return Err(e),
        }
    }
    let profiles = store.list_profiles()?;
    match profiles.into_iter().next() {
        None => Ok(None),
        Some(p) => {
            store.set_active_profile_id(p.id)?;
            Ok(Some(p))
        }
    }
}

/// Builds the menu summary from the course and the active profile's progress,
/// computing each activity's gate through the core engine.
fn build_summary(store: &SqliteStore, course: &Course) -> Result<Summary, Box<dyn Error>> {
    let total =
        (course.kana.len() + course.lessons.iter().map(|l| l.cards.len()).sum::<usize>()) as i64;

    let Some(profile) = resolve_profile(store)? else {
        return Ok(Summary {
            total,
            reading_locked: true,
            rikai_locked: true,
            assessment_locked: true,
            ..Summary::default()
        });
    };

    let stats = store.get_stats(profile.id)?;
    let learned = store.count_learned_cards(profile.id)?;

    let kana_progress = store.get_kana_progress(profile.id)?;
    let gate = study::new_gate(&course.kana, &kana_progress);

    let card_states = store.get_card_states(profile.id)?;
    let known: HashSet<String> = card_states
        .iter()
        .filter(|(_, s)| s.reps > 0)
        .map(|(id, _)| id.clone())
        .collect();
    let rikai_locked = !course
        .patterns
        .iter()
        .any(|p| study::pattern_ready(p, &known));

    let story_progress = store.get_story_progress(profile.id)?;
    let all_mastered = !course.chapters.is_empty()
        && course
            .chapters
            .iter()
            .all(|c| story_progress.get(&c.id).is_some_and(|sp| sp.mastered));
    let assessment_passed = store.get_assessment_result(profile.id, Jlpt::N5)?.passed;

    Ok(Summary {
        name: profile.name,
        xp: stats.xp,
        streak: stats.streak,
        learned,
        total,
        reading_locked: !gate.reading_unlocked(),
        rikai_locked,
        assessment_locked: !all_mastered,
        assessment_passed,
    })
}
