package model

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"
)

// MaxNameLen is the maximum length, in runes, of a profile name.
const MaxNameLen = 24

// ErrEmptyName and ErrNameTooLong report why a profile name was rejected. Other
// invalid input (control characters, no letters) returns ErrInvalidName.
var (
	ErrEmptyName   = errors.New("model: name is empty")
	ErrNameTooLong = errors.New("model: name is too long")
	ErrInvalidName = errors.New("model: name is invalid")
)

// NormalizeName trims a raw profile name and validates it. The check is simple but
// works for names of any nationality: it accepts any string of Unicode letters (so
// every script qualifies) plus marks, spaces, and common name punctuation, as long
// as it has at least one letter, no control characters, and fits MaxNameLen runes.
func NormalizeName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", ErrEmptyName
	}
	if utf8.RuneCountInString(name) > MaxNameLen {
		return "", ErrNameTooLong
	}

	hasLetter := false
	for _, r := range name {
		switch {
		case unicode.IsControl(r):
			return "", ErrInvalidName
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsMark(r), unicode.IsSpace(r), isNamePunct(r):
			// allowed
		default:
			return "", ErrInvalidName
		}
	}
	if !hasLetter {
		return "", ErrInvalidName
	}
	return name, nil
}

// isNamePunct reports whether r is punctuation that legitimately appears in names
// across languages (e.g. O'Brien, Jean-Luc, Nuñez·).
func isNamePunct(r rune) bool {
	switch r {
	case '-', '\'', '’', '.', '·':
		return true
	default:
		return false
	}
}
