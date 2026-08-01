use super::Jlpt;

/// A named blank in a grammar pattern's frame, filled by choosing among
/// candidate vocabulary cards the learner already knows. A slot never
/// introduces new vocabulary through the grammar drill itself ("words before
/// sentences") — every candidate must be an existing `Card::id`.
#[derive(Clone, Debug, PartialEq)]
pub struct Slot {
    /// Placeholder name referenced in the frame, e.g. `"X"`, `"N"`.
    pub name: String,
    /// Candidate vocab `Card::id` values this slot may be filled with.
    pub card_ids: Vec<String>,
    /// `Card::id` held fixed when this slot is not the round's variable slot
    /// (Cognitive Load Theory: only one slot varies per round).
    pub default: String,
}

/// A fixed sentence frame with one or more slots, used for structured-input /
/// minimal-substitution practice (Processing Instruction, VanPatten). `frame`
/// holds each slot's name in `"{Name}"` placeholders, e.g. `"{X}は{N}です"`.
#[derive(Clone, Debug, PartialEq)]
pub struct Pattern {
    pub id: String,
    /// Short label, e.g. `"X wa N desu"`.
    pub title: String,
    pub jlpt: Option<Jlpt>,
    pub frame: String,
    pub slots: Vec<Slot>,
    pub notes: String,
}
