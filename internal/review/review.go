// Package review builds cross-curriculum spaced-repetition study queues. It is
// the shared scheduler that turns the curriculum's learnable items — kana,
// vocabulary, and (later) grammar — into a single due-ordered, strand-interleaved
// session, so every study screen schedules review the same way instead of
// reimplementing the due/ordering logic. It is UI-free: it depends only on the
// content model, the SRS engine, and storage.
package review

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/srs"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
)

// Strand identifies which part of the curriculum an item belongs to. The queue
// interleaves strands so the learner practices them mixed rather than in blocks,
// which research on interleaving shows aids retention.
type Strand int

const (
	Vocab Strand = iota // vocabulary cards
	Kana                // kana (syllabary) characters
)

// Item is a single schedulable, renderable unit of study, independent of the
// curriculum strand it came from. The prompt is shown first; the answer (with an
// optional secondary line and notes) is revealed on demand.
type Item struct {
	CardID    string // stable key used for the card's scheduling state
	Strand    Strand
	Prompt    string // the question, shown first
	Answer    string // the answer, revealed on demand
	Secondary string // optional secondary line shown with the answer (e.g. romaji)
	Notes     string // optional usage notes
}

// Scheduled pairs an item with its current spaced-repetition state.
type Scheduled struct {
	Item  Item
	State model.CardState
}

// KanaCardID returns the stable scheduling key for a kana item. Kana characters
// are unique across the syllabaries, so the character alone identifies the card.
func KanaCardID(it model.KanaItem) string {
	return "kana:" + it.Char
}

// VocabItems turns every lesson's cards into review items, reusing each card's
// stable ID as the scheduling key.
func VocabItems(lessons []model.Lesson) []Item {
	var items []Item
	for _, lesson := range lessons {
		for _, c := range lesson.Cards {
			items = append(items, Item{
				CardID:    c.ID,
				Strand:    Vocab,
				Prompt:    c.Source,
				Answer:    c.JP,
				Secondary: c.Romaji,
				Notes:     c.Notes,
			})
		}
	}
	return items
}

// KanaItems turns kana characters into review items: the character is the prompt,
// its romaji reading the answer.
func KanaItems(kana []model.KanaItem) []Item {
	items := make([]Item, 0, len(kana))
	for _, k := range kana {
		items = append(items, Item{
			CardID: KanaCardID(k),
			Strand: Kana,
			Prompt: k.Char,
			Answer: k.Romaji,
		})
	}
	return items
}

// BuildQueue returns the items currently due for the profile, ordered most-overdue
// first within each strand and interleaved across strands, capped to at most limit
// items (limit <= 0 means no cap). It loads each item's scheduling state from the
// store, treating a never-seen item as a new card that is immediately due. The
// result is deterministic for a given input.
func BuildQueue(ctx context.Context, store storage.Storage, profileID int64, items []Item, now time.Time, limit int) ([]Scheduled, error) {
	var due []Scheduled
	for _, it := range items {
		state, err := store.GetCardState(ctx, profileID, it.CardID)
		switch {
		case errors.Is(err, storage.ErrNotFound):
			state = srs.NewCard(it.CardID)
		case err != nil:
			return nil, fmt.Errorf("review: load state for %q: %w", it.CardID, err)
		}
		if srs.IsDue(state, now) {
			due = append(due, Scheduled{Item: it, State: state})
		}
	}

	ordered := interleave(due)
	if limit > 0 && len(ordered) > limit {
		ordered = ordered[:limit]
	}
	return ordered, nil
}

// interleave orders due items most-overdue first within each strand, then pulls
// from the strands round-robin so the session mixes them.
func interleave(items []Scheduled) []Scheduled {
	buckets := make(map[Strand][]Scheduled)
	var strands []Strand
	for _, s := range items {
		if _, ok := buckets[s.Item.Strand]; !ok {
			strands = append(strands, s.Item.Strand)
		}
		buckets[s.Item.Strand] = append(buckets[s.Item.Strand], s)
	}
	sort.Slice(strands, func(i, j int) bool { return strands[i] < strands[j] })
	for _, st := range strands {
		b := buckets[st]
		sort.SliceStable(b, func(i, j int) bool { return b[i].State.DueAt.Before(b[j].State.DueAt) })
	}

	out := make([]Scheduled, 0, len(items))
	for {
		progressed := false
		for _, st := range strands {
			b := buckets[st]
			if len(b) == 0 {
				continue
			}
			out = append(out, b[0])
			buckets[st] = b[1:]
			progressed = true
		}
		if !progressed {
			break
		}
	}
	return out
}
