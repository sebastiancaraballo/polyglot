// Package app wires together the Polyglot application: storage, content, and the
// Bubble Tea UI router. For now it is a placeholder that will be fleshed out as the
// UI and storage layers land.
package app

import "fmt"

// Run starts the Polyglot application. The version string is displayed in the UI.
func Run(version string) error {
	// TODO: initialize storage, load content, and start the Bubble Tea program.
	fmt.Printf("Polyglot %s — scaffolding in place. UI coming soon.\n", version)
	return nil
}
