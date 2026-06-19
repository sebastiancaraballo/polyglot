package study

import (
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

// UpdateStreak advances the study streak for a session occurring at now. It is a
// no-op if the profile already studied today; it increments the streak if the
// last study day was yesterday, and otherwise resets it to 1. BestStreak and
// LastStudiedAt are updated accordingly.
func UpdateStreak(stats model.Stats, now time.Time) model.Stats {
	today := truncateDay(now)
	switch {
	case stats.LastStudiedAt.IsZero():
		stats.Streak = 1
	case truncateDay(stats.LastStudiedAt).Equal(today):
		return stats // already counted today
	case truncateDay(stats.LastStudiedAt).Equal(today.AddDate(0, 0, -1)):
		stats.Streak++
	default:
		stats.Streak = 1
	}

	if stats.Streak > stats.BestStreak {
		stats.BestStreak = stats.Streak
	}
	stats.LastStudiedAt = now
	return stats
}

func truncateDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}
