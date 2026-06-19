package kana

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func TestSpaceAnswersKanaQuestion(t *testing.T) {
	m := Model{
		theme:   ui.NewTheme(true),
		msgs:    i18n.ES,
		deck:    []model.KanaItem{{Char: "あ", Romaji: "a", Type: model.Hiragana}},
		options: []string{"a"},
		correct: 0,
	}

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeySpace})
	got := next.(Model)
	if !got.answered {
		t.Fatal("space should answer the current kana question")
	}
}
