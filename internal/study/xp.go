package study

import "github.com/sebastiancaraballo/polyglot/internal/srs"

// OnboardingXP is the one-time bonus granted for completing onboarding.
const OnboardingXP = 20

// gradeXP maps each spaced-repetition grade to the experience points an answer
// earns. Every answer earns something so all interaction is rewarded, but more
// accurate recall earns more.
var gradeXP = map[srs.Grade]int{
	srs.Again: 2,
	srs.Hard:  6,
	srs.Good:  10,
	srs.Easy:  14,
}

// XPForGrade returns the experience points earned for an answer graded by the
// spaced-repetition scheduler.
func XPForGrade(grade srs.Grade) int {
	return gradeXP[grade]
}

// XPForAnswer is the correct/incorrect shorthand used by screens that only
// distinguish right from wrong (quiz, kana trainer).
func XPForAnswer(correct bool) int {
	if correct {
		return XPForGrade(srs.Good)
	}
	return XPForGrade(srs.Again)
}
