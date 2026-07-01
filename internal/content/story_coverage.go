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

// checkStoryCoverage verifies every practice beat's RefID resolves to real
// content: an existing lesson (vocab) or a kana type with teachable items
// (kana), so no chapter depends on content that doesn't exist.
func checkStoryCoverage(c model.Chapter, lessonIDs map[string]bool, kanaTypes map[model.KanaType]bool) error {
	for i, b := range c.Beats {
		if b.Kind != model.Practice {
			continue
		}
		switch b.Practice {
		case model.PracticeVocab:
			if !lessonIDs[b.RefID] {
				return fmt.Errorf("beat %d: practice references unknown lesson id %q", i+1, b.RefID)
			}
		case model.PracticeKana:
			kt := model.KanaType(b.RefID)
			if !kt.Valid() || !kanaTypes[kt] {
				return fmt.Errorf("beat %d: practice references unknown or empty kana type %q", i+1, b.RefID)
			}
		}
	}
	return nil
}
