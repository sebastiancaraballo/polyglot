use std::collections::HashMap;

use crate::model::{FreqEntry, Lesson};

/// Builds a lookup from a frequency list: surface word first, then reading for
/// entries whose surface form (e.g. kanji 私) can't match our kana-only cards.
/// The lowest (most frequent) rank wins on collisions.
///
/// Known limitation, accepted: kana homographs may borrow a more frequent
/// homograph's rank; an explicit `freq:` in the lesson YAML overrides it.
pub(super) fn freq_index(entries: &[FreqEntry]) -> HashMap<String, i64> {
    let mut index: HashMap<String, i64> = HashMap::with_capacity(entries.len() * 2);
    for e in entries {
        index.entry(e.word.clone()).or_insert(e.rank);
    }
    for e in entries {
        if e.reading.is_empty() {
            continue;
        }
        index.entry(e.reading.clone()).or_insert(e.rank);
    }
    index
}

/// Fills each card's `freq` from the index. A rank set explicitly in the lesson
/// YAML wins; words not in the list stay at `0` = unranked.
pub(super) fn backfill_freq(lessons: &mut [Lesson], index: &HashMap<String, i64>) {
    for lesson in lessons.iter_mut() {
        for card in lesson.cards.iter_mut() {
            if card.freq != 0 {
                continue;
            }
            if let Some(&rank) = index.get(&card.jp) {
                card.freq = rank;
            }
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::Card;

    fn card(jp: &str, freq: i64) -> Card {
        Card {
            id: jp.to_string(),
            source: String::new(),
            jp: jp.to_string(),
            romaji: String::new(),
            notes: String::new(),
            jlpt: None,
            functions: Vec::new(),
            freq,
        }
    }

    #[test]
    fn backfills_by_word_and_reading_but_respects_explicit() {
        let entries = vec![
            FreqEntry {
                rank: 1,
                word: "\u{79c1}".to_string(),
                reading: "\u{308f}\u{305f}\u{3057}".to_string(),
                count: 0,
            },
            FreqEntry {
                rank: 5,
                word: "\u{307f}\u{305a}".to_string(),
                reading: String::new(),
                count: 0,
            },
        ];
        let index = freq_index(&entries);
        let mut lessons = vec![Lesson {
            id: "l".to_string(),
            title: String::new(),
            jlpt: None,
            functions: Vec::new(),
            cards: vec![
                card("\u{308f}\u{305f}\u{3057}", 0), // matches by reading -> rank 1
                card("\u{307f}\u{305a}", 0),         // matches by word -> rank 5
                card("\u{307f}\u{305a}", 3),         // explicit rank wins
                card("\u{306a}\u{3044}", 0),         // absent -> stays 0
            ],
        }];
        backfill_freq(&mut lessons, &index);
        let cards = &lessons[0].cards;
        assert_eq!(cards[0].freq, 1);
        assert_eq!(cards[1].freq, 5);
        assert_eq!(cards[2].freq, 3);
        assert_eq!(cards[3].freq, 0);
    }
}
