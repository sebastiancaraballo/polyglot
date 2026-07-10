use super::fsys::ContentFs;
use super::LoadError;
use crate::model::FreqEntry;

/// Reads a target-language word-frequency list from `<lang>/frequency.tsv`.
pub(super) fn load_frequency(
    fsys: &dyn ContentFs,
    lang: &str,
) -> Result<Vec<FreqEntry>, LoadError> {
    let file = format!("{lang}/frequency.tsv");
    let data = fsys
        .read(&file)
        .map_err(|e| LoadError::new(format!("read {file}: {e}")))?;
    parse_frequency(&file, &data)
}

/// Parses a frequency list (`rank<TAB>word<TAB>reading<TAB>count`), skipping
/// blank lines and `#` comments.
pub(super) fn parse_frequency(file: &str, data: &[u8]) -> Result<Vec<FreqEntry>, LoadError> {
    let text = std::str::from_utf8(data)
        .map_err(|e| LoadError::new(format!("read {file}: invalid utf-8: {e}")))?;

    let mut entries = Vec::new();
    for (i, line) in text.lines().enumerate() {
        let lineno = i + 1;
        if line.is_empty() || line.starts_with('#') {
            continue;
        }
        let fields: Vec<&str> = line.split('\t').collect();
        if fields.len() != 4 {
            return Err(LoadError::new(format!(
                "{file}:{lineno}: expected 4 tab-separated fields, got {}",
                fields.len()
            )));
        }
        let rank: i64 = fields[0].parse().map_err(|e| {
            LoadError::new(format!(
                "{file}:{lineno}: invalid rank {:?}: {e}",
                fields[0]
            ))
        })?;
        let count: i64 = fields[3].parse().map_err(|e| {
            LoadError::new(format!(
                "{file}:{lineno}: invalid count {:?}: {e}",
                fields[3]
            ))
        })?;
        entries.push(FreqEntry {
            rank,
            word: fields[1].to_string(),
            reading: fields[2].to_string(),
            count,
        });
    }
    Ok(entries)
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn parses_and_skips_comments_and_blanks() {
        let tsv = "# comment\n1\t\u{79c1}\twatashi\t100\n\n2\t\u{306e}\tno\t80\n";
        let entries = parse_frequency("f.tsv", tsv.as_bytes()).unwrap();
        assert_eq!(entries.len(), 2);
        assert_eq!(entries[0].rank, 1);
        assert_eq!(entries[0].reading, "watashi");
        assert_eq!(entries[1].count, 80);
    }

    #[test]
    fn rejects_wrong_field_count() {
        let tsv = "1\tword\tonly-three\n";
        assert!(parse_frequency("f.tsv", tsv.as_bytes()).is_err());
    }
}
