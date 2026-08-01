use std::collections::HashSet;

use crate::model::{
    is_han, Beat, BeatKind, Chapter, KanaItem, KanaType, KanjiItem, Lesson, Pattern, PracticeKind,
};

/// Builds the set of teachable kana strings from the loaded kana tables.
/// Combination kana (yōon) are stored as two-scalar strings, so the set
/// contains both single- and two-scalar entries.
pub(super) fn kana_set(items: &[KanaItem]) -> HashSet<String> {
    items.iter().map(|it| it.char.clone()).collect()
}

/// Builds the set of teachable kanji from the loaded kanji table.
pub(super) fn kanji_set(items: &[KanjiItem]) -> HashSet<String> {
    items.iter().map(|it| it.char.clone()).collect()
}

/// Fails when `jp` uses a kanji that is not in the kanji table.
///
/// This is the kanji counterpart of [`check_kana_coverage`], and it closes a
/// real gap: that check skips every non-kana character, so before this existed a
/// card written with kanji passed validation and then failed the decoding gate
/// forever — content that loads and is never shown.
pub(super) fn check_kanji_coverage(jp: &str, set: &HashSet<String>) -> Result<(), String> {
    for c in jp.chars() {
        if is_han(c) && !set.contains(&c.to_string()) {
            return Err(format!(
                "uses kanji {:?} not present in the kanji table",
                c.to_string()
            ));
        }
    }
    Ok(())
}

fn is_kana(c: char) -> bool {
    let u = c as u32;
    (0x3041..=0x309F).contains(&u) || (0x30A0..=0x30FF).contains(&u)
}

fn is_kana_mark(c: char) -> bool {
    matches!(c, 'っ' | 'ッ' | 'ー')
}

/// Verifies that every kana used in `jp` is present in `set`. It tokenizes with
/// longest match — a two-scalar combination (yōon) before a single scalar — so
/// combos like `きゅう` decompose as `きゅ` + `う`. Non-kana scalars (kanji,
/// ASCII, punctuation) are skipped.
pub(super) fn check_kana_coverage(jp: &str, set: &HashSet<String>) -> Result<(), String> {
    let runes: Vec<char> = jp.chars().collect();
    let mut i = 0;
    while i < runes.len() {
        let r = runes[i];
        if is_kana_mark(r) || !is_kana(r) {
            i += 1;
            continue;
        }
        if i + 1 < runes.len() {
            let pair: String = runes[i..i + 2].iter().collect();
            if set.contains(&pair) {
                i += 2;
                continue;
            }
        }
        if set.contains(&r.to_string()) {
            i += 1;
            continue;
        }
        return Err(format!(
            "uses kana {:?} not present in the kana tables",
            r.to_string()
        ));
    }
    Ok(())
}

/// Builds the set of every vocab card ID across all lessons.
pub(super) fn card_id_set(lessons: &[Lesson]) -> HashSet<String> {
    lessons
        .iter()
        .flat_map(|l| l.cards.iter().map(|c| c.id.clone()))
        .collect()
}

/// Verifies that every candidate card ID a pattern's slots reference resolves
/// to an existing vocab card ("words before sentences", at validation time).
pub(super) fn check_vocab_coverage(p: &Pattern, card_set: &HashSet<String>) -> Result<(), String> {
    for slot in &p.slots {
        for id in &slot.card_ids {
            if !card_set.contains(id) {
                return Err(format!(
                    "slot {:?} references unknown card id {id:?}",
                    slot.name
                ));
            }
        }
    }
    Ok(())
}

/// Builds the set of every lesson ID, for validating a vocab practice beat.
pub(super) fn lesson_id_set(lessons: &[Lesson]) -> HashSet<String> {
    lessons.iter().map(|l| l.id.clone()).collect()
}

/// Reports which kana types have at least one teachable item, for validating a
/// kana practice beat's ref_id.
pub(super) fn kana_types_present(kana: &[KanaItem]) -> HashSet<KanaType> {
    kana.iter().map(|k| k.kana_type).collect()
}

/// Verifies every present and practice beat's ref_id resolves to real content:
/// an existing lesson (vocab) or a kana type with teachable items (kana).
pub(super) fn check_story_coverage(
    c: &Chapter,
    lesson_ids: &HashSet<String>,
    kana_types: &HashSet<KanaType>,
) -> Result<(), String> {
    for (i, b) in c.beats.iter().enumerate() {
        if b.kind != BeatKind::Practice && b.kind != BeatKind::Present {
            continue;
        }
        match b.practice {
            Some(PracticeKind::Vocab) => {
                if !lesson_ids.contains(&b.ref_id) {
                    return Err(format!(
                        "beat {}: {} references unknown lesson id {:?}",
                        i + 1,
                        b.kind.as_str(),
                        b.ref_id
                    ));
                }
            }
            Some(PracticeKind::Kana) => {
                let ok = KanaType::from_str(&b.ref_id).is_some_and(|t| kana_types.contains(&t));
                if !ok {
                    return Err(format!(
                        "beat {}: {} references unknown or empty kana type {:?}",
                        i + 1,
                        b.kind.as_str(),
                        b.ref_id
                    ));
                }
            }
            None => {}
        }
    }
    Ok(())
}

