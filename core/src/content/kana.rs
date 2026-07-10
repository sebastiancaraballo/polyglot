use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{KanaCategory, KanaItem, KanaType};

/// Mirrors the on-disk YAML shape of a kana table.
#[derive(Deserialize)]
struct KanaFile {
    #[serde(default, rename = "type")]
    kana_type: String,
    #[serde(default)]
    items: Vec<KanaEntry>,
}

#[derive(Deserialize)]
struct KanaEntry {
    #[serde(default)]
    char: String,
    #[serde(default)]
    romaji: String,
    #[serde(default)]
    category: String,
}

pub(super) fn load_kana(fsys: &dyn ContentFs, pair: &str) -> Result<Vec<KanaItem>, LoadError> {
    let dir = format!("{pair}/kana");
    let files = fsys
        .glob_yaml(&dir)
        .map_err(|e| LoadError::new(format!("glob kana: {e}")))?;

    let mut items: Vec<KanaItem> = Vec::new();
    for file in files {
        items.extend(parse_kana(fsys, &file)?);
    }
    if items.is_empty() {
        return Err(LoadError::new(format!("no kana found in {dir}")));
    }
    Ok(items)
}

fn parse_kana(fsys: &dyn ContentFs, file: &str) -> Result<Vec<KanaItem>, LoadError> {
    let data = fsys
        .read(file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    let kf: KanaFile = serde_yaml_ng::from_slice(&data)
        .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;

    let Some(kt) = KanaType::from_str(&kf.kana_type) else {
        return Err(LoadError::new(format!(
            "{file}: invalid kana type {:?}",
            kf.kana_type
        )));
    };
    if kf.items.is_empty() {
        return Err(LoadError::new(format!("{file}: no kana items")));
    }

    let mut items = Vec::with_capacity(kf.items.len());
    for (i, it) in kf.items.iter().enumerate() {
        if it.char.is_empty() || it.romaji.is_empty() {
            return Err(LoadError::new(format!(
                "{file}: item {} is missing char or romaji",
                i + 1
            )));
        }
        let category = if it.category.is_empty() {
            KanaCategory::Base
        } else {
            match KanaCategory::from_str(&it.category) {
                Some(c) => c,
                None => {
                    return Err(LoadError::new(format!(
                        "{file}: item {} has invalid category {:?}",
                        i + 1,
                        it.category
                    )))
                }
            }
        };
        items.push(KanaItem {
            char: it.char.clone(),
            romaji: it.romaji.clone(),
            kana_type: kt,
            category,
        });
    }
    Ok(items)
}
