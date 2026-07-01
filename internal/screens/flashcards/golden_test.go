package flashcards

import (
	"testing"

	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func goldenModel() Model {
	vocab := review.Item{CardID: "greetings:1", Strand: review.Vocab, Prompt: "Gracias", Answer: "ありがとう", Secondary: "arigatou", Notes: "Fórmula básica de cortesía."}
	kana := review.Item{CardID: "kana:あ", Strand: review.Kana, Prompt: "あ", Answer: "a"}
	return Model{
		deps:  Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, ShowRomaji: true, Title: i18n.ES.ReviewScreenTitle},
		queue: []review.Scheduled{scheduled(vocab), scheduled(kana)},
	}
}

func TestPromptGolden(t *testing.T) {
	m := goldenModel()
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestRevealedVocabGolden(t *testing.T) {
	m := goldenModel()
	m.revealed = true
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestRevealedKanaGolden(t *testing.T) {
	m := goldenModel()
	m.index = 1
	m.revealed = true
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestNothingDueGolden(t *testing.T) {
	m := Model{deps: Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Title: i18n.ES.ReviewScreenTitle}}
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestSummaryGolden(t *testing.T) {
	m := goldenModel()
	m.index = 2
	m.reviewed = 2
	golden.RequireEqual(t, []byte(m.View().Content))
}

func TestHeldBackNoticeGolden(t *testing.T) {
	m := goldenModel()
	m.heldBackNew = 4
	golden.RequireEqual(t, []byte(m.View().Content))
}
