// Package app wires together the Polyglot application: storage, content, and the
// Bubble Tea UI router.
package app

import (
	"context"
	"errors"
	"fmt"

	tea "charm.land/bubbletea/v2"

	"github.com/sebastiancaraballo/polyglot/internal/content"
	"github.com/sebastiancaraballo/polyglot/internal/i18n"
	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
	"github.com/sebastiancaraballo/polyglot/internal/ui"
)

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

	profile, err := resolveProfile(ctx, store)
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

// resolveProfile returns the persisted active profile, the first existing profile,
// or the zero Profile when this is a first run with no profiles yet.
func resolveProfile(ctx context.Context, store storage.Storage) (model.Profile, error) {
	if id, ok, err := store.GetActiveProfileID(ctx); err != nil {
		return model.Profile{}, err
	} else if ok {
		profile, err := store.GetProfile(ctx, id)
		if err == nil {
			return profile, nil
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return model.Profile{}, err
		}
	}

	profiles, err := store.ListProfiles(ctx)
	if err != nil {
		return model.Profile{}, err
	}
	if len(profiles) == 0 {
		return model.Profile{}, nil
	}
	if err := store.SetActiveProfileID(ctx, profiles[0].ID); err != nil {
		return model.Profile{}, err
	}
	return profiles[0], nil
}
