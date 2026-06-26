package model

import "time"

// Profile is a local learner. Multiple profiles can exist on the same machine,
// each with its own progress and statistics.
type Profile struct {
	ID        int64
	Name      string
	Onboarded bool
	// ShowRomaji controls whether romaji is displayed alongside Japanese in the
	// study screens. New profiles default to true.
	ShowRomaji bool
	// KanaOnboarded records whether the learner has seen the kana trainer's
	// first-time intro. New profiles default to false.
	KanaOnboarded bool
	CreatedAt     time.Time
}
