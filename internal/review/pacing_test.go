package review

import "testing"

func TestNewCardBudget(t *testing.T) {
	tests := []struct {
		name                      string
		dueReviews, lapsedReviews int
		limit                     int
		want                      int
	}{
		{"fresh profile: gentle intake, not a full-session dump", 0, 0, 20, 10},
		{"light day: full intake", 5, 0, 20, 10},
		{"light but lapse-heavy: half intake", 5, 3, 20, 5},
		{"seats left after reviews", 12, 2, 20, 8},
		{"seats left, halved by lapses", 12, 6, 20, 4},
		{"nearly full session of reviews", 18, 0, 20, 2},
		{"full backlog: reviews only", 20, 0, 20, 0},
		{"overloaded backlog: reviews only", 35, 10, 20, 0},
		{"all-lapsed small set: halved", 6, 6, 20, 5},
		{"uncapped queue still paces intake", 0, 0, 0, 10},
		{"uncapped and lapse-heavy still halves", 4, 2, 0, 5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCardBudget(tt.dueReviews, tt.lapsedReviews, tt.limit)
			if got != tt.want {
				t.Errorf("NewCardBudget(%d, %d, %d) = %d, want %d",
					tt.dueReviews, tt.lapsedReviews, tt.limit, got, tt.want)
			}
		})
	}
}
