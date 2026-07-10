use std::collections::HashSet;

use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{Card, FunctionCatalog, Jlpt, Lesson};

/// Mirrors the on-disk YAML shape of a lesson. The source-language key is `es`
/// for the v1 Spanish → Japanese pair.
#[derive(Deserialize)]
struct LessonFile {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    jlpt: String,
    #[serde(default)]
    functions: Vec<String>,
    #[serde(default)]
    cards: Vec<CardFile>,
}

#[derive(Deserialize)]
struct CardFile {
    #[serde(default, rename = "es")]
    source: String,
    #[serde(default)]
    jp: String,
    #[serde(default)]
    romaji: String,
    #[serde(default)]
    notes: String,
    #[serde(default)]
    freq: i64,
}

pub(super) fn load_lessons(
    fsys: &dyn ContentFs,
    pair: &str,
    catalog: &FunctionCatalog,
) -> Result<Vec<Lesson>, LoadError> {
    let dir = format!("{pair}/lessons");
    let files = fsys
        .glob_yaml(&dir)
        .map_err(|e| LoadError::new(format!("glob lessons: {e}")))?;

    let mut seen: HashSet<String> = HashSet::new();
    let mut lessons: Vec<Lesson> = Vec::with_capacity(files.len());
    for file in files {
        let lesson = parse_lesson(fsys, &file, catalog)?;
        if !seen.insert(lesson.id.clone()) {
            return Err(LoadError::new(format!(
                "{file}: duplicate lesson id {:?}",
                lesson.id
            )));
        }
        lessons.push(lesson);
    }
    if lessons.is_empty() {
        return Err(LoadError::new(format!("no lessons found in {dir}")));
    }
    Ok(lessons)
}

fn parse_lesson(
    fsys: &dyn ContentFs,
    file: &str,
    catalog: &FunctionCatalog,
) -> Result<Lesson, LoadError> {
    let data = fsys
        .read(file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    let lf: LessonFile = serde_yaml_ng::from_slice(&data)
        .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;

    if lf.id.is_empty() {
        return Err(LoadError::new(format!("{file}: missing lesson id")));
    }
    if lf.title.is_empty() {
        return Err(LoadError::new(format!("{file}: missing lesson title")));
    }
    let Some(level) = Jlpt::from_str(&lf.jlpt) else {
        return Err(LoadError::new(format!(
            "{file}: invalid jlpt level {:?}",
            lf.jlpt
        )));
    };
    for fname in &lf.functions {
        if !catalog.contains_key(fname) {
            return Err(LoadError::new(format!(
                "{file}: unknown function {fname:?}"
            )));
        }
    }
    if lf.cards.is_empty() {
        return Err(LoadError::new(format!("{file}: lesson has no cards")));
    }

    let mut cards = Vec::with_capacity(lf.cards.len());
    for (i, c) in lf.cards.iter().enumerate() {
        if c.source.is_empty() || c.jp.is_empty() || c.romaji.is_empty() {
            return Err(LoadError::new(format!(
                "{file}: card {} is missing es, jp, or romaji",
                i + 1
            )));
        }
        if c.freq < 0 {
            return Err(LoadError::new(format!(
                "{file}: card {} has negative freq {}",
                i + 1,
                c.freq
            )));
        }
        cards.push(Card {
            id: format!("{}:{}", lf.id, i + 1),
            source: c.source.clone(),
            jp: c.jp.clone(),
            romaji: c.romaji.clone(),
            notes: c.notes.clone(),
            jlpt: Some(level),
            functions: lf.functions.clone(),
            freq: c.freq,
        });
    }

    Ok(Lesson {
        id: lf.id,
        title: lf.title,
        jlpt: Some(level),
        functions: lf.functions,
        cards,
    })
}
