package model

import "time"

// Stats holds per-profile aggregate progress shown on the menu and stats screens.
type Stats struct {
	Streak        int
	BestStreak    int
	LastStudiedAt time.Time // zero value means the profile has never studied
	XP            int       // cumulative experience points earned across all activity
}
