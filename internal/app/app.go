// Package app wires together the Polyglot application: storage, content, and the
// Bubble Tea UI router. For now it is a placeholder that will be fleshed out as the
// UI layer lands.
package app

import (
	"fmt"

	"github.com/sebastiancaraballo/polyglot/internal/content"
)

// Run starts the Polyglot application. The version string is displayed in the UI.
func Run(version string) error {
	course, err := content.LoadEmbedded(content.DefaultPair)
	if err != nil {
		return fmt.Errorf("load course %q: %w", content.DefaultPair, err)
	}

	// TODO: initialize storage and start the Bubble Tea program.
	fmt.Printf("Polyglot %s — loaded %d lessons and %d kana for %s. UI coming soon.\n",
		version, len(course.Lessons), len(course.Kana), course.Pair)
	return nil
}
