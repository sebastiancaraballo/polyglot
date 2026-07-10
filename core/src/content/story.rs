use std::collections::HashSet;

use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{Beat, BeatKind, Chapter, PracticeKind};

/// Mirrors the on-disk YAML shape of a story chapter.
#[derive(Deserialize)]
struct StoryFile {
    #[serde(default)]
    id: String,
    #[serde(default)]
    title: String,
    #[serde(default)]
    beats: Vec<BeatFile>,
}

/// Mirrors the on-disk YAML shape of a single beat.
#[derive(Deserialize)]
struct BeatFile {
    #[serde(default)]
    kind: String,
    #[serde(default)]
    speaker: String,
    #[serde(default)]
    place: String,
    #[serde(default, rename = "es")]
    source: String,
    #[serde(default)]
    jp: String,
    #[serde(default)]
    romaji: String,
    #[serde(default)]
    practice: String,
    #[serde(default)]
    ref_id: String,
}

/// Reads every story chapter for `pair`. Like grammar patterns, story content
/// is optional.
pub(super) fn load_chapters(fsys: &dyn ContentFs, pair: &str) -> Result<Vec<Chapter>, LoadError> {
    let dir = format!("{pair}/story");
    let files = fsys
        .glob_yaml(&dir)
        .map_err(|e| LoadError::new(format!("glob story: {e}")))?;

    let mut seen: HashSet<String> = HashSet::new();
    let mut chapters = Vec::new();
    for file in files {
        let c = parse_chapter(fsys, &file)?;
        if !seen.insert(c.id.clone()) {
            return Err(LoadError::new(format!(
                "{file}: duplicate chapter id {:?}",
                c.id
            )));
        }
        chapters.push(c);
    }
    Ok(chapters)
}

fn parse_chapter(fsys: &dyn ContentFs, file: &str) -> Result<Chapter, LoadError> {
    let data = fsys
        .read(file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    let sf: StoryFile = serde_yaml_ng::from_slice(&data)
        .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;

    if sf.id.is_empty() {
        return Err(LoadError::new(format!("{file}: missing chapter id")));
    }
    if sf.title.is_empty() {
        return Err(LoadError::new(format!("{file}: missing chapter title")));
    }
    if sf.beats.is_empty() {
        return Err(LoadError::new(format!("{file}: chapter has no beats")));
    }

    let mut beats = Vec::with_capacity(sf.beats.len());
    for (i, b) in sf.beats.iter().enumerate() {
        let beat =
            parse_beat(b).map_err(|e| LoadError::new(format!("{file}: beat {}: {e}", i + 1)))?;
        beats.push(beat);
    }

    Ok(Chapter {
        id: sf.id,
        title: sf.title,
        beats,
    })
}

fn parse_beat(b: &BeatFile) -> Result<Beat, String> {
    let Some(kind) = BeatKind::from_str(&b.kind) else {
        return Err(format!("invalid beat kind {:?}", b.kind));
    };

    let mut beat = Beat {
        kind,
        speaker: b.speaker.clone(),
        place: b.place.clone(),
        source: String::new(),
        jp: String::new(),
        romaji: String::new(),
        practice: None,
        ref_id: String::new(),
    };

    match kind {
        BeatKind::Narration | BeatKind::Dialogue => {
            if b.source.is_empty() || b.jp.is_empty() {
                return Err(format!("{} beat is missing es or jp", kind.as_str()));
            }
            if kind == BeatKind::Dialogue && b.speaker.is_empty() {
                return Err("dialogue beat is missing speaker".to_string());
            }
            beat.source = b.source.clone();
            beat.jp = b.jp.clone();
            beat.romaji = b.romaji.clone();
        }
        BeatKind::Present => {
            let Some(practice) = PracticeKind::from_str(&b.practice) else {
                return Err(format!("invalid practice kind {:?}", b.practice));
            };
            if b.ref_id.is_empty() {
                return Err("present beat is missing ref_id".to_string());
            }
            // The framing line is optional, but es/jp travel together.
            if b.source.is_empty() != b.jp.is_empty() {
                return Err("present beat framing needs both es and jp, or neither".to_string());
            }
            beat.practice = Some(practice);
            beat.ref_id = b.ref_id.clone();
            beat.source = b.source.clone();
            beat.jp = b.jp.clone();
            beat.romaji = b.romaji.clone();
        }
        BeatKind::Practice => {
            let Some(practice) = PracticeKind::from_str(&b.practice) else {
                return Err(format!("invalid practice kind {:?}", b.practice));
            };
            if b.ref_id.is_empty() {
                return Err("practice beat is missing ref_id".to_string());
            }
            if !b.source.is_empty()
                || !b.jp.is_empty()
                || !b.romaji.is_empty()
                || !b.speaker.is_empty()
            {
                return Err("practice beat must not carry dialogue fields".to_string());
            }
            beat.practice = Some(practice);
            beat.ref_id = b.ref_id.clone();
        }
    }
    Ok(beat)
}
