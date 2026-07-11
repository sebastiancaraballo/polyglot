//! Tiny helpers to substitute the Go-style `%s`/`%d` placeholders kept verbatim
//! in the i18n strings. The TUI interpolates them at render time.

/// Substitutes the first `%s`.
pub fn s(template: &str, a: &str) -> String {
    template.replacen("%s", a, 1)
}

/// Substitutes the first `%d`.
pub fn d(template: &str, a: i64) -> String {
    template.replacen("%d", &a.to_string(), 1)
}
