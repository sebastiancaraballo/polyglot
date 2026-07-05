package story

import (
	"fmt"
	"sort"
	"testing"

	"charm.land/lipgloss/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// TestEmbeddedStoryFitsFrame renders every beat (and every page of every present
// beat) of every embedded chapter at the real frame size and asserts nothing
// overflows the frame — width (long, space-less Japanese must wrap, not clip) or
// height. Practice beats are checked deterministically and exhaustively (every
// prompt gloss against the widest option set) rather than through the random
// draw, so the test is stable and covers the worst case, not a lucky one.
func TestEmbeddedStoryFitsFrame(t *testing.T) {
	course, err := content.LoadEmbedded(content.DefaultPair)
	if err != nil {
		t.Fatalf("LoadEmbedded: %v", err)
	}
	th := ui.PlainTheme()
	const termW, termH = 80, 30
	maxW := ui.FrameContentWidth(th, termW)
	maxH := ui.FrameContentHeight(th, termH)

	for _, showRomaji := range []bool{true, false} {
		deps := Deps{
			Theme:      th,
			Msgs:       i18n.ES,
			Chapters:   course.Chapters,
			Lessons:    course.Lessons,
			Kana:       course.Kana,
			ShowRomaji: showRomaji,
		}
		for ci := range course.Chapters {
			m := New(deps)
			m.width, m.height = termW, termH
			m = m.startChapter(ci)
			ch := course.Chapters[ci]
			for bi := range ch.Beats {
				m.beatIndex = bi
				m = m.enterBeat()
				beat := ch.Beats[bi]
				label := fmt.Sprintf("ch=%s beat=%d kind=%s romaji=%v", ch.ID, bi, beat.Kind, showRomaji)
				for _, v := range renderVariants(m, beat) {
					if w := lipgloss.Width(v.content); w > maxW {
						t.Errorf("[WIDTH] %s %s: widest line %d > frame %d", label, v.note, w, maxW)
					}
					if h := lipgloss.Height(v.content); h > maxH {
						t.Errorf("[HEIGHT] %s %s: %d lines > frame %d", label, v.note, h, maxH)
					}
				}
			}
		}
	}
}

type variant struct {
	note    string
	content string
}

// renderVariants returns every framed content string a beat can display. Present
// beats yield one per page; practice-vocab beats yield one per card (each card's
// gloss as the prompt, with the lesson's widest option labels shown), covering
// the worst case deterministically; everything else yields its single view.
func renderVariants(m Model, beat model.Beat) []variant {
	switch beat.Kind {
	case model.Present:
		pages := m.presentPages(beat)
		out := make([]variant, 0, len(pages))
		for pg := range pages {
			m.presentPage = pg
			out = append(out, variant{fmt.Sprintf("page=%d/%d", pg+1, len(pages)), m.presentView()})
		}
		return out
	case model.Practice:
		if beat.Practice != model.PracticeVocab {
			return []variant{{"", m.practiceView()}}
		}
		lesson := lessonByID(m.deps.Lessons, beat.RefID)
		if lesson == nil {
			return []variant{{"", m.practiceView()}}
		}
		m.options, m.correct = widestOptions(lesson.Cards, optionCount)
		out := make([]variant, 0, len(lesson.Cards))
		for _, c := range lesson.Cards {
			m.practiceCard = c
			out = append(out, variant{"card=" + c.ID, m.practiceView()})
		}
		return out
	default:
		return []variant{{"", m.beatView()}}
	}
}

// widestOptions picks the n cards whose Japanese forms render widest, so the
// option rows in a rendered question are the worst case for width.
func widestOptions(cards []model.Card, n int) ([]string, int) {
	sorted := append([]model.Card(nil), cards...)
	sort.SliceStable(sorted, func(i, j int) bool {
		return lipgloss.Width(sorted[i].JP) > lipgloss.Width(sorted[j].JP)
	})
	if n > len(sorted) {
		n = len(sorted)
	}
	opts := make([]string, 0, n)
	for _, c := range sorted[:n] {
		opts = append(opts, c.JP)
	}
	return opts, 0
}
