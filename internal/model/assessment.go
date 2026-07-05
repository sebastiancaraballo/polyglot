package model

import "time"

// AssessmentResult records a profile's outcome on a level's mock assessment:
// one row per (profile, level). Passed is sticky — once a level is passed it is
// never revoked (Mastery Learning), the same idiom as StoryProgress.Mastered.
// BestCorrect/Total keep the learner's best score for the level.
type AssessmentResult struct {
	Level       JLPT
	Passed      bool
	BestCorrect int
	Total       int
	TakenAt     time.Time // when the assessment was last taken; zero if never
}
