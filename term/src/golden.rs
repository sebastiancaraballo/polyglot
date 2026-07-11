//! Golden (snapshot) tests for the deterministic screens — the analogue of the
//! Go screens' `testdata/*.golden` files. Screens that draw with an RNG (kana,
//! quiz, rikai, assessment, story) are covered by behavior tests in their own
//! modules instead, exactly as in the Go tree.

use polyglot_core::content::{self, Course};
use polyglot_core::i18n::{self, Messages};
use polyglot_core::storage::SqliteStore;
use ratatui::crossterm::event::{KeyCode, KeyModifiers};

use crate::app::Ctx;
use crate::screens::flashcards::Flashcards;
use crate::screens::kanachart::KanaChart;
use crate::screens::menu::{Menu, Summary};
use crate::screens::onboarding::Onboarding;
use crate::screens::profiles::Profiles;
use crate::screens::profilesetup::ProfileSetup;
use crate::screens::settings::Settings;
use crate::screens::stats::Stats;
use crate::testutil::snapshot;

fn course() -> Course {
    content::load_embedded(content::DEFAULT_PAIR).unwrap()
}
fn msgs() -> &'static Messages {
    i18n::default()
}

fn store_with_profile() -> (SqliteStore, i64) {
    let s = SqliteStore::open_in_memory().unwrap();
    let p = s.create_profile("Yui").unwrap();
    (s, p.id)
}

fn sample_summary() -> Summary {
    Summary {
        name: "Yui".to_string(),
        xp: 42,
        streak: 3,
        learned: 5,
        total: 100,
        reading_locked: true,
        rikai_locked: true,
        assessment_locked: true,
        ..Default::default()
    }
}

#[test]
fn menu_top_level() {
    let m = Menu::new(msgs(), sample_summary(), "0.1.0".to_string());
    insta::assert_snapshot!(snapshot(|f, inner, theme| m.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn menu_submenu() {
    let store = SqliteStore::open_in_memory().unwrap();
    let ctx = Ctx {
        store: &store,
        profile_id: None,
    };
    let mut m = Menu::new(msgs(), sample_summary(), "0.1.0".to_string());
    m.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // open "Aprender"
    insta::assert_snapshot!(snapshot(|f, inner, theme| m.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn stats_screen() {
    let (store, pid) = store_with_profile();
    store.add_xp(pid, 30).unwrap();
    let s = Stats::new(&store, &course(), Some(pid));
    insta::assert_snapshot!(snapshot(|f, inner, theme| s.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn kana_chart_first_page() {
    let c = KanaChart::new(&course());
    insta::assert_snapshot!(snapshot(|f, inner, theme| c.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn settings_list() {
    let s = Settings::new(true);
    insta::assert_snapshot!(snapshot(|f, inner, theme| s.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn settings_confirm() {
    let store = SqliteStore::open_in_memory().unwrap();
    let ctx = Ctx {
        store: &store,
        profile_id: None,
    };
    let mut s = Settings::new(true);
    s.handle(KeyCode::Down, KeyModifiers::NONE, &ctx); // to "delete profile"
    s.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // open confirm
    insta::assert_snapshot!(snapshot(|f, inner, theme| s.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn profiles_list() {
    let (store, pid) = store_with_profile();
    let p = Profiles::new(&store, Some(pid));
    insta::assert_snapshot!(snapshot(|f, inner, theme| p.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn profile_setup_empty() {
    let s = ProfileSetup::new(true);
    insta::assert_snapshot!(snapshot(|f, inner, theme| s.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn profile_setup_invalid_name() {
    let store = SqliteStore::open_in_memory().unwrap();
    let ctx = Ctx {
        store: &store,
        profile_id: None,
    };
    let mut s = ProfileSetup::new(true);
    for c in ['1', '2', '3'] {
        s.handle(KeyCode::Char(c), KeyModifiers::NONE, &ctx); // digits: no letters
    }
    insta::assert_snapshot!(snapshot(|f, inner, theme| s.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn onboarding_welcome() {
    let o = Onboarding::new(Some(1));
    insta::assert_snapshot!(snapshot(|f, inner, theme| o.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn onboarding_exercise_correct() {
    let store = SqliteStore::open_in_memory().unwrap();
    let ctx = Ctx {
        store: &store,
        profile_id: None,
    };
    let mut o = Onboarding::new(Some(1));
    o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // welcome -> exercise
    o.handle(KeyCode::Char('2'), KeyModifiers::NONE, &ctx); // pick "Agua" (correct)
    insta::assert_snapshot!(snapshot(|f, inner, theme| o.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn onboarding_done() {
    let store = SqliteStore::open_in_memory().unwrap();
    let ctx = Ctx {
        store: &store,
        profile_id: None,
    };
    let mut o = Onboarding::new(Some(1));
    o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx);
    o.handle(KeyCode::Char('2'), KeyModifiers::NONE, &ctx);
    o.handle(KeyCode::Enter, KeyModifiers::NONE, &ctx); // exercise -> done
    insta::assert_snapshot!(snapshot(|f, inner, theme| o.render(
        f,
        inner,
        theme,
        msgs()
    )));
}

#[test]
fn flashcards_nothing_due() {
    // A fresh profile can decode nothing yet, so the review queue is empty.
    let (store, pid) = store_with_profile();
    let f = Flashcards::new(&store, Some(pid), &[], msgs().flash_title.clone(), true);
    insta::assert_snapshot!(snapshot(|frame, inner, theme| f.render(
        frame,
        inner,
        theme,
        msgs()
    )));
}
