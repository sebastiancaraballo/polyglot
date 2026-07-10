use std::io;
use std::path::PathBuf;

use include_dir::Dir;

/// A read-only content filesystem, abstracting over the embedded content bundle
/// and a real on-disk directory (used in tests). Paths are relative to the
/// content root (e.g. `"es-ja/lessons"`, `"ja/frequency.tsv"`) — the Rust port
/// drops the Go loader's leading `content/` segment, since each backend is
/// already rooted at `content/`.
pub trait ContentFs {
    /// Returns the sorted relative paths of every `*.yaml` file directly in
    /// `dir`. A missing directory yields an empty vec (not an error), so
    /// optional content dirs (grammar, story) simply contribute nothing.
    fn glob_yaml(&self, dir: &str) -> io::Result<Vec<String>>;

    /// Reads a file by its content-root-relative path. A missing file yields an
    /// [`io::ErrorKind::NotFound`] error, which the loader treats specially for
    /// the optional frequency list.
    fn read(&self, path: &str) -> io::Result<Vec<u8>>;
}

/// A [`ContentFs`] backed by the content bundle embedded into the binary.
pub struct EmbeddedFs<'a> {
    dir: &'a Dir<'a>,
}

impl<'a> EmbeddedFs<'a> {
    pub fn new(dir: &'a Dir<'a>) -> Self {
        EmbeddedFs { dir }
    }
}

impl ContentFs for EmbeddedFs<'_> {
    fn glob_yaml(&self, dir: &str) -> io::Result<Vec<String>> {
        let Some(d) = self.dir.get_dir(dir) else {
            return Ok(Vec::new());
        };
        let mut paths: Vec<String> = d
            .files()
            .filter(|f| f.path().extension().is_some_and(|e| e == "yaml"))
            .filter_map(|f| f.path().to_str().map(str::to_string))
            .collect();
        paths.sort();
        Ok(paths)
    }

    fn read(&self, path: &str) -> io::Result<Vec<u8>> {
        self.dir
            .get_file(path)
            .map(|f| f.contents().to_vec())
            .ok_or_else(|| io::Error::new(io::ErrorKind::NotFound, path.to_string()))
    }
}

/// A [`ContentFs`] backed by a real directory on disk, rooted at a `content/`
/// directory. Used by tests to load ad-hoc content fixtures.
pub struct DirFs {
    root: PathBuf,
}

impl DirFs {
    pub fn new(root: impl Into<PathBuf>) -> Self {
        DirFs { root: root.into() }
    }
}

impl ContentFs for DirFs {
    fn glob_yaml(&self, dir: &str) -> io::Result<Vec<String>> {
        let full = self.root.join(dir);
        let entries = match std::fs::read_dir(&full) {
            Ok(e) => e,
            Err(e) if e.kind() == io::ErrorKind::NotFound => return Ok(Vec::new()),
            Err(e) => return Err(e),
        };
        let mut paths = Vec::new();
        for entry in entries {
            let entry = entry?;
            let name = entry.file_name();
            let name = name.to_string_lossy();
            if name.ends_with(".yaml") {
                paths.push(format!("{dir}/{name}"));
            }
        }
        paths.sort();
        Ok(paths)
    }

    fn read(&self, path: &str) -> io::Result<Vec<u8>> {
        std::fs::read(self.root.join(path))
    }
}
