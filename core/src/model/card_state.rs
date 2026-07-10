use chrono::{DateTime, Utc};

/// The starting ease factor for a new card (SM-2 style).
pub const DEFAULT_EASE: f64 = 2.5;

/// The spaced-repetition scheduling state of a single card, scoped to a
/// profile. It is updated after every review.
///
/// `due_at`/`last_reviewed_at` are `None` for a card that has never been
/// scheduled/reviewed; a `None` `due_at` means the card is immediately due
/// (mirroring the zero-`time.Time` sentinel used by the Go original).
#[derive(Clone, Debug, PartialEq)]
pub struct CardState {
    pub card_id: String,
    /// Days until the next review.
    pub interval: i64,
    /// Ease factor; higher means longer intervals.
    pub ease: f64,
    /// Consecutive successful reviews.
    pub reps: i64,
    /// Number of times the card was forgotten.
    pub lapses: i64,
    /// When the card is next due for review.
    pub due_at: Option<DateTime<Utc>>,
    pub last_reviewed_at: Option<DateTime<Utc>>,
}

impl CardState {
    /// Creates a card state with the given id and default ease, all other
    /// fields zeroed — the initial state of a never-reviewed card.
    pub fn new(card_id: impl Into<String>) -> Self {
        CardState {
            card_id: card_id.into(),
            interval: 0,
            ease: DEFAULT_EASE,
            reps: 0,
            lapses: 0,
            due_at: None,
            last_reviewed_at: None,
        }
    }
}
