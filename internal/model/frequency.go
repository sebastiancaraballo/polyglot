package model

// FreqEntry is one ranked word in a target-language word-frequency list. Rank 1
// is the most frequent word; Count is the raw occurrence count in the source
// corpus. Frequency is a property of the target language, so the list is shared
// across every language pair that teaches it.
type FreqEntry struct {
	Rank    int
	Word    string
	Reading string
	Count   int
}
