package model

// KanaProgress tracks a learner's progress toward *automaticity* on a single
// kana: fast, accurate recognition rather than effortful decoding. Mastery is
// earned by a run of correct, fast answers; a wrong or slow answer breaks the
// run. Once reached, Mastered stays set — long-term retention is the spaced
// repetition system's job, not the decoding gate's.
type KanaProgress struct {
	Char     string
	Streak   int  // current run of correct, fast answers
	Attempts int  // total answers seen
	Mastered bool // reached the automaticity threshold at least once
	BestMs   int  // fastest correct answer, in milliseconds (0 = none yet)
}
