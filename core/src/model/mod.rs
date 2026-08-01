//! Domain types shared across the learning engine.
//!
//! Port of the Go `internal/model` package. String-valued Go enum types (e.g.
//! `KanaType`, `CEFR`) become Rust enums with `as_str`/`from_str`, preserving
//! the exact wire strings used in YAML content and the SQLite database. The Go
//! `Valid()` check on those string types maps to `from_str(..).is_some()`.

mod assessment;
mod card_state;
mod cefr;
mod frequency;
mod function;
mod grammar;
mod jlpt;
mod kana;
mod lesson;
mod name;
mod profile;
mod progress;
mod stats;
mod story;

pub use assessment::AssessmentResult;
pub use card_state::{CardState, DEFAULT_EASE};
pub use cefr::{Cefr, CEFR_LEVELS};
pub use frequency::FreqEntry;
pub use function::{Function, FunctionCatalog};
pub use grammar::{Pattern, Slot};
pub use jlpt::{Jlpt, JLPT_LEVELS};
pub use kana::{KanaCategory, KanaItem, KanaType};
pub use lesson::{Card, Lesson};
pub use name::{normalize_name, NameError, MAX_NAME_LEN};
pub use profile::Profile;
pub use progress::{KanaProgress, PatternProgress, StoryProgress};
pub use stats::Stats;
pub use story::{Beat, BeatKind, Chapter, PracticeKind};
