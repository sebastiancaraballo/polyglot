package content

import (
	"fmt"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// lessonIDSet builds the set of every lesson ID, for validating a vocab
// practice beat's RefID.
func lessonIDSet(lessons []model.Lesson) map[string]bool {
	set := make(map[string]bool, len(lessons))
	for _, lesson := range lessons {
		set[lesson.ID] = true
	}
	return set
}

// kanaTypesPresent reports which kana types have at least one teachable item,
// for validating a kana practice beat's RefID.
func kanaTypesPresent(kana []model.KanaItem) map[model.KanaType]bool {
	set := make(map[model.KanaType]bool)
	for _, k := range kana {
		set[k.Type] = true
	}
	return set
}

// checkStoryCoverage verifies every present and practice beat's RefID resolves
// to real content: an existing lesson (vocab) or a kana type with teachable
// items (kana), so no chapter depends on content that doesn't exist.
func checkStoryCoverage(c model.Chapter, lessonIDs map[string]bool, kanaTypes map[model.KanaType]bool) error {
	for i, b := range c.Beats {
		if b.Kind != model.Practice && b.Kind != model.Present {
			continue
		}
		switch b.Practice {
		case model.PracticeVocab:
			if !lessonIDs[b.RefID] {
				return fmt.Errorf("beat %d: %s references unknown lesson id %q", i+1, b.Kind, b.RefID)
			}
		case model.PracticeKana:
			kt := model.KanaType(b.RefID)
			if !kt.Valid() || !kanaTypes[kt] {
				return fmt.Errorf("beat %d: %s references unknown or empty kana type %q", i+1, b.Kind, b.RefID)
			}
		}
	}
	return nil
}

// poolKey identifies the material pool a present or practice beat draws on, so
// presentation and practice of the same lesson/kana set compare equal.
func poolKey(b model.Beat) string {
	return string(b.Practice) + ":" + b.RefID
}

// checkStoryPresentation enforces present-before-practice: a chapter may only
// practice (and, via the end-of-chapter challenge, be tested on) a pool it —
// or an earlier chapter on the linear path — has already presented. Retrieval
// practice strengthens studied material (the testing effect), so quizzing a
// pool the learner has never met is a content bug, not a design. The rule is
// language-agnostic (like checkStoryCoverage): it operates on pool keys, not
// on any particular language's words.
//
// Chapters are validated in their embedded (mastery-gated) order, so a pool
// presented in chapter 1 counts as presented for chapter 2, which the learner
// can only reach after mastering chapter 1. The end-of-chapter challenge draws
// only from a chapter's practice-beat pools, so validating those covers it.
func checkStoryPresentation(chapters []model.Chapter) error {
	presentedOnPath := make(map[string]bool) // presented in an earlier chapter
	for _, c := range chapters {
		presentedHere := make(map[string]bool)
		for i, b := range c.Beats {
			switch b.Kind {
			case model.Present:
				presentedHere[poolKey(b)] = true
			case model.Practice:
				key := poolKey(b)
				if !presentedHere[key] && !presentedOnPath[key] {
					return fmt.Errorf("chapter %q: beat %d practices %q before it is presented; add a present beat for it earlier in this chapter or a prior one", c.ID, i+1, b.RefID)
				}
			}
		}
		for key := range presentedHere {
			presentedOnPath[key] = true
		}
	}
	return nil
}
