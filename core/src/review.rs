//! Cross-curriculum spaced-repetition study queues.
//!
//! Port of the Go `internal/review` package. The shared scheduler that turns
//! the curriculum's learnable items — kana, vocabulary, and (later) grammar —
//! into a single due-ordered, strand-interleaved session. UI-free: it depends
//! only on the content model, the SRS engine, and storage.

use std::cmp::Ordering;
use std::collections::HashMap;

use chrono::{DateTime, Utc};

use crate::model::{CardState, KanaItem, Lesson};
use crate::srs;
use crate::storage::{SqliteStore, StorageError};

/// New-vs-review pacing: reviews always take priority over introducing new
/// material. New cards only fill seats reviews don't need; a struggling due set
/// further slows the intake.
const MAX_NEW_PER_SESSION: i64 = 10;

/// Returns how many new (never-reviewed) cards may join a session that already
/// contains `due_reviews` due review cards, `lapsed_reviews` of which have
/// lapsed at least once. New cards only fill seats reviews don't need within
/// `limit` (`limit <= 0` means no cap), and a lapse-heavy due set — half or more
/// of the due reviews carry a lapse — halves the intake.
pub fn new_card_budget(due_reviews: i64, lapsed_reviews: i64, limit: i64) -> i64 {
    let mut budget = MAX_NEW_PER_SESSION;
    if limit > 0 {
        let space = limit - due_reviews;
        if space < budget {
            budget = space;
        }
    }
    if budget < 0 {
        budget = 0;
    }
    if due_reviews > 0 && lapsed_reviews * 2 >= due_reviews {
        budget /= 2;
    }
    budget
}

/// Identifies which part of the curriculum an item belongs to. The queue
/// interleaves strands so the learner practices them mixed rather than in
/// blocks.
#[derive(Clone, Copy, Debug, PartialEq, Eq, PartialOrd, Ord, Hash)]
pub enum Strand {
    /// Vocabulary cards.
    Vocab,
    /// Kana (syllabary) characters.
    Kana,
}

/// A single schedulable, renderable unit of study, independent of the strand it
/// came from. The prompt is shown first; the answer is revealed on demand.
#[derive(Clone, Debug, PartialEq)]
pub struct Item {
    /// Stable key used for the card's scheduling state.
    pub card_id: String,
    pub strand: Strand,
    /// The question, shown first.
    pub prompt: String,
    /// The answer, revealed on demand.
    pub answer: String,
    /// Optional secondary line shown with the answer (e.g. romaji).
    pub secondary: String,
    /// Optional usage notes.
    pub notes: String,
    /// Target-language frequency rank; `0` = unranked.
    pub freq: i64,
}

/// Pairs an item with its current spaced-repetition state.
#[derive(Clone, Debug, PartialEq)]
pub struct Scheduled {
    pub item: Item,
    pub state: CardState,
}

/// A built study session plus the pacing facts the UI needs to explain it.
#[derive(Clone, Debug, Default, PartialEq)]
pub struct Queue {
    pub items: Vec<Scheduled>,
    /// Due cards that have been reviewed before.
    pub due_reviews: usize,
    /// Due new cards deferred by pacing, so the UI can say why.
    pub held_back_new: usize,
}

/// Returns the stable scheduling key for a kana item. Kana characters are unique
/// across the syllabaries, so the character alone identifies the card.
pub fn kana_card_id(it: &KanaItem) -> String {
    format!("kana:{}", it.char)
}

/// Turns every lesson's cards into review items, reusing each card's stable ID.
pub fn vocab_items(lessons: &[Lesson]) -> Vec<Item> {
    let mut items = Vec::new();
    for lesson in lessons {
        for c in &lesson.cards {
            items.push(Item {
                card_id: c.id.clone(),
                strand: Strand::Vocab,
                prompt: c.source.clone(),
                answer: c.jp.clone(),
                secondary: c.romaji.clone(),
                notes: c.notes.clone(),
                freq: c.freq,
            });
        }
    }
    items
}

