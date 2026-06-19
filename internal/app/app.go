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
	"github.com/sebastiancaraballo/polyglot/internal/screens/menu"
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
	summary, err := buildSummary(ctx, store, course, profile.ID)
	if err != nil {
		return err
	}

	menuModel := menu.New(ui.DefaultTheme(), i18n.Default, summary, version)
	program := tea.NewProgram(newRoot(menuModel))
	if _, err := program.Run(); err != nil {
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
	return store.CreateProfile(ctx, defaultProfileName)
}

// buildSummary computes the menu's progress badge from stored progress and the
// loaded course.
func buildSummary(ctx context.Context, store storage.Storage, course *content.Course, profileID int64) (menu.Summary, error) {
	stats, err := store.GetStats(ctx, profileID)
	if err != nil {
		return menu.Summary{}, err
	}
	learned, err := store.CountLearnedCards(ctx, profileID)
	if err != nil {
		return menu.Summary{}, err
	}

	total := 0
	for _, lesson := range course.Lessons {
		total += len(lesson.Cards)
	}
	percent := 0
	if total > 0 {
		percent = learned * 100 / total
	}

	// v1 ships N5 content, so the target level is fixed for now.
	return menu.Summary{
		Level:     string(model.N5),
		NextLevel: string(model.N4),
		Percent:   percent,
		Streak:    stats.Streak,
		Learned:   learned,
		Total:     total,
	}, nil
}
