use std::collections::HashMap;

use super::Cefr;

/// A communicative function in the language-agnostic curriculum spine:
/// something a learner can do with language (greet, count, ask for directions),
/// graded with a CEFR level. Functions are shared across all language pairs;
/// each per-language lesson references them to provide a concrete realization
/// (the cultural "skin" over the universal "spine").
#[derive(Clone, Debug, PartialEq)]
pub struct Function {
    pub id: String,
    pub cefr: Cefr,
    /// Authored in-house; not copied from external can-do catalogs.
    pub description: String,
}

/// Maps a function ID to its definition. Look up with `.get(id)`.
pub type FunctionCatalog = HashMap<String, Function>;