/// Identifies the material pool a present or practice beat draws on, so
/// presentation and practice of the same pool compare equal.
fn pool_key(b: &Beat) -> String {
    let practice = b.practice.map(|p| p.as_str()).unwrap_or("");
    format!("{practice}:{}", b.ref_id)
}

/// Enforces present-before-practice: a chapter may only practice a pool it — or
/// an earlier chapter on the linear path — has already presented. Chapters are
/// validated in embedded (mastery-gated) order.
pub(super) fn check_story_presentation(chapters: &[Chapter]) -> Result<(), super::LoadError> {
    let mut presented_on_path: HashSet<String> = HashSet::new();
    for c in chapters {
        let mut presented_here: HashSet<String> = HashSet::new();
        for (i, b) in c.beats.iter().enumerate() {
            match b.kind {
                BeatKind::Present => {
                    presented_here.insert(pool_key(b));
                }
                BeatKind::Practice => {
                    let key = pool_key(b);
                    if !presented_here.contains(&key) && !presented_on_path.contains(&key) {
                        return Err(super::LoadError::new(format!(
                            "chapter {:?}: beat {} practices {:?} before it is presented; \
                             add a present beat for it earlier in this chapter or a prior one",
                            c.id,
                            i + 1,
                            b.ref_id
                        )));
                    }
                }
                _ => {}
            }
        }
        presented_on_path.extend(presented_here);
    }
    Ok(())
}

#[cfg(test)]
mod tests {
    use super::*;

    fn kset(items: &[&str]) -> HashSet<String> {
        items.iter().map(|s| s.to_string()).collect()
    }

    #[test]
    fn kana_coverage_longest_match() {
        let set = kset(&["き", "ゅ", "う", "きゅ"]);
        assert!(check_kana_coverage("きゅう", &set).is_ok()); // combo + う
        assert!(check_kana_coverage("がっこう", &kset(&["が", "こ", "う"])).is_ok()); // sokuon transparent
        assert!(check_kana_coverage("き", &kset(&["ゅ"])).is_err()); // missing き
        assert!(check_kana_coverage("日本", &set).is_ok()); // kanji skipped, no kana required to be present
    }

    /// The sokuon (っ/ッ) and the chōonpu (ー) modify a neighbor and have no
    /// item of their own, so words using them still pass coverage.
    #[test]
    fn kana_coverage_allows_kana_marks() {
        let course = crate::content::load_embedded(crate::content::DEFAULT_PAIR).unwrap();
        let set = kana_set(&course.kana);
        for jp in [
            "がっこう",
            "きって",
            "ざっし",
            "ちょっと",
            "コーヒー",
            "ノート",
            "スーパー",
        ] {
            check_kana_coverage(jp, &set).unwrap_or_else(|e| panic!("{jp:?}: {e}"));
        }
    }

    /// Retrieval practice only operates on material that was actually taught:
    /// a chapter may practice a pool only if it, or an earlier chapter on the
    /// linear path, already presented it.
    #[test]
    fn story_presentation_requires_presenting_first() {
        let present = Beat {
            kind: BeatKind::Present,
            speaker: String::new(),
            place: String::new(),
            source: String::new(),
            jp: String::new(),
            romaji: String::new(),
            practice: Some(PracticeKind::Vocab),
            ref_id: "greetings".to_string(),
        };
        let practice = Beat {
            kind: BeatKind::Practice,
            ..present.clone()
        };
        let narration = Beat {
            kind: BeatKind::Narration,
            speaker: String::new(),
            place: String::new(),
            source: "x".to_string(),
            jp: "あ".to_string(),
            romaji: String::new(),
            practice: None,
            ref_id: String::new(),
        };

        let chapter = |id: &str, beats: Vec<Beat>| Chapter {
            id: id.to_string(),
            title: "t".to_string(),
            beats,
        };

        let cases: Vec<(&str, Vec<Chapter>, bool)> = vec![
            (
                "present then practice in same chapter",
                vec![chapter("c1", vec![present.clone(), practice.clone()])],
                false,
            ),
            (
                "practice with no presentation",
                vec![chapter("c1", vec![practice.clone()])],
                true,
            ),
            (
                "present after practice does not count",
                vec![chapter("c1", vec![practice.clone(), present.clone()])],
                true,
            ),
            (
                "presented in a prior chapter",
                vec![
                    chapter("c1", vec![present.clone(), narration.clone()]),
                    chapter("c2", vec![practice.clone()]),
                ],
                false,
            ),
            (
                "presented only in a later chapter",
                vec![
                    chapter("c1", vec![practice.clone()]),
                    chapter("c2", vec![present.clone()]),
                ],
                true,
            ),
            (
                "no practice beats at all",
                vec![chapter("c1", vec![narration.clone()])],
                false,
            ),
        ];

        for (name, chapters, want_err) in cases {
            assert_eq!(
                check_story_presentation(&chapters).is_err(),
                want_err,
                "{name}"
            );
        }
    }
}
