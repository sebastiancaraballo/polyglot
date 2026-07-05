package study

import (
	"math/rand"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// The N5 mock assessment: a cross-curriculum, end-of-level exam that samples
// across the three learnable strands — vocabulary, kana, and grammar patterns —
// and grades against the same 80% mastery band as the end-of-chapter challenge
// (ChallengePassed/ChallengeNeeded). It is a capstone retrieval-practice test,
// so — like the challenge — every answer flows through the regular
// spaced-repetition and XP paths, and a failed attempt is still learning.

// AssessmentLength is how many questions a full assessment asks when the
// curriculum can supply them: more than a 5-question chapter challenge, so an
// 80% bar is a meaningful level check.
const AssessmentLength = 15

// optionCount is the number of multiple-choice options per question (one
// correct answer plus up to three distractors), matching the study screens.
const optionCount = 4

// AssessKind is which strand an assessment question tests.
type AssessKind int

const (
	AssessVocab   AssessKind = iota // recall a word's Japanese form from its gloss
	AssessKana                      // read a kana character
	AssessPattern                   // fill a blanked slot in a grammar pattern
)

// AssessQuestion is one multiple-choice question. Its Options and Correct index
// are built at sample time (with the injected RNG) so the sampler is fully
// deterministic and unit-testable; the screen only renders them.
type AssessQuestion struct {
	Kind    AssessKind
	Card    model.Card        // AssessVocab: the prompt card; AssessPattern: the correct filler card
	Kana    model.KanaItem    // AssessKana
	Pattern model.Pattern     // AssessPattern
	SlotIdx int               // AssessPattern: index of the blanked slot in Pattern.Slots
	Fill    map[string]string // AssessPattern: the non-blank slots' names -> default JP
	Options []string          // the choice strings
	Correct int               // index of the correct option in Options
}

// key identifies the item a question tests, for deduplication across strands.
func (q AssessQuestion) key() string {
	switch q.Kind {
	case AssessKana:
		return "kana:" + q.Kana.Char
	case AssessPattern:
		return "pattern:" + q.Pattern.ID
	default:
		return "card:" + q.Card.ID
	}
}

// BuildAssessment draws up to AssessmentLength questions for level, sampled
// without replacement and round-robin across the three strands (vocab, kana,
// patterns) so every strand the curriculum supplies contributes. cards resolves
// a pattern slot's candidate/default card IDs back into full cards.
func BuildAssessment(rng *rand.Rand, level model.JLPT, lessons []model.Lesson, kana []model.KanaItem, patterns []model.Pattern, cards map[string]model.Card) []AssessQuestion {
	pools := [][]AssessQuestion{
		vocabQuestions(rng, level, lessons),
		kanaQuestions(rng, kana),
		patternQuestions(rng, level, patterns, cards),
	}

	seen := make(map[string]bool)
	var out []AssessQuestion
	for len(out) < AssessmentLength {
		progressed := false
		for i := range pools {
			if len(out) >= AssessmentLength {
				break
			}
			if len(pools[i]) == 0 {
				continue
			}
			q := pools[i][0]
			pools[i] = pools[i][1:]
			progressed = true
			if seen[q.key()] {
				continue
			}
			seen[q.key()] = true
			out = append(out, q)
		}
		if !progressed {
			break
		}
	}
	return out
}

// vocabQuestions builds a shuffled, capped pool of vocab questions from every
// card at the given level, with distractors drawn from that same card set.
func vocabQuestions(rng *rand.Rand, level model.JLPT, lessons []model.Lesson) []AssessQuestion {
	var cards []model.Card
	var pool []string
	for _, l := range lessons {
		if l.JLPT != level {
			continue
		}
		for _, c := range l.Cards {
			cards = append(cards, c)
			pool = append(pool, c.JP)
		}
	}
	rng.Shuffle(len(cards), func(i, j int) { cards[i], cards[j] = cards[j], cards[i] })
	if len(cards) > AssessmentLength {
		cards = cards[:AssessmentLength]
	}
	out := make([]AssessQuestion, 0, len(cards))
	for _, c := range cards {
		opts, correct := Options(rng, c.JP, pool, optionCount)
		out = append(out, AssessQuestion{Kind: AssessVocab, Card: c, Options: opts, Correct: correct})
	}
	return out
}

// kanaQuestions builds a shuffled, capped pool of kana-reading questions, with
// romaji distractors drawn from the whole syllabary set.
func kanaQuestions(rng *rand.Rand, kana []model.KanaItem) []AssessQuestion {
	items := append([]model.KanaItem(nil), kana...)
	pool := make([]string, 0, len(kana))
	for _, k := range kana {
		pool = append(pool, k.Romaji)
	}
	rng.Shuffle(len(items), func(i, j int) { items[i], items[j] = items[j], items[i] })
	if len(items) > AssessmentLength {
		items = items[:AssessmentLength]
	}
	out := make([]AssessQuestion, 0, len(items))
	for _, k := range items {
		opts, correct := Options(rng, k.Romaji, pool, optionCount)
		out = append(out, AssessQuestion{Kind: AssessKana, Kana: k, Options: opts, Correct: correct})
	}
	return out
}

// patternQuestions builds a shuffled, capped pool of grammar questions: one per
// pattern at the given level, blanking a random slot and offering that slot's
// candidate fillers as options. A slot with fewer than optionCount candidates
// yields a shorter option list (Options caps gracefully).
func patternQuestions(rng *rand.Rand, level model.JLPT, patterns []model.Pattern, cards map[string]model.Card) []AssessQuestion {
	var pats []model.Pattern
	for _, p := range patterns {
		if p.JLPT == level && len(p.Slots) > 0 {
			pats = append(pats, p)
		}
	}
	rng.Shuffle(len(pats), func(i, j int) { pats[i], pats[j] = pats[j], pats[i] })
	if len(pats) > AssessmentLength {
		pats = pats[:AssessmentLength]
	}
	out := make([]AssessQuestion, 0, len(pats))
	for _, p := range pats {
		slotIdx := rng.Intn(len(p.Slots))
		slot := p.Slots[slotIdx]
		correctCard, ok := cards[slot.Default]
		if !ok {
			continue
		}
		fill := make(map[string]string, len(p.Slots)-1)
		for i, s := range p.Slots {
			if i == slotIdx {
				continue
			}
			if c, ok := cards[s.Default]; ok {
				fill[s.Name] = c.JP
			}
		}
		pool := make([]string, 0, len(slot.CardIDs))
		for _, id := range slot.CardIDs {
			if c, ok := cards[id]; ok {
				pool = append(pool, c.JP)
			}
		}
		opts, correct := Options(rng, correctCard.JP, pool, optionCount)
		out = append(out, AssessQuestion{
			Kind: AssessPattern, Pattern: p, SlotIdx: slotIdx,
			Fill: fill, Card: correctCard, Options: opts, Correct: correct,
		})
	}
	return out
}
