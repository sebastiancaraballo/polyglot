//! Polyglot terminal frontend (ratatui).
//!
//! Wires storage + content + the ratatui router, mirroring the Go
//! `cmd/polyglot` + `internal/app`. During the TUI port the menu, stats, and
//! kana chart are real; other destinations render a placeholder.

mod app;
mod art;
mod frame;
mod screens;
mod textfmt;
mod theme;

use std::error::Error;

use polyglot_core::content::{self};
use polyglot_core::i18n;
use polyglot_core::storage::SqliteStore;

use app::App;
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

    let app = App::new(
        Theme::default_theme(),
        i18n::default(),
        env!("CARGO_PKG_VERSION").to_string(),
        store,
        course,
    )?;
    app::run(app)?;
    Ok(())
}
