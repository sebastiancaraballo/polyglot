package model

import "time"

// Profile is a local learner. Multiple profiles can exist on the same machine,
// each with its own progress and statistics.
type Profile struct {
	ID        int64
	Name      string
	Onboarded bool
	CreatedAt time.Time
}
