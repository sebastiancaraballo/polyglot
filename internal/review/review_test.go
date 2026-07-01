package review_test

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/sebastiancaraballo/polyglot/internal/model"
	"github.com/sebastiancaraballo/polyglot/internal/review"
	"github.com/sebastiancaraballo/polyglot/internal/storage"
)

func newStore(t *testing.T) (*storage.SQLiteStore, int64) {
	t.Helper()
	store, err := storage.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	profile, err := store.CreateProfile(context.Background(), "learner")
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	return store, profile.ID
}

func item(id string, strand review.Strand) review.Item {
	return review.Item{CardID: id, Strand: strand, Prompt: id, Answer: id}
}

func save(t *testing.T, store *storage.SQLiteStore, profileID int64, id string, due time.Time) {
	t.Helper()
	st := model.CardState{CardID: id, Ease: model.DefaultEase, Reps: 1, Interval: 1, DueAt: due, LastReviewedAt: due}
	if err := store.SaveCardState(context.Background(), profileID, st); err != nil {
		t.Fatalf("SaveCardState %s: %v", id, err)
	}
}

func ids(q []review.Scheduled) []string {
	out := make([]string, len(q))
	for i, s := range q {
		out[i] = s.Item.CardID
	}
	return out
}

func equal(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Never-seen items have a zero due date, so they are immediately due.
func TestBuildQueueNewItemsAreDue(t *testing.T) {
	store, pid := newStore(t)
	items := []review.Item{item("v:1", review.Vocab), item("v:2", review.Vocab)}

	q, err := review.BuildQueue(context.Background(), store, pid, items, time.Now(), 0)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	if len(q.Items) != 2 {
		t.Fatalf("want 2 due items, got %d", len(q.Items))
	}
}

// Items scheduled in the future are excluded from the queue.
func TestBuildQueueFiltersNotDue(t *testing.T) {
	store, pid := newStore(t)
	now := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	save(t, store, pid, "v:1", now.Add(-time.Hour)) // due
	save(t, store, pid, "v:2", now.Add(48*time.Hour))

	items := []review.Item{item("v:1", review.Vocab), item("v:2", review.Vocab)}
	q, err := review.BuildQueue(context.Background(), store, pid, items, now, 0)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	if got := ids(q.Items); !equal(got, []string{"v:1"}) {
		t.Fatalf("want [v:1], got %v", got)
	}
}

// Within and across strands, the queue is most-overdue first per strand and
// interleaved round-robin across strands.
func TestBuildQueueOverdueFirstInterleaved(t *testing.T) {
	store, pid := newStore(t)
	now := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	save(t, store, pid, "v:1", now.Add(-3*day))
	save(t, store, pid, "v:2", now.Add(-1*day))
	save(t, store, pid, "v:3", now.Add(-2*day))
	save(t, store, pid, "k:1", now.Add(-5*day))
	save(t, store, pid, "k:2", now.Add(-4*day))

	items := []review.Item{
		item("v:1", review.Vocab), item("v:2", review.Vocab), item("v:3", review.Vocab),
		item("k:1", review.Kana), item("k:2", review.Kana),
	}
	q, err := review.BuildQueue(context.Background(), store, pid, items, now, 0)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	want := []string{"v:1", "k:1", "v:3", "k:2", "v:2"}
	if got := ids(q.Items); !equal(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestBuildQueueRespectsLimit(t *testing.T) {
	store, pid := newStore(t)
	items := []review.Item{
		item("v:1", review.Vocab), item("v:2", review.Vocab), item("v:3", review.Vocab),
	}
	q, err := review.BuildQueue(context.Background(), store, pid, items, time.Now(), 2)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	if len(q.Items) != 2 {
		t.Fatalf("want 2 items with limit, got %d", len(q.Items))
	}
}

// saveLapsed stores a due review card carrying a lapse.
func saveLapsed(t *testing.T, store *storage.SQLiteStore, profileID int64, id string, due time.Time) {
	t.Helper()
	st := model.CardState{CardID: id, Ease: model.DefaultEase, Reps: 1, Lapses: 1, Interval: 1, DueAt: due, LastReviewedAt: due}
	if err := store.SaveCardState(context.Background(), profileID, st); err != nil {
		t.Fatalf("SaveCardState %s: %v", id, err)
	}
}

// A fresh profile's all-new queue is paced: at most maxNewPerSession enter,
// and the rest are reported as held back rather than silently dropped.
func TestBuildQueuePacesNewCards(t *testing.T) {
	store, pid := newStore(t)
	var items []review.Item
	for i := 0; i < 15; i++ {
		items = append(items, item(fmt.Sprintf("v:%d", i), review.Vocab))
	}

	q, err := review.BuildQueue(context.Background(), store, pid, items, time.Now(), 20)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	if len(q.Items) != 10 {
		t.Fatalf("want 10 new cards admitted, got %d", len(q.Items))
	}
	if q.HeldBackNew != 5 {
		t.Fatalf("HeldBackNew = %d, want 5", q.HeldBackNew)
	}
	if q.DueReviews != 0 {
		t.Fatalf("DueReviews = %d, want 0", q.DueReviews)
	}
}

// Due reviews take every seat they need; new cards only fill what's left.
func TestBuildQueueReviewsTakePriorityOverNew(t *testing.T) {
	store, pid := newStore(t)
	now := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	var items []review.Item
	for i := 0; i < 18; i++ {
		id := fmt.Sprintf("r:%d", i)
		save(t, store, pid, id, now.Add(-time.Hour))
		items = append(items, item(id, review.Vocab))
	}
	for i := 0; i < 6; i++ {
		items = append(items, item(fmt.Sprintf("n:%d", i), review.Vocab))
	}

	q, err := review.BuildQueue(context.Background(), store, pid, items, now, 20)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	if q.DueReviews != 18 {
		t.Fatalf("DueReviews = %d, want 18", q.DueReviews)
	}
	// 18 reviews leave 2 seats: 2 new admitted, 4 held back.
	if len(q.Items) != 20 {
		t.Fatalf("queue size = %d, want 20", len(q.Items))
	}
	if q.HeldBackNew != 4 {
		t.Fatalf("HeldBackNew = %d, want 4", q.HeldBackNew)
	}
}

// A lapse-heavy due set halves the new-card intake end-to-end.
func TestBuildQueueLapseHeavySetHalvesIntake(t *testing.T) {
	store, pid := newStore(t)
	now := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	var items []review.Item
	for i := 0; i < 4; i++ {
		id := fmt.Sprintf("r:%d", i)
		saveLapsed(t, store, pid, id, now.Add(-time.Hour))
		items = append(items, item(id, review.Vocab))
	}
	for i := 0; i < 12; i++ {
		items = append(items, item(fmt.Sprintf("n:%d", i), review.Vocab))
	}

	q, err := review.BuildQueue(context.Background(), store, pid, items, now, 20)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	// 4 due reviews, all lapsed → budget 10 halved to 5.
	if len(q.Items) != 9 {
		t.Fatalf("queue size = %d, want 9 (4 reviews + 5 new)", len(q.Items))
	}
	if q.HeldBackNew != 7 {
		t.Fatalf("HeldBackNew = %d, want 7", q.HeldBackNew)
	}
}

// Strand interleaving still operates on the admitted set.
func TestBuildQueueInterleavesAdmittedNewCards(t *testing.T) {
	store, pid := newStore(t)
	items := []review.Item{
		item("v:1", review.Vocab), item("v:2", review.Vocab),
		item("k:1", review.Kana), item("k:2", review.Kana),
	}
	q, err := review.BuildQueue(context.Background(), store, pid, items, time.Now(), 20)
	if err != nil {
		t.Fatalf("BuildQueue: %v", err)
	}
	want := []string{"v:1", "k:1", "v:2", "k:2"}
	if got := ids(q.Items); !equal(got, want) {
		t.Fatalf("want %v, got %v", want, got)
	}
}

func TestKanaCardID(t *testing.T) {
	if got := review.KanaCardID(model.KanaItem{Char: "あ"}); got != "kana:あ" {
		t.Fatalf("KanaCardID = %q, want kana:あ", got)
	}
}

func TestKanaItemsMapFields(t *testing.T) {
	items := review.KanaItems([]model.KanaItem{{Char: "あ", Romaji: "a"}})
	if len(items) != 1 {
		t.Fatalf("want 1 item, got %d", len(items))
	}
	it := items[0]
	if it.Prompt != "あ" || it.Answer != "a" || it.Strand != review.Kana {
		t.Fatalf("unexpected kana item: %+v", it)
	}
}
