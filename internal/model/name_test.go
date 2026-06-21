package model

import (
	"errors"
	"strings"
	"testing"
)

func TestNormalizeNameValid(t *testing.T) {
	cases := map[string]string{
		"Sebastián":                     "Sebastián",
		"  José Niño  ":                 "José Niño",                     // trimmed
		"李":                             "李",                             // CJK, single letter
		"Анна":                          "Анна",                          // Cyrillic
		"O'Brien":                       "O'Brien",                       // apostrophe
		"Jean-Luc":                      "Jean-Luc",                      // hyphen
		strings.Repeat("a", MaxNameLen): strings.Repeat("a", MaxNameLen), // at the limit
	}
	for in, want := range cases {
		got, err := NormalizeName(in)
		if err != nil {
			t.Errorf("NormalizeName(%q) returned error %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("NormalizeName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNormalizeNameInvalid(t *testing.T) {
	cases := map[string]error{
		"":                                ErrEmptyName,
		"   ":                             ErrEmptyName,
		strings.Repeat("a", MaxNameLen+1): ErrNameTooLong,
		"123":                             ErrInvalidName, // no letter
		"ab\ncd":                          ErrInvalidName, // control char
		"🎉":                               ErrInvalidName, // emoji, no letter
	}
	for in, want := range cases {
		if _, err := NormalizeName(in); !errors.Is(err, want) {
			t.Errorf("NormalizeName(%q) error = %v, want %v", in, err, want)
		}
	}
}
