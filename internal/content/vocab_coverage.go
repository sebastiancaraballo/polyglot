package content

import (
	"fmt"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// cardIDSet builds the set of every vocab card ID across all lessons.
func cardIDSet(lessons []model.Lesson) map[string]bool {
	set := make(map[string]bool)
	for _, lesson := range lessons {
		for _, c := range lesson.Cards {
			set[c.ID] = true
		}
	}
	return set
}

// checkVocabCoverage verifies that every candidate card ID a pattern's slots
// reference resolves to an existing vocab card, so no pattern depends on a
// word the learner is never taught ("words before sentences" enforced at
// content-validation time, not just at runtime).
func checkVocabCoverage(p model.Pattern, cardSet map[string]bool) error {
	for _, slot := range p.Slots {
		for _, id := range slot.CardIDs {
			if !cardSet[id] {
				return fmt.Errorf("slot %q references unknown card id %q", slot.Name, id)
			}
		}
	}
	return nil
}
