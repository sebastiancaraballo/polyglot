use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{Jlpt, KanjiItem};

/// Mirrors the on-disk YAML shape of a kanji table.
#[derive(Deserialize)]
struct KanjiFile {
    #[serde(default)]
    items: Vec<KanjiEntry>,
}

#[derive(Deserialize)]
struct KanjiEntry {
    #[serde(default)]
    char: String,
    #[serde(default)]
    on: Vec<String>,
    #[serde(default)]
    kun: Vec<String>,
    #[serde(default)]
    meaning: String,
    #[serde(default)]
    jlpt: String,
}

/// Loads every kanji table for `pair` from `<pair>/kanji/*.yaml`.
///
/// The directory is **optional**, like grammar and story: a pair that teaches no
/// kanji simply has none, and every pair shipped today is in that state.
pub(super) fn load_kanji(fsys: &dyn ContentFs, pair: &str) -> Result<Vec<KanjiItem>, LoadError> {
    let dir = format!("{pair}/kanji");
    let files = fsys
        .glob_yaml(&dir)
        .map_err(|e| LoadError::new(format!("glob kanji: {e}")))?;

    let mut items: Vec<KanjiItem> = Vec::new();
    let mut seen: std::collections::HashSet<String> = std::collections::HashSet::new();
    for file in files {
        for item in parse_kanji(fsys, &file)? {
            if !seen.insert(item.char.clone()) {
                return Err(LoadError::new(format!(
                    "{file}: duplicate kanji {:?}",
                    item.char
                )));
            }
            items.push(item);
        }
    }
    Ok(items)
}

fn parse_kanji(fsys: &dyn ContentFs, file: &str) -> Result<Vec<KanjiItem>, LoadError> {
    let data = fsys
        .read(file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    let kf: KanjiFile = serde_yaml_ng::from_slice(&data)
        .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;

    if kf.items.is_empty() {
        return Err(LoadError::new(format!("{file}: no kanji items")));
    }

    let mut items = Vec::with_capacity(kf.items.len());
    for (i, it) in kf.items.iter().enumerate() {
        let n = i + 1;
        // Exactly one character: a multi-character entry would silently never
        // match anything the decoder looks up.
        let mut chars = it.char.chars();
        let (Some(c), None) = (chars.next(), chars.next()) else {
            return Err(LoadError::new(format!(
                "{file}: item {n} must be exactly one character, got {:?}",
                it.char
            )));
        };
        if !crate::model::is_han(c) {
            return Err(LoadError::new(format!(
                "{file}: item {n} {:?} is not a kanji",
                it.char
            )));
        }
        if it.on.is_empty() && it.kun.is_empty() {
            return Err(LoadError::new(format!(
                "{file}: item {n} {:?} has no readings",
                it.char
            )));
        }
        for r in it.on.iter().chain(it.kun.iter()) {
            if !valid_reading(r) {
                return Err(LoadError::new(format!(
                    "{file}: item {n} {:?} has invalid reading {r:?} \
                     (kana, with okurigana in a trailing parenthesized group)",
                    it.char
                )));
            }
        }
        if it.meaning.trim().is_empty() {
            return Err(LoadError::new(format!(
                "{file}: item {n} {:?} has no meaning",
                it.char
            )));
        }
        let jlpt = if it.jlpt.is_empty() {
            None
        } else {
            match Jlpt::from_str(&it.jlpt) {
                Some(l) => Some(l),
                None => {
                    return Err(LoadError::new(format!(
                        "{file}: item {n} {:?} has invalid jlpt {:?}",
                        it.char, it.jlpt
                    )))
                }
            }
        };
        items.push(KanjiItem {
            char: it.char.clone(),
            on: it.on.clone(),
            kun: it.kun.clone(),
            meaning: it.meaning.clone(),
            jlpt,
        });
    }
    Ok(items)
}

/// A reading is kana, optionally ending in one parenthesized okurigana group:
/// `みず`, `た(べる)`, `あたら(しい)`. The parentheses mark the inflectional
/// ending written in kana after the kanji — the dictionary convention — so a
/// stem is never shown bare and unpronounceable.
fn valid_reading(r: &str) -> bool {
    #[derive(PartialEq)]
    enum St {
        Stem,
        Okurigana,
        Closed,
    }
    let is_kana = |c: char| {
        let u = c as u32;
        (0x3040..=0x309F).contains(&u) || (0x30A0..=0x30FF).contains(&u)
    };
    let mut st = St::Stem;
    let mut saw_kana = false;
    let mut group_has_kana = false;
    for c in r.chars() {
        match (c, &st) {
            ('(', St::Stem) if saw_kana => st = St::Okurigana,
            (')', St::Okurigana) if group_has_kana => st = St::Closed,
            (_, St::Stem) if is_kana(c) => saw_kana = true,
            (_, St::Okurigana) if is_kana(c) => group_has_kana = true,
            _ => return false, // non-kana, nested/unopened paren, or text after ')'
        }
    }
    saw_kana && st != St::Okurigana
}
