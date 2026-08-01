use super::Jlpt;

/// A single vocabulary or phrase item: a term in the learner's source language
/// paired with its Japanese form and romaji reading.
#[derive(Clone, Debug, PartialEq)]
pub struct Card {
    /// Stable identifier, `"<lesson_id>:<index>"`.
    pub id: String,
    /// Term in the learner's source language (Spanish in v1).
    pub source: String,
    /// Japanese form.
    pub jp: String,
    /// Romanized reading.
    pub romaji: String,
    /// Optional usage notes.
    pub notes: String,
    pub jlpt: Option<Jlpt>,
    /// Communicative function IDs, inherited from the lesson.
    pub functions: Vec<String>,
    /// Frequency rank (lower = more frequent); `0` means unset.
    pub freq: i64,
}

/// An ordered collection of cards sharing a theme and JLPT level.
#[derive(Clone, Debug, PartialEq)]
pub struct Lesson {
    pub id: String,
    pub title: String,
    pub jlpt: Option<Jlpt>,
    /// Communicative function IDs realized by this lesson.
    pub functions: Vec<String>,
    pub cards: Vec<Card>,
}
