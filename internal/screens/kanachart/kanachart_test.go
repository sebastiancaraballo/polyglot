package kanachart

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/exp/golden"

	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

func testKana() []model.KanaItem {
	return []model.KanaItem{
		{Char: "あ", Romaji: "a", Type: model.Hiragana, Category: model.Base},
		{Char: "か", Romaji: "ka", Type: model.Hiragana, Category: model.Base},
		{Char: "が", Romaji: "ga", Type: model.Hiragana, Category: model.Dakuten},
		{Char: "ぱ", Romaji: "pa", Type: model.Hiragana, Category: model.Handakuten},
		{Char: "きゃ", Romaji: "kya", Type: model.Hiragana, Category: model.Combo},
		{Char: "ア", Romaji: "a", Type: model.Katakana, Category: model.Base},
		{Char: "ガ", Romaji: "ga", Type: model.Katakana, Category: model.Dakuten},
		{Char: "キャ", Romaji: "kya", Type: model.Katakana, Category: model.Combo},
	}
}

func newTestModel() Model {
	m := New(Deps{Theme: ui.PlainTheme(), Msgs: i18n.ES, Kana: testKana()})
	m.width, m.height = 80, 40
	return m
}

func right() tea.KeyPressMsg { return tea.KeyPressMsg{Code: tea.KeyRight} }
func left() tea.KeyPressMsg  { return tea.KeyPressMsg{Code: tea.KeyLeft} }

func TestChartFirstPageGolden(t *testing.T) {
	golden.RequireEqual(t, []byte(newTestModel().View().Content))
}

func TestRightAdvancesAndClamps(t *testing.T) {
	m := newTestModel()
	if m.page != 0 {
		t.Fatalf("initial page = %d, want 0", m.page)
	}
	// Advance past the end; it should clamp at the last page.
	var tm tea.Model = m
	for i := 0; i < len(pageDefs)+3; i++ {
		tm, _ = tm.Update(right())
	}
	if got := tm.(Model).page; got != len(pageDefs)-1 {
		t.Fatalf("page after advancing to end = %d, want %d", got, len(pageDefs)-1)
	}
	// Go back past the start; it should clamp at 0.
	for i := 0; i < len(pageDefs)+3; i++ {
		tm, _ = tm.Update(left())
	}
	if got := tm.(Model).page; got != 0 {
		t.Fatalf("page after going back to start = %d, want 0", got)
	}
}

func TestPageShowsOnlyItsKana(t *testing.T) {
	m := newTestModel() // page 0: Hiragana base
	view := m.content()
	if !strings.Contains(view, "あ") || strings.Contains(view, "が") {
		t.Errorf("base page should show あ but not が:\n%s", view)
	}
	next, _ := m.Update(right()) // page 1: Hiragana dakuten/handakuten
	view = next.(Model).content()
	if !strings.Contains(view, "が") || !strings.Contains(view, "ぱ") {
		t.Errorf("voiced page should show が and ぱ:\n%s", view)
	}
}
