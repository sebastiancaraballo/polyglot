// Package model defines the core domain types shared across Polyglot: cards,
// lessons, kana items, learner profiles, and progress.
//
// Curriculum content is split into a language-agnostic "spine" and per-language
// "skin". The spine is a catalog of communicative Functions (what a learner can
// do), each graded with a CEFR level. Per-language Lessons reference those
// functions and realize them with Cards, which carry their own JLPT level and an
// optional frequency rank.
package model
