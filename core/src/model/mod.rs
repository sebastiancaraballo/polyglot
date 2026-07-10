//! Domain types shared across the learning engine.
//!
//! Port of the Go `internal/model` package. Types are added here as the port
//! progresses, one per file, re-exported from this module.

mod card_state;

pub use card_state::{CardState, DEFAULT_EASE};
