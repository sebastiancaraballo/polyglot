package storage

import (
	"context"
	"errors"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// ErrNotFound is returned when a requested record does not exist.
var ErrNotFound = errors.New("storage: not found")

// Storage persists learner profiles and their progress. Implementations must be
// safe for sequential use by a single running application instance.
type Storage interface {
	// CreateProfile creates a new profile (with an empty stats row) and returns it.
	CreateProfile(ctx context.Context, name string) (model.Profile, error)
	// DeleteProfile removes a profile and its stats and card states.
	DeleteProfile(ctx context.Context, id int64) error
	// ListProfiles returns all profiles ordered by creation time.
	ListProfiles(ctx context.Context) ([]model.Profile, error)
	// GetProfile returns the profile with the given id, or ErrNotFound.
	GetProfile(ctx context.Context, id int64) (model.Profile, error)
	// SetOnboarded marks a profile as having completed onboarding.
	SetOnboarded(ctx context.Context, profileID int64) error
	// SetShowRomaji sets whether a profile displays romaji alongside Japanese.
	SetShowRomaji(ctx context.Context, profileID int64, enabled bool) error

	// GetActiveProfileID returns the persisted active profile id; ok is false when
	// none has been set.
	GetActiveProfileID(ctx context.Context) (id int64, ok bool, err error)
	// SetActiveProfileID persists which profile is active.
	SetActiveProfileID(ctx context.Context, id int64) error

	// GetCardState returns the scheduling state for a card, or ErrNotFound if the
	// card has never been reviewed by the profile.
	GetCardState(ctx context.Context, profileID int64, cardID string) (model.CardState, error)
	// SaveCardState inserts or updates the scheduling state for a card.
	SaveCardState(ctx context.Context, profileID int64, state model.CardState) error

	// GetKanaProgress returns the profile's kana automaticity progress, keyed by
	// kana character. Kana the profile has never practiced are absent.
	GetKanaProgress(ctx context.Context, profileID int64) (map[string]model.KanaProgress, error)
	// SaveKanaProgress inserts or updates the automaticity progress for one kana.
	SaveKanaProgress(ctx context.Context, profileID int64, p model.KanaProgress) error

	// GetStats returns the aggregate stats for a profile.
	GetStats(ctx context.Context, profileID int64) (model.Stats, error)
	// SaveStats replaces the aggregate stats for a profile.
	SaveStats(ctx context.Context, profileID int64, stats model.Stats) error
	// AddXP atomically increments a profile's cumulative experience points.
	AddXP(ctx context.Context, profileID int64, amount int) error

	// CountLearnedCards returns how many cards the profile has reviewed at least
	// once successfully (reps > 0).
	CountLearnedCards(ctx context.Context, profileID int64) (int, error)

	// Close releases the underlying resources.
	Close() error
}
