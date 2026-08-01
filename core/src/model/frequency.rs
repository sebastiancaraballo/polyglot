/// One ranked word in a target-language word-frequency list. Rank 1 is the most
/// frequent word; `count` is the raw occurrence count in the source corpus.
/// Frequency is a property of the target language, so the list is shared across
/// every language pair that teaches it.
#[derive(Clone, Debug, PartialEq)]
pub struct FreqEntry {
    pub rank: i64,
    pub word: String,
    pub reading: String,
    pub count: i64,
}
