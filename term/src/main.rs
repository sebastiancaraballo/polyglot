//! Polyglot terminal frontend (ratatui).
//!
//! Wires storage + content + the ratatui router, mirroring the Go
//! `cmd/polyglot` + `internal/app`.

mod app;
mod art;
mod frame;
mod screens;
mod textfmt;
mod theme;

#[cfg(test)]
mod golden;
#[cfg(test)]
mod testutil;

use std::error::Error;
use std::path::PathBuf;

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
    // `POLYGLOT_DB` overrides the database location — useful for trying the app
    // against a throwaway database without touching the real profile data.
    let db_path = match std::env::var_os("POLYGLOT_DB") {
        Some(p) => PathBuf::from(p),
        None => polyglot_core::storage::default_path()?,
    };
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
