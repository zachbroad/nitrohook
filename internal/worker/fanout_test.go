package worker

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zachbroad/nitrohook/internal/model"
	"github.com/zachbroad/nitrohook/internal/script"
)

func TestNextRetryTime(t *testing.T) {
	w := &FanoutWorker{
		maxRetries:     5,
		retryBaseDelay: 5 * time.Second,
	}

	// Attempt 1: base delay * 2^0 = 5s (with jitter 75-125% → 3.75s-6.25s)
	t1 := w.nextRetryTime(1)
	if t1 == nil {
		t.Fatal("expected retry time for attempt 1")
	}
	diff1 := time.Until(*t1)
	if diff1 < 3*time.Second || diff1 > 7*time.Second {
		t.Fatalf("attempt 1 delay out of expected range: %v", diff1)
	}

	// Attempt 2: base delay * 2^1 = 10s (with jitter → 7.5s-12.5s)
	t2 := w.nextRetryTime(2)
	if t2 == nil {
		t.Fatal("expected retry time for attempt 2")
	}
	diff2 := time.Until(*t2)
	if diff2 < 6*time.Second || diff2 > 14*time.Second {
		t.Fatalf("attempt 2 delay out of expected range: %v", diff2)
	}

	// Attempt 4: base delay * 2^3 = 40s (with jitter → 30s-50s)
	t4 := w.nextRetryTime(4)
	if t4 == nil {
		t.Fatal("expected retry time for attempt 4")
	}
	diff4 := time.Until(*t4)
	if diff4 < 25*time.Second || diff4 > 55*time.Second {
		t.Fatalf("attempt 4 delay out of expected range: %v", diff4)
	}

	// Cap at 5 minutes: attempt with very large delay should cap
	wLong := &FanoutWorker{
		maxRetries:     20,
		retryBaseDelay: 5 * time.Second,
	}
	tCap := wLong.nextRetryTime(10) // 5s * 2^9 = 2560s → capped to 300s
	if tCap == nil {
		t.Fatal("expected retry time for high attempt")
	}
	diffCap := time.Until(*tCap)
	// Capped at 5min with jitter 75-125% → 225s-375s
	if diffCap > 380*time.Second {
		t.Fatalf("expected delay capped at ~5min, got %v", diffCap)
	}

	// Beyond maxRetries: should return nil
	tNil := w.nextRetryTime(5)
	if tNil != nil {
		t.Fatal("expected nil for attempt >= maxRetries")
	}

	tNil2 := w.nextRetryTime(6)
	if tNil2 != nil {
		t.Fatal("expected nil for attempt > maxRetries")
	}
}

func TestFilterActions(t *testing.T) {
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()

	all := []model.Action{
		{ID: id1},
		{ID: id2},
		{ID: id3},
	}

	// Keep only id1 and id3
	kept := []script.ActionRef{
		{ID: id1},
		{ID: id3},
	}

	filtered := filterActions(all, kept)
	if len(filtered) != 2 {
		t.Fatalf("expected 2 filtered actions, got %d", len(filtered))
	}
	if filtered[0].ID != id1 {
		t.Fatalf("expected first action to be %s, got %s", id1, filtered[0].ID)
	}
	if filtered[1].ID != id3 {
		t.Fatalf("expected second action to be %s, got %s", id3, filtered[1].ID)
	}

	// Empty kept returns empty
	empty := filterActions(all, []script.ActionRef{})
	if len(empty) != 0 {
		t.Fatalf("expected 0 filtered actions, got %d", len(empty))
	}

	// Non-matching IDs returns empty
	nomatch := filterActions(all, []script.ActionRef{{ID: uuid.New()}})
	if len(nomatch) != 0 {
		t.Fatalf("expected 0 filtered actions for non-matching IDs, got %d", len(nomatch))
	}
}
