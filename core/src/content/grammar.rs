use std::collections::HashSet;

use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{Jlpt, Pattern, Slot};

/// Mirrors the on-disk YAML shape of a grammar pattern.
#[derive(Deserialize)]
struct PatternFile {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    jlpt: String,
    #[serde(default)]
    frame: String,
    #[serde(default)]
    notes: String,
    #[serde(default)]
    slots: Vec<SlotFile>,
}

#[derive(Deserialize)]
struct SlotFile {
    #[serde(default)]
    name: String,
    #[serde(default)]
    cards: Vec<String>,
    #[serde(default)]
    default: String,
}

/// Reads every grammar pattern for `pair`. Unlike lessons and kana, grammar
/// content is optional: a pair with no `grammar` directory yet is not an error.
pub(super) fn load_patterns(fsys: &dyn ContentFs, pair: &str) -> Result<Vec<Pattern>, LoadError> {
    let dir = format!("{pair}/grammar");
    let files = fsys
        .glob_yaml(&dir)
        .map_err(|e| LoadError::new(format!("glob grammar: {e}")))?;

    let mut seen: HashSet<String> = HashSet::new();
    let mut patterns = Vec::new();
    for file in files {
        let p = parse_pattern(fsys, &file)?;
        if !seen.insert(p.id.clone()) {
            return Err(LoadError::new(format!(
                "{file}: duplicate pattern id {:?}",
                p.id
            )));
        }
        patterns.push(p);
    }
    Ok(patterns)
}

fn parse_pattern(fsys: &dyn ContentFs, file: &str) -> Result<Pattern, LoadError> {
    let data = fsys
        .read(file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    let pf: PatternFile = serde_yaml_ng::from_slice(&data)
        .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;

    if pf.id.is_empty() {
        return Err(LoadError::new(format!("{file}: missing pattern id")));
    }
    if pf.title.is_empty() {
        return Err(LoadError::new(format!("{file}: missing pattern title")));
    }
    if pf.frame.is_empty() {
        return Err(LoadError::new(format!("{file}: missing pattern frame")));
    }
    let Some(level) = Jlpt::from_str(&pf.jlpt) else {
        return Err(LoadError::new(format!(
            "{file}: invalid jlpt level {:?}",
            pf.jlpt
        )));
    };
    if pf.slots.is_empty() {
        return Err(LoadError::new(format!("{file}: pattern has no slots")));
    }

    let mut slot_names: HashSet<String> = HashSet::new();
    let mut slots = Vec::with_capacity(pf.slots.len());
    for (i, s) in pf.slots.iter().enumerate() {
        if s.name.is_empty() {
            return Err(LoadError::new(format!(
                "{file}: slot {} is missing a name",
                i + 1
            )));
        }
        if !slot_names.insert(s.name.clone()) {
            return Err(LoadError::new(format!(
                "{file}: duplicate slot name {:?}",
                s.name
            )));
        }
        if s.cards.is_empty() {
            return Err(LoadError::new(format!(
                "{file}: slot {:?} has no candidate cards",
                s.name
            )));
        }
        let def = if s.default.is_empty() {
            s.cards[0].clone()
        } else if !s.cards.contains(&s.default) {
            return Err(LoadError::new(format!(
                "{file}: slot {:?} default {:?} is not among its candidate cards",
                s.name, s.default
            )));
        } else {
            s.default.clone()
        };
        slots.push(Slot {
            name: s.name.clone(),
            card_ids: s.cards.clone(),
            default: def,
        });
    }

    check_frame_placeholders(&pf.frame, &slot_names)
        .map_err(|e| LoadError::new(format!("{file}: {e}")))?;

    Ok(Pattern {
        id: pf.id,
        title: pf.title,
        jlpt: Some(level),
        frame: pf.frame,
        slots,
        notes: pf.notes,
    })
}

/// Verifies the frame's `{name}` placeholders are exactly the declared slot
/// names: no undeclared placeholder, and no declared slot left unused.
fn check_frame_placeholders(frame: &str, slot_names: &HashSet<String>) -> Result<(), String> {
    let mut found: HashSet<String> = HashSet::new();
    for name in find_placeholders(frame) {
        if !slot_names.contains(&name) {
            return Err(format!("frame references undeclared slot {name:?}"));
        }
        found.insert(name);
    }
    for name in slot_names {
        if !found.contains(name) {
            return Err(format!(
                "slot {name:?} is declared but never used in the frame"
            ));
        }
    }
    Ok(())
}

/// Finds every `{name}` placeholder in `frame`, where `name` matches
/// `[A-Za-z][A-Za-z0-9_]*`. Braces are ASCII, so scanning bytes is safe within
/// UTF-8 (the frame may contain Japanese around the placeholders).
fn find_placeholders(frame: &str) -> Vec<String> {
    let bytes = frame.as_bytes();
    let mut names = Vec::new();
    let mut i = 0;
    while i < bytes.len() {
        if bytes[i] == b'{' {
            let mut j = i + 1;
            if j < bytes.len() && bytes[j].is_ascii_alphabetic() {
                j += 1;
                while j < bytes.len() && (bytes[j].is_ascii_alphanumeric() || bytes[j] == b'_') {
                    j += 1;
                }
                if j < bytes.len() && bytes[j] == b'}' {
                    names.push(frame[i + 1..j].to_string());
                    i = j + 1;
                    continue;
                }
            }
        }
        i += 1;
    }
    names
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn placeholders_extracted() {
        assert_eq!(find_placeholders("{X}は{N}です"), vec!["X", "N"]);
        assert_eq!(find_placeholders("no slots"), Vec::<String>::new());
        assert_eq!(find_placeholders("{1bad}"), Vec::<String>::new()); // must start with a letter
    }

    /// Every embedded pattern's frame and slot declarations agree: each
    /// placeholder has a slot and each slot appears in the frame.
    #[test]
    fn embedded_frames_match_their_slots() {
        let course = crate::content::load_embedded(crate::content::DEFAULT_PAIR).unwrap();
        assert!(!course.patterns.is_empty());
        for p in &course.patterns {
            let names: HashSet<String> = p.slots.iter().map(|s| s.name.clone()).collect();
            check_frame_placeholders(&p.frame, &names)
                .unwrap_or_else(|e| panic!("pattern {:?}: {e}", p.id));
        }
    }
}
