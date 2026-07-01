package model

// StoryProgress tracks a learner's position within one Katsudoo chapter: many
// rows per profile (one per chapter), the same idiom as KanaProgress and
// PatternProgress, so progress on multiple chapters can coexist as the story
// content grows.
type StoryProgress struct {
	ChapterID string
	BeatIndex int // index of the next beat to show; 0 = not yet started
	Completed bool
}
