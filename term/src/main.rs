//! Polyglot terminal frontend (ratatui).
//!
//! Placeholder entry point for the rewrite scaffold. The ratatui router and
//! screens land in the TUI porting phase; for now this smoke-tests that the
//! frontend links the core crate.

fn main() {
    let card = polyglot_core::srs::new_card("demo");
    println!("polyglot — Rust rewrite scaffold");
    println!(
        "core linked OK: new card {:?} due immediately",
        card.card_id
    );
}