/// Turns kana characters into review items: the character is the prompt, its
/// romaji reading the answer.
pub fn kana_items(kana: &[KanaItem]) -> Vec<Item> {
    kana.iter()
        .map(|k| Item {
            card_id: kana_card_id(k),
            strand: Strand::Kana,
            prompt: k.char.clone(),
            answer: k.romaji.clone(),
            secondary: String::new(),
            notes: String::new(),
            freq: 0,
        })
        .collect()
}

/// Returns the items currently due for the profile, ordered most-overdue first
/// within each strand and interleaved across strands, capped to at most `limit`
/// items (`limit <= 0` means no cap). A never-seen item is treated as a new card
/// that is immediately due. Due reviews take priority; new cards are admitted
/// only up to [`new_card_budget`] — most frequent words first — and the number
/// held back is reported. Deterministic for a given input.
pub fn build_queue(
    store: &SqliteStore,
    profile_id: i64,
    items: &[Item],
    now: DateTime<Utc>,
    limit: i64,
) -> Result<Queue, StorageError> {
    let mut reviews: Vec<Scheduled> = Vec::new();
    let mut fresh: Vec<Scheduled> = Vec::new();
    let mut lapsed: i64 = 0;

    for it in items {
        let state = match store.get_card_state(profile_id, &it.card_id) {
            Ok(s) => s,
            Err(StorageError::NotFound) => srs::new_card(&it.card_id),
            Err(e) => return Err(e),
        };
        if !srs::is_due(&state, now) {
            continue;
        }
        if state.last_reviewed_at.is_none() {
            fresh.push(Scheduled {
                item: it.clone(),
                state,
            });
        } else {
            if state.lapses > 0 {
                lapsed += 1;
            }
            reviews.push(Scheduled {
                item: it.clone(),
                state,
            });
        }
    }

    // Cut the new-card budget by frequency rank: ranked before unranked, most
    // frequent first, stable (curricular input order) otherwise.
    fresh.sort_by(|a, b| {
        let (fi, fj) = (a.item.freq, b.item.freq);
        let (ri, rj) = (fi > 0, fj > 0);
        if ri != rj {
            return if ri {
                Ordering::Less
            } else {
                Ordering::Greater
            };
        }
        if ri {
            fi.cmp(&fj)
        } else {
            Ordering::Equal
        }
    });

    let due_reviews = reviews.len();
    let budget = new_card_budget(due_reviews as i64, lapsed, limit).max(0) as usize;
    let admitted_count = fresh.len().min(budget);
    let held_back_new = fresh.len() - admitted_count;

    let mut combined = reviews;
    combined.extend(fresh.into_iter().take(admitted_count));

    let mut ordered = interleave(combined);
    if limit > 0 && ordered.len() > limit as usize {
        ordered.truncate(limit as usize);
    }

    Ok(Queue {
        items: ordered,
        due_reviews,
        held_back_new,
    })
}

