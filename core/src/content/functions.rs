use serde::Deserialize;

use super::fsys::ContentFs;
use super::LoadError;
use crate::model::{Cefr, Function, FunctionCatalog};

/// Mirrors the on-disk YAML shape of a functions catalog file.
#[derive(Deserialize)]
struct FunctionsFile {
    #[serde(default)]
    functions: Vec<FunctionEntry>,
}

#[derive(Deserialize)]
struct FunctionEntry {
    #[serde(default)]
    id: String,
    #[serde(default)]
    cefr: String,
    #[serde(default)]
    description: String,
}

/// Reads and validates the language-agnostic communicative-function catalog
/// from `functions/*.yaml`. A missing or empty catalog is not an error here;
/// lessons that reference an unknown function are rejected during lesson
/// parsing.
pub(super) fn load_functions(fsys: &dyn ContentFs) -> Result<FunctionCatalog, LoadError> {
    let files = fsys
        .glob_yaml("functions")
        .map_err(|e| LoadError::new(format!("glob functions: {e}")))?;

    let mut catalog = FunctionCatalog::new();
    for file in files {
        let data = fsys
            .read(&file)
            .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
        let ff: FunctionsFile = serde_yaml_ng::from_slice(&data)
            .map_err(|e| LoadError::new(format!("parse {file}: {e}")))?;
        for (i, entry) in ff.functions.iter().enumerate() {
            if entry.id.is_empty() {
                return Err(LoadError::new(format!(
                    "{file}: function {} is missing id",
                    i + 1
                )));
            }
            if catalog.contains_key(&entry.id) {
                return Err(LoadError::new(format!(
                    "{file}: duplicate function id {:?}",
                    entry.id
                )));
            }
            let Some(level) = Cefr::from_str(&entry.cefr) else {
                return Err(LoadError::new(format!(
                    "{file}: function {:?} has invalid cefr level {:?}",
                    entry.id, entry.cefr
                )));
            };
            if entry.description.is_empty() {
                return Err(LoadError::new(format!(
                    "{file}: function {:?} is missing description",
                    entry.id
                )));
            }
            catalog.insert(
                entry.id.clone(),
                Function {
                    id: entry.id.clone(),
                    cefr: level,
                    description: entry.description.clone(),
                },
            );
        }
    }
    Ok(catalog)
}
