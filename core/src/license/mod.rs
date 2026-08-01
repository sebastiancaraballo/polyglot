//! Records and validates the provenance of every third-party asset bundled into
//! or derived into the repository, so the project's MIT promise holds for assets
//! as well as code (see CLAUDE.md "Hard constraints").
//!
//! Port of the Go `internal/license` package. The machine-readable manifest
//! (`assets.yaml`) is the single source of truth: the repo-root `NOTICE` file is
//! generated from it, and a test fails CI if any asset is missing provenance,
//! carries a non-permissive license, or if `NOTICE` has drifted.

use serde::Deserialize;

/// The embedded third-party asset manifest.
const MANIFEST_YAML: &str = include_str!("assets.yaml");

/// One third-party resource and its provenance. In-house authored content is
/// not represented here — it is covered by the repo's MIT LICENSE.
#[derive(Clone, Debug, Deserialize, PartialEq)]
pub struct Asset {
    #[serde(default)]
    pub id: String,
    #[serde(default)]
    pub name: String,
    /// `data | font | image | audio | generated`.
    #[serde(default)]
    pub kind: String,
    #[serde(default)]
    pub license: String,
    #[serde(default)]
    pub source: String,
    #[serde(default)]
    pub attribution: String,
    #[serde(default)]
    pub used_by: Vec<String>,
    #[serde(default)]
    pub notes: String,
}

#[derive(Deserialize)]
struct Manifest {
    #[serde(default)]
    assets: Vec<Asset>,
}

const VALID_KINDS: [&str; 5] = ["data", "font", "image", "audio", "generated"];

/// Parses and returns the embedded third-party asset manifest.
pub fn load() -> Result<Vec<Asset>, String> {
    let m: Manifest =
        serde_yaml_ng::from_str(MANIFEST_YAML).map_err(|e| format!("parse asset manifest: {e}"))?;
    Ok(m.assets)
}

impl Asset {
    /// Reports whether the asset records complete provenance and carries a
    /// license on Polyglot's permissive allowlist.
    pub fn validate(&self) -> Result<(), String> {
        if self.id.is_empty() {
            return Err("asset is missing id".to_string());
        }
        if self.name.is_empty() {
            return Err(format!("asset {:?} is missing name", self.id));
        }
        if !VALID_KINDS.contains(&self.kind.as_str()) {
            return Err(format!(
                "asset {:?} has invalid kind {:?}",
                self.id, self.kind
            ));
        }
        if self.license.is_empty() {
            return Err(format!("asset {:?} is missing license", self.id));
        }
        if self.source.is_empty() {
            return Err(format!("asset {:?} is missing source", self.id));
        }
        if self.attribution.is_empty() {
            return Err(format!("asset {:?} is missing attribution", self.id));
        }
        if self.used_by.is_empty() {
            return Err(format!("asset {:?} is missing used_by", self.id));
        }
        if !permitted_license(&self.license) {
            return Err(format!(
                "asset {:?}: license {:?} is not on the permissive allowlist",
                self.id, self.license
            ));
        }
        Ok(())
    }
}

/// The exact license identifiers always allowed.
const PERMITTED: [&str; 6] = [
    "Public Domain",
    "CC0-1.0",
    "MIT",
    "BSD-2-Clause",
    "BSD-3-Clause",
    "Apache-2.0",
];

/// Reports whether a license identifier is on Polyglot's permissive allowlist.
/// The CC-BY family is allowed only when it carries neither a NonCommercial
/// (-NC) nor a ShareAlike (-SA) restriction; copyleft (GPL/LGPL/AGPL) and
/// unknown licenses are rejected.
pub fn permitted_license(id: &str) -> bool {
    if PERMITTED.contains(&id) {
        return true;
    }
    let up = id.to_uppercase();
    if up.contains("-NC") || up.contains("-SA") {
        return false;
    }
    if up.starts_with("GPL") || up.starts_with("LGPL") || up.starts_with("AGPL") {
        return false;
    }
    up.starts_with("CC-BY-")
}

const NOTICE_HEADER: &str = "Polyglot
Copyright (c) 2026 Sebastián Caraballo

Polyglot is MIT-licensed (see LICENSE). This NOTICE lists the third-party assets
bundled in or derived into this repository, with their licenses and the
attributions those licenses require. In-house authored content is covered by the
repository's MIT license and is not listed here.

This file is GENERATED from internal/license/assets.yaml.
Regenerate it with: go run ./tools/gennotice
";

/// Returns the full, deterministic contents of the repo-root NOTICE file
/// describing every third-party asset in the manifest (sorted by id).
pub fn render_notice(assets: &[Asset]) -> String {
    let mut sorted: Vec<&Asset> = assets.iter().collect();
    sorted.sort_by(|a, b| a.id.cmp(&b.id));

    let mut out = String::from(NOTICE_HEADER);
    for a in sorted {
        out.push('\n');
        out.push_str(&"-".repeat(76));
        out.push('\n');
        out.push_str(&format!("{}\n", a.name));
        out.push_str(&format!("  License:     {}\n", a.license));
        out.push_str(&format!("  Source:      {}\n", a.source));
        out.push_str(&format!("  Attribution: {}\n", a.attribution));
        if !a.notes.is_empty() {
            out.push_str(&format!("  Notes:       {}\n", a.notes));
        }
    }
    out
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn manifest_is_valid_and_unique() {
        let assets = load().expect("manifest must parse");
        assert!(!assets.is_empty(), "manifest has at least one asset");
        let mut seen = std::collections::HashSet::new();
        for a in &assets {
            a.validate()
                .unwrap_or_else(|e| panic!("invalid asset: {e}"));
            assert!(seen.insert(a.id.clone()), "duplicate asset id {:?}", a.id);
        }
    }

    #[test]
    fn required_assets_are_registered() {
        let assets = load().unwrap();
        for want in ["tools/genglobe", "content/ja/frequency.tsv"] {
            assert!(
                assets.iter().any(|a| a.used_by.iter().any(|u| u == want)),
                "no manifest asset registered as used by {want:?}"
            );
        }
    }

    #[test]
    fn notice_in_sync_with_manifest() {
        let assets = load().unwrap();
        let want = render_notice(&assets);
        let path = concat!(env!("CARGO_MANIFEST_DIR"), "/../NOTICE");
        let got = std::fs::read_to_string(path).expect("read NOTICE");
        assert_eq!(
            got.replace("\r\n", "\n"),
            want.replace("\r\n", "\n"),
            "NOTICE is out of sync with the manifest"
        );
    }

    #[test]
    fn permitted_license_allowlist() {
        let cases = [
            ("Public Domain", true),
            ("CC0-1.0", true),
            ("MIT", true),
            ("BSD-2-Clause", true),
            ("BSD-3-Clause", true),
            ("Apache-2.0", true),
            ("CC-BY-4.0", true),
            ("CC-BY-2.0-FR", true),
            ("CC-BY-SA-4.0", false),
            ("CC-BY-NC-4.0", false),
            ("GPL-3.0", false),
            ("LGPL-2.1", false),
            ("AGPL-3.0", false),
            ("", false),
            ("Proprietary", false),
        ];
        for (id, want) in cases {
            assert_eq!(permitted_license(id), want, "{id:?}");
        }
    }
}
