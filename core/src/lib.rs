//! Polyglot core: the language-agnostic learning engine.
//!
//! This crate is the Rust port of the original Go `internal/*` engine packages
//! (`model`, `srs`, `study`, `content`, `review`, `story`, `storage`,
//! `license`, `i18n`). It is UI-free: terminal, mobile, and desktop frontends
//! consume it (natively as a crate, or via FFI bindings).

pub mod content;
pub mod i18n;
pub mod license;
pub mod model;
pub mod review;
pub mod srs;
pub mod storage;
pub mod study;
