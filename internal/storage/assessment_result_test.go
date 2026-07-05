package storage

import (
	"context"
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
)

func TestAssessmentResultRoundTrip(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}

	// A level never assessed yields a zero-value result, not an error.
	got, err := store.GetAssessmentResult(ctx, profile.ID, model.N5)
	if err != nil {
		t.Fatalf("GetAssessmentResult: %v", err)
	}
	if got.Passed || got.BestCorrect != 0 || got.Total != 0 {
		t.Fatalf("fresh result = %+v, want zero", got)
	}
	if got.Level != model.N5 {
		t.Fatalf("fresh result level = %q, want N5", got.Level)
	}

	taken := time.Now().UTC().Truncate(time.Second)
	want := model.AssessmentResult{Level: model.N5, Passed: true, BestCorrect: 13, Total: 15, TakenAt: taken}
	if err := store.SaveAssessmentResult(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveAssessmentResult: %v", err)
	}

	got, err = store.GetAssessmentResult(ctx, profile.ID, model.N5)
	if err != nil {
		t.Fatalf("GetAssessmentResult: %v", err)
	}
	if got.Passed != want.Passed || got.BestCorrect != want.BestCorrect || got.Total != want.Total {
		t.Fatalf("round-trip = %+v, want %+v", got, want)
	}
	if !got.TakenAt.Equal(taken) {
		t.Errorf("TakenAt = %v, want %v", got.TakenAt, taken)
	}

	// Saving again upserts rather than duplicating.
	want.BestCorrect = 15
	if err := store.SaveAssessmentResult(ctx, profile.ID, want); err != nil {
		t.Fatalf("SaveAssessmentResult (update): %v", err)
	}
	got, err = store.GetAssessmentResult(ctx, profile.ID, model.N5)
	if err != nil {
		t.Fatalf("GetAssessmentResult: %v", err)
	}
	if got.BestCorrect != 15 {
		t.Errorf("BestCorrect after update = %d, want 15", got.BestCorrect)
	}

	var n int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM assessment_result WHERE profile_id = ?`, profile.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 1 {
		t.Errorf("assessment_result rows = %d, want 1 (upsert)", n)
	}
}

func TestAssessmentResultDeletedWithProfile(t *testing.T) {
	store := newTestStore(t)
	ctx := context.Background()

	profile, err := store.CreateProfile(ctx, "tester")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	if err := store.SaveAssessmentResult(ctx, profile.ID,
		model.AssessmentResult{Level: model.N5, Passed: true, BestCorrect: 12, Total: 15}); err != nil {
		t.Fatalf("SaveAssessmentResult: %v", err)
	}
	if err := store.DeleteProfile(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfile: %v", err)
	}

	var n int
	if err := store.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM assessment_result WHERE profile_id = ?`, profile.ID).Scan(&n); err != nil {
		t.Fatalf("count: %v", err)
	}
	if n != 0 {
		t.Errorf("assessment_result rows after profile delete = %d, want 0 (cascade)", n)
	}
}
