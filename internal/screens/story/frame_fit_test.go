package story

import (
	"fmt"
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
// height. Guards against authoring a beat whose prose is too long for the frame.
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
				pages := 1
				if ch.Beats[bi].Kind == model.Present {
					pages = len(m.presentPages(ch.Beats[bi]))
				}
				for pg := 0; pg < pages; pg++ {
					m.presentPage = pg
					label := fmt.Sprintf("ch=%s beat=%d kind=%s romaji=%v page=%d/%d",
						ch.ID, bi, ch.Beats[bi].Kind, showRomaji, pg+1, pages)
					content := beatContent(m)
					if w := lipgloss.Width(content); w > maxW {
						t.Errorf("[WIDTH] %s: widest line %d > frame %d", label, w, maxW)
					}
					if h := lipgloss.Height(content); h > maxH {
						t.Errorf("[HEIGHT] %s: %d lines > frame %d", label, h, maxH)
					}
				}
			}
		}
	}
}

// beatContent renders the pre-frame content for the current beat, mirroring the
// branch View() takes.
func beatContent(m Model) string {
	switch m.chapter.Beats[m.beatIndex].Kind {
	case model.Practice:
		return m.practiceView()
	case model.Present:
		return m.presentView()
	default:
		return m.beatView()
	}
}
