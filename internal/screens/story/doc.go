// Package story implements the Katsudoo runner: the communicative-activity
// strand, a sequence of narration/dialogue/present/practice beats grouped into
// chapters. A present beat diegetically introduces a pool of material (a
// vocabulary lesson or a kana set) so the learner meets it before a practice
// beat or the end-of-chapter challenge asks them to retrieve it — the loader
// enforces this present-before-practice rule at content-validation time.
// Practice beats reuse the same pure grading logic as the real kana trainer
// and quiz screens (study.GradeKana, srs.Review, the same storage calls) to
// run one inline check, rather than embedding those screens' Bubble Tea
// models.
package story
