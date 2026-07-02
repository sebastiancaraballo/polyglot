package content

import "github.com/sebastiancaraballo/polyglot/internal/model"

// Frequency backfill: map each vocabulary card to its target-language
// frequency rank so sequencing can be frequency-driven (Nation: learn the
// most frequent words first). Lesson and card *authoring* order stays
// curricular; the rank drives the order in which new cards enter study
// sessions, and is surfaced in the UI.

// freqIndex builds a lookup from a frequency list: surface word first, then
// reading for entries whose surface form (e.g. kanji 私) can't match our
// kana-only cards. The lowest (most frequent) rank wins on collisions.
//
// Known limitation, accepted: kana homographs may borrow a more frequent
// homograph's rank (e.g. the card に "dos" matches the particle に). That
// only nudges a card earlier in the intake order; an explicit freq: in the
// lesson YAML overrides the backfill entirely.
func freqIndex(entries []model.FreqEntry) map[string]int {
	index := make(map[string]int, len(entries)*2)
	for _, e := range entries {
		if _, ok := index[e.Word]; !ok {
			index[e.Word] = e.Rank
		}
	}
	for _, e := range entries {
		if e.Reading == "" {
			continue
		}
		if _, ok := index[e.Reading]; !ok {
			index[e.Reading] = e.Rank
		}
	}
	return index
}

// backfillFreq fills each card's Freq from the index. A rank set explicitly
// in the lesson YAML wins; words not in the list (including multi-token
// expressions the tokenizer never emits as one word) stay at 0 = unranked
// and sort after ranked cards wherever the rank is consumed.
func backfillFreq(lessons []model.Lesson, index map[string]int) {
	for li := range lessons {
		for ci := range lessons[li].Cards {
			card := &lessons[li].Cards[ci]
			if card.Freq != 0 {
				continue
			}
			if rank, ok := index[card.JP]; ok {
				card.Freq = rank
			}
		}
	}
}
