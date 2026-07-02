package review

// New-vs-review pacing: reviews always take priority over introducing new
// material. New (never-reviewed) cards only fill session seats that due
// reviews don't need, and a struggling due set further slows the intake —
// consolidate before adding (Mastery Learning, Bloom).

// maxNewPerSession caps how many never-reviewed cards a single session
// introduces, regardless of how much new content exists.
const maxNewPerSession = 10

// NewCardBudget returns how many new (never-reviewed) cards may join a
// session that already contains dueReviews due review cards, lapsedReviews of
// which have lapsed at least once. New cards only fill seats reviews don't
// need within limit (limit <= 0 means no session cap), and a lapse-heavy due
// set — half or more of the due reviews carry a lapse — halves the intake,
// because struggling recall is the signal to consolidate, not to add.
func NewCardBudget(dueReviews, lapsedReviews, limit int) int {
	budget := maxNewPerSession
	if limit > 0 {
		if space := limit - dueReviews; space < budget {
			budget = space
		}
	}
	if budget < 0 {
		budget = 0
	}
	if dueReviews > 0 && lapsedReviews*2 >= dueReviews {
		budget /= 2
	}
	return budget
}
