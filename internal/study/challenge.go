package study

import (
	"math/rand"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// The end-of-chapter challenge: a short retrieval-practice quiz over the
// material a chapter introduced, gating progression to the next chapter.
// Mastery Learning (Bloom): advance on demonstrated mastery, not on having
// clicked through. Retrieval practice itself strengthens retention (the
// testing effect, Roediger & Karpicke), so a failed attempt is still
// learning — retries are immediate and each answer is graded through the
// same spaced-repetition paths as regular practice.

// ChallengeLength is how many retrieval questions an end-of-chapter challenge
// asks, when the chapter's pools are large enough to supply them.
const ChallengeLength = 5

// ChallengeQuestion is one retrieval-practice question drawn from a chapter's
// practice-beat pools. It mirrors a practice Beat's shape so the story runner
// reuses the exact same question-building and grading paths.
type ChallengeQuestion struct {
	Practice model.PracticeKind
	RefID    string         // the pool it was drawn from: a Lesson.ID or kana type
	Card     model.Card     // set when Practice == PracticeVocab
	Kana     model.KanaItem // set when Practice == PracticeKana
}

// BuildChallenge draws up to ChallengeLength questions for the chapter,
// sampled without replacement, round-robin across the chapter's distinct
// practice pools (in beat order) so every referenced pool contributes.
// Returns nil when the chapter has no practice beats: a chapter that
// evaluates nothing has nothing to gate on, so completing it counts as
// mastering it. With pools smaller than ChallengeLength the challenge is
// shorter and the 80% criterion effectively requires all answers correct.
func BuildChallenge(rng *rand.Rand, chapter model.Chapter, lessons []model.Lesson, kana []model.KanaItem) []ChallengeQuestion {
	pools := challengePools(chapter, lessons, kana)
	if len(pools) == 0 {
		return nil
	}
	for _, p := range pools {
		rng.Shuffle(len(p.questions), func(i, j int) {
			p.questions[i], p.questions[j] = p.questions[j], p.questions[i]
		})
	}

	var out []ChallengeQuestion
	for len(out) < ChallengeLength {
		progressed := false
		for _, p := range pools {
			if len(out) >= ChallengeLength {
				break
			}
			if len(p.questions) == 0 {
				continue
			}
			out = append(out, p.questions[0])
			p.questions = p.questions[1:]
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return out
}

// pool is one practice beat's question source, deduplicated across the chapter.
type pool struct {
	questions []ChallengeQuestion
}

// challengePools builds one pool per distinct practice reference in beat
// order, deduplicating individual items (by card ID / kana char) so the same
// question can never be drawn twice even when pools overlap.
func challengePools(chapter model.Chapter, lessons []model.Lesson, kana []model.KanaItem) []*pool {
	seenRef := make(map[string]bool)
	seenItem := make(map[string]bool)
	var pools []*pool
	for _, beat := range chapter.Beats {
		if beat.Kind != model.Practice || seenRef[beat.RefID] {
			continue
		}
		seenRef[beat.RefID] = true

		p := &pool{}
		switch beat.Practice {
		case model.PracticeVocab:
			for _, lesson := range lessons {
				if lesson.ID != beat.RefID {
					continue
				}
				for _, c := range lesson.Cards {
					if seenItem["card:"+c.ID] {
						continue
					}
					seenItem["card:"+c.ID] = true
					p.questions = append(p.questions, ChallengeQuestion{
						Practice: model.PracticeVocab, RefID: beat.RefID, Card: c,
					})
				}
			}
		case model.PracticeKana:
			for _, k := range kana {
				if k.Type != model.KanaType(beat.RefID) || seenItem["kana:"+k.Char] {
					continue
				}
				seenItem["kana:"+k.Char] = true
				p.questions = append(p.questions, ChallengeQuestion{
					Practice: model.PracticeKana, RefID: beat.RefID, Kana: k,
				})
			}
		}
		if len(p.questions) > 0 {
			pools = append(pools, p)
		}
	}
	return pools
}

// ChallengePassed applies the mastery criterion: at least 80% correct —
// Bloom's mastery band, expressed as a ratio so short challenges from small
// pools stay principled (below 5 questions it effectively requires them all).
func ChallengePassed(correct, total int) bool {
	return total > 0 && correct*5 >= total*4
}

// ChallengeNeeded returns the minimum correct answers that pass a challenge
// of total questions: the smallest n with ChallengePassed(n, total). The UI
// states it up front so the mastery bar is never hidden logic.
func ChallengeNeeded(total int) int {
	return (total*4 + 4) / 5
}
