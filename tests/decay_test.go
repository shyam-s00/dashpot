package tests

import (
	"testing"
	"time"

	"github.com/shyam-s00/dashpot/internal/reflex"
)

// TestReflex_TripsThenRecoversAfterDecay proves the reflex engine's
// full contract: repeated calls trip the breaker, and it releases on
// its own once enough time passes — no separate reset call needed.
func TestReflex_TripsThenRecoversAfterDecay(t *testing.T) {
	e := reflex.New(reflex.Config{
		NumBuckets:   1024,
		TickDuration: 100 * time.Millisecond,
		Threshold:    4,
	}, nil)
	key := []byte("bash:{\"command\":\"cargo test\"}")

	var last reflex.Action
	for range 4 {
		last = e.Decide("bash", key)
	}
	if last != reflex.ActionIntercept {
		t.Fatalf("4th rapid call = %v, want ActionIntercept", last)
	}

	time.Sleep(150 * time.Millisecond)

	if got := e.Decide("bash", key); got != reflex.ActionPass {
		t.Fatalf("call after decay = %v, want ActionPass", got)
	}
}
