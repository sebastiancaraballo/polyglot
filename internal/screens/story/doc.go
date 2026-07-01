// Package story implements the Katsudoo runner: the communicative-activity
// strand, a sequence of narration/dialogue/practice beats grouped into
// chapters. Practice beats reuse the same pure grading logic as the real kana
// trainer and quiz screens (study.GradeKana, srs.Review, the same storage
// calls) to run one inline check, rather than embedding those screens'
// Bubble Tea models.
package story