/// Orders due items most-overdue first within each strand, then pulls from the
/// strands round-robin so the session mixes them.
fn interleave(items: Vec<Scheduled>) -> Vec<Scheduled> {
    let mut buckets: HashMap<Strand, Vec<Scheduled>> = HashMap::new();
    let mut strands: Vec<Strand> = Vec::new();
    for s in items {
        if !buckets.contains_key(&s.item.strand) {
            strands.push(s.item.strand);
        }
        buckets.entry(s.item.strand).or_default().push(s);
    }
    strands.sort();
    for st in &strands {
        // `None` due_at (never scheduled) sorts before any timestamp — earliest.
        buckets.get_mut(st).unwrap().sort_by_key(|s| s.state.due_at);
    }

    let total: usize = buckets.values().map(Vec::len).sum();
    let mut out = Vec::with_capacity(total);
    let mut idx: HashMap<Strand, usize> = HashMap::new();
    loop {
        let mut progressed = false;
        for st in &strands {
            let i = idx.entry(*st).or_insert(0);
            let bucket = &buckets[st];
            if *i < bucket.len() {
                out.push(bucket[*i].clone());
                *i += 1;
                progressed = true;
            }
        }
        if !progressed {
            break;
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::model::{Card, KanaCategory, KanaType, DEFAULT_EASE};
    use chrono::{Duration, TimeZone};

    fn mem_store() -> SqliteStore {
        SqliteStore::open_in_memory().unwrap()
    }

    /// A bare item, mirroring the Go test helper of the same name.
    fn item(id: &str, strand: Strand) -> Item {
        Item {
            card_id: id.to_string(),
            strand,
            prompt: id.to_string(),
            answer: id.to_string(),
            secondary: String::new(),
            notes: String::new(),
            freq: 0,
        }
    }

    fn freq_item(id: &str, freq: i64) -> Item {
        Item {
            freq,
            ..item(id, Strand::Vocab)
        }
    }

    /// Stores a due review card (reviewed before, so not a new card).
    fn save(store: &SqliteStore, profile_id: i64, id: &str, due: DateTime<Utc>) {
        save_state(store, profile_id, id, due, 0);
    }

    /// Stores a due review card carrying a lapse.
    fn save_lapsed(store: &SqliteStore, profile_id: i64, id: &str, due: DateTime<Utc>) {
        save_state(store, profile_id, id, due, 1);
    }

    fn save_state(store: &SqliteStore, profile_id: i64, id: &str, due: DateTime<Utc>, lapses: i64) {
        let state = CardState {
            card_id: id.to_string(),
            interval: 1,
            ease: DEFAULT_EASE,
            reps: 1,
            lapses,
            due_at: Some(due),
            last_reviewed_at: Some(due),
        };
        store.save_card_state(profile_id, &state).unwrap();
    }

    fn ids(q: &Queue) -> Vec<&str> {
        q.items.iter().map(|s| s.item.card_id.as_str()).collect()
    }

    /// The fixed clock the ordering tests schedule around.
    fn at() -> DateTime<Utc> {
        Utc.with_ymd_and_hms(2026, 6, 25, 9, 0, 0).unwrap()
    }

    fn lesson(cards: Vec<Card>) -> Lesson {
        Lesson {
            id: "l".to_string(),
            title: String::new(),
            jlpt: None,
            functions: Vec::new(),
            cards,
        }
    }

    fn card(id: &str, freq: i64) -> Card {
        Card {
            id: id.to_string(),
            source: id.to_string(),
            jp: id.to_string(),
            romaji: String::new(),
            notes: String::new(),
            jlpt: None,
            functions: Vec::new(),
            freq,
        }
    }

    fn kana(ch: &str) -> KanaItem {
        KanaItem {
            char: ch.to_string(),
            romaji: ch.to_string(),
            kana_type: KanaType::Hiragana,
            category: KanaCategory::Base,
        }
    }

    #[test]
    fn budget_rules() {
        // (name, due_reviews, lapsed_reviews, limit, want)
        let cases = [
            (
                "fresh profile: gentle intake, not a full-session dump",
                0,
                0,
                20,
                10,
            ),
            ("light day: full intake", 5, 0, 20, 10),
            ("light but lapse-heavy: half intake", 5, 3, 20, 5),
            ("seats left after reviews", 12, 2, 20, 8),
            ("seats left, halved by lapses", 12, 6, 20, 4),
            ("nearly full session of reviews", 18, 0, 20, 2),
            ("full backlog: reviews only", 20, 0, 20, 0),
            ("overloaded backlog: reviews only", 35, 10, 20, 0),
            ("all-lapsed small set: halved", 6, 6, 20, 5),
            ("uncapped queue still paces intake", 0, 0, 0, 10),
            ("uncapped and lapse-heavy still halves", 4, 2, 0, 5),
            ("session cap below the intake ceiling", 0, 0, 5, 5),
            ("reviews consume seats", 3, 0, 5, 2),
            ("reviews exceed the limit", 10, 0, 5, 0),
        ];
        for (name, due, lapsed, limit, want) in cases {
            assert_eq!(
                new_card_budget(due, lapsed, limit),
                want,
                "new_card_budget({due}, {lapsed}, {limit}) — {name}"
            );
        }
    }

    #[test]
    fn new_cards_are_due_and_capped_by_budget() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        // 15 fresh vocab cards; budget caps new intake at 10.
        let cards: Vec<Card> = (0..15).map(|i| card(&format!("l:{i}"), 0)).collect();
        let items = vocab_items(std::slice::from_ref(&lesson(cards)));
        let q = build_queue(&s, p.id, &items, Utc::now(), 0).unwrap();
        assert_eq!(q.items.len(), 10);
        assert_eq!(q.due_reviews, 0);
        assert_eq!(q.held_back_new, 5);
    }

    #[test]
    fn ranked_new_cards_admitted_first() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        // Card with rank 1 must be admitted before unranked ones.
        let mut cards = vec![card("l:rare", 0), card("l:common", 1)];
        cards.extend((0..12).map(|i| card(&format!("l:pad{i}"), 0)));
        let items = vocab_items(std::slice::from_ref(&lesson(cards)));
        let q = build_queue(&s, p.id, &items, Utc::now(), 0).unwrap();
        assert!(
            q.items.iter().any(|s| s.item.card_id == "l:common"),
            "the ranked card is admitted"
        );
    }

    #[test]
    fn interleaves_strands_round_robin() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let mut items = vocab_items(std::slice::from_ref(&lesson(vec![
            card("l:0", 0),
            card("l:1", 0),
        ])));
        items.extend(kana_items(&[kana("あ"), kana("い")]));
        let q = build_queue(&s, p.id, &items, Utc::now(), 0).unwrap();
        // Round-robin: strands alternate, so consecutive items differ in strand
        // at least once at the boundary.
        let strands: Vec<Strand> = q.items.iter().map(|s| s.item.strand).collect();
        assert!(strands.contains(&Strand::Vocab) && strands.contains(&Strand::Kana));
        assert_eq!(strands[0], Strand::Vocab); // Vocab sorts before Kana
        assert_eq!(strands[1], Strand::Kana); // interleaved next
    }

    #[test]
    fn reviewed_not_due_is_excluded() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        // Review a card so it schedules into the future, then it must not be due.
        let state = srs::review(&srs::new_card("l:0"), srs::Grade::Easy, Utc::now());
        s.save_card_state(p.id, &state).unwrap();
        let items = vocab_items(std::slice::from_ref(&lesson(vec![card("l:0", 0)])));
        let q = build_queue(&s, p.id, &items, Utc::now(), 0).unwrap();
        assert!(q.items.is_empty(), "a future-scheduled card is not due");
    }

    /// Never-seen items have no due date, so they are immediately due.
    #[test]
    fn new_items_are_due() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let items = [item("v:1", Strand::Vocab), item("v:2", Strand::Vocab)];
        let q = build_queue(&s, p.id, &items, Utc::now(), 0).unwrap();
        assert_eq!(q.items.len(), 2);
    }

    /// Items scheduled in the future are excluded; the due one survives.
    #[test]
    fn filters_not_due() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let now = at();
        save(&s, p.id, "v:1", now - Duration::hours(1)); // due
        save(&s, p.id, "v:2", now + Duration::hours(48)); // not due

        let items = [item("v:1", Strand::Vocab), item("v:2", Strand::Vocab)];
        let q = build_queue(&s, p.id, &items, now, 0).unwrap();
        assert_eq!(ids(&q), ["v:1"]);
    }

    /// Within a strand the queue is most-overdue first; across strands it is
    /// pulled round-robin.
    #[test]
    fn overdue_first_interleaved() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let now = at();
        let day = Duration::days(1);
        save(&s, p.id, "v:1", now - day * 3);
        save(&s, p.id, "v:2", now - day);
        save(&s, p.id, "v:3", now - day * 2);
        save(&s, p.id, "k:1", now - day * 5);
        save(&s, p.id, "k:2", now - day * 4);

        let items = [
            item("v:1", Strand::Vocab),
            item("v:2", Strand::Vocab),
            item("v:3", Strand::Vocab),
            item("k:1", Strand::Kana),
            item("k:2", Strand::Kana),
        ];
        let q = build_queue(&s, p.id, &items, now, 0).unwrap();
        assert_eq!(ids(&q), ["v:1", "k:1", "v:3", "k:2", "v:2"]);
    }

    #[test]
    fn respects_limit() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let items = [
            item("v:1", Strand::Vocab),
            item("v:2", Strand::Vocab),
            item("v:3", Strand::Vocab),
        ];
        let q = build_queue(&s, p.id, &items, Utc::now(), 2).unwrap();
        assert_eq!(q.items.len(), 2);
    }

    /// A fresh profile's all-new queue is paced: at most `MAX_NEW_PER_SESSION`
    /// enter, and the rest are reported as held back rather than dropped.
    #[test]
    fn paces_new_cards() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let items: Vec<Item> = (0..15)
            .map(|i| item(&format!("v:{i}"), Strand::Vocab))
            .collect();
        let q = build_queue(&s, p.id, &items, Utc::now(), 20).unwrap();
        assert_eq!(q.items.len(), 10, "new cards admitted");
        assert_eq!(q.held_back_new, 5);
        assert_eq!(q.due_reviews, 0);
    }

    /// Due reviews take every seat they need; new cards only fill what's left.
    #[test]
    fn reviews_take_priority_over_new() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let now = at();
        let mut items = Vec::new();
        for i in 0..18 {
            let id = format!("r:{i}");
            save(&s, p.id, &id, now - Duration::hours(1));
            items.push(item(&id, Strand::Vocab));
        }
        items.extend((0..6).map(|i| item(&format!("n:{i}"), Strand::Vocab)));

        let q = build_queue(&s, p.id, &items, now, 20).unwrap();
        assert_eq!(q.due_reviews, 18);
        // 18 reviews leave 2 seats: 2 new admitted, 4 held back.
        assert_eq!(q.items.len(), 20);
        assert_eq!(q.held_back_new, 4);
    }

    /// A lapse-heavy due set halves the new-card intake end-to-end.
    #[test]
    fn lapse_heavy_set_halves_intake() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let now = at();
        let mut items = Vec::new();
        for i in 0..4 {
            let id = format!("r:{i}");
            save_lapsed(&s, p.id, &id, now - Duration::hours(1));
            items.push(item(&id, Strand::Vocab));
        }
        items.extend((0..12).map(|i| item(&format!("n:{i}"), Strand::Vocab)));

        let q = build_queue(&s, p.id, &items, now, 20).unwrap();
        // 4 due reviews, all lapsed → budget 10 halved to 5.
        assert_eq!(q.items.len(), 9, "4 reviews + 5 new");
        assert_eq!(q.held_back_new, 7);
    }

    /// The budget cut prefers the most frequent words: ranked cards enter
    /// most-frequent-first, unranked ones only after them.
    #[test]
    fn admits_most_frequent_new_cards_first() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let items = [
            freq_item("unranked-a", 0),
            freq_item("rare", 900),
            freq_item("common", 14),
            freq_item("mid", 300),
            freq_item("unranked-b", 0),
        ];
        // limit 3, no due reviews → budget 3: the ranked cards, most frequent
        // first; both unranked cards held back behind them.
        let q = build_queue(&s, p.id, &items, Utc::now(), 3).unwrap();
        assert_eq!(ids(&q), ["common", "mid", "rare"]);
        assert_eq!(q.held_back_new, 2);
    }

    /// Strand interleaving still operates on the admitted set.
    #[test]
    fn interleaves_admitted_new_cards() {
        let s = mem_store();
        let p = s.create_profile("A").unwrap();
        let items = [
            item("v:1", Strand::Vocab),
            item("v:2", Strand::Vocab),
            item("k:1", Strand::Kana),
            item("k:2", Strand::Kana),
        ];
        let q = build_queue(&s, p.id, &items, Utc::now(), 20).unwrap();
        assert_eq!(ids(&q), ["v:1", "k:1", "v:2", "k:2"]);
    }

    #[test]
    fn kana_card_id_is_prefixed() {
        assert_eq!(kana_card_id(&kana("あ")), "kana:あ");
    }

    #[test]
    fn kana_items_map_fields() {
        let items = kana_items(&[KanaItem {
            char: "あ".to_string(),
            romaji: "a".to_string(),
            kana_type: KanaType::Hiragana,
            category: KanaCategory::Base,
        }]);
        assert_eq!(items.len(), 1);
        let it = &items[0];
        assert_eq!(it.prompt, "あ");
        assert_eq!(it.answer, "a");
        assert_eq!(it.strand, Strand::Kana);
    }
}
