package study

import (
	"testing"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func decoderFor(mastered ...string) Decoder {
	items := []model.KanaItem{
		{Char: "こ", Type: model.Hiragana, Category: model.Base},
		{Char: "ん", Type: model.Hiragana, Category: model.Base},
		{Char: "に", Type: model.Hiragana, Category: model.Base},
		{Char: "ち", Type: model.Hiragana, Category: model.Base},
		{Char: "は", Type: model.Hiragana, Category: model.Base},
		{Char: "き", Type: model.Hiragana, Category: model.Base},
		{Char: "ゅ", Type: model.Hiragana, Category: model.Base},
		{Char: "う", Type: model.Hiragana, Category: model.Base},
		{Char: "きゅ", Type: model.Hiragana, Category: model.Combo},
		{Char: "が", Type: model.Hiragana, Category: model.Dakuten},
		{Char: "コ", Type: model.Katakana, Category: model.Base},
		{Char: "ヒ", Type: model.Katakana, Category: model.Base},
	}
	progress := make(map[string]model.KanaProgress)
	for _, c := range mastered {
		progress[c] = model.KanaProgress{Char: c, Mastered: true}
	}
	return NewDecoder(items, progress)
}

func TestDecodable(t *testing.T) {
	tests := []struct {
		name     string
		mastered []string
		jp       string
		want     bool
	}{
		{
			name:     "all kana mastered is decodable",
			mastered: []string{"こ", "ん", "に", "ち", "は"},
			jp:       "こんにちは",
			want:     true,
		},
		{
			name:     "one missing kana is not decodable",
			mastered: []string{"こ", "ん", "に", "ち"}, // missing は
			jp:       "こんにちは",
			want:     false,
		},
		{
			name:     "combo requires the combo itself, not its parts",
			mastered: []string{"き", "ゅ", "う"}, // parts mastered, きゅ not
			jp:       "きゅう",
			want:     false,
		},
		{
			name:     "combo mastered makes it decodable",
			mastered: []string{"きゅ", "う"},
			jp:       "きゅう",
			want:     true,
		},
		{
			name:     "empty or kana-less text is not decodable",
			mastered: []string{"こ"},
			jp:       "",
			want:     false,
		},
		{
			name:     "kanji is never decodable yet",
			mastered: []string{"こ", "ん", "に", "ち", "は"},
			jp:       "日本は", // 日本 are kanji
			want:     false,
		},
		{
			name:     "sokuon is transparent when the real kana are mastered",
			mastered: []string{"が", "こ", "う"},
			jp:       "がっこう", // small tsu っ has no reading of its own
			want:     true,
		},
		{
			name:     "sokuon does not excuse a missing kana",
			mastered: []string{"が", "こ"}, // missing う
			jp:       "がっこう",
			want:     false,
		},
		{
			name:     "chōonpu is transparent",
			mastered: []string{"コ", "ヒ"},
			jp:       "コーヒー", // long-vowel mark ー modifies its neighbor
			want:     true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := decoderFor(tt.mastered...)
			if got := d.Decodable(tt.jp); got != tt.want {
				t.Errorf("Decodable(%q) = %v, want %v", tt.jp, got, tt.want)
			}
		})
	}
}
