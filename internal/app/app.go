// Package app wires together the Polyglot application: storage, content, and the
// Bubble Tea UI router.
package app

import (
	"context"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

// defaultProfileName is used when bootstrapping the first profile. Proper profile
// selection arrives with the onboarding flow.
const defaultProfileName = "Estudiante"

// Run starts the Polyglot application. The version string is displayed in the UI.
func Run(version string) error {
	ctx := context.Background()

	course, err := content.LoadEmbedded(content.DefaultPair)
	if err != nil {
		return fmt.Errorf("load course %q: %w", content.DefaultPair, err)
	}

	dbPath, err := storage.DefaultPath()
	if err != nil {
		return err
	}
	store, err := storage.Open(dbPath)
	if err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	defer func() { _ = store.Close() }()

	profile, err := ensureProfile(ctx, store)
	if err != nil {
		return err
	}

	root := newRoot(appContext{
		store:   store,
		course:  course,
		profile: profile,
		theme:   ui.DefaultTheme(),
		msgs:    i18n.Default,
		version: version,
		dbPath:  dbPath,
	})

	if _, err := tea.NewProgram(root).Run(); err != nil {
		return fmt.Errorf("run program: %w", err)
	}
	return nil
}

// ensureProfile returns the first existing profile or creates a default one.
func ensureProfile(ctx context.Context, store storage.Storage) (model.Profile, error) {
	profiles, err := store.ListProfiles(ctx)
	if err != nil {
		return model.Profile{}, err
	}
	if len(profiles) > 0 {
		return profiles[0], nil
	}
	return store.CreateProfile(ctx, defaultProfileName, "")
}
