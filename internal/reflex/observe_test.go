package reflex

import "testing"

func TestDecide_PassesUnderThreshold(t *testing.T) {
	e := New(testConfig(), nil)
	key := []byte("bash:{\"command\":\"cargo test\"}")

	for i := range 3 {
		if got := e.Decide("bash", key); got != ActionPass {
			t.Fatalf("call %d = %v, want ActionPass", i+1, got)
		}
	}
}

func TestDecide_TripsAtThreshold(t *testing.T) {
	e := New(testConfig(), nil)
	key := []byte("bash:{\"command\":\"cargo test\"}")

	var last Action
	for range 4 {
		last = e.Decide("bash", key)
	}
	if last != ActionIntercept {
		t.Fatalf("4th call = %v, want ActionIntercept", last)
	}
}

func TestDecide_DistinctKeysDoNotShareCount(t *testing.T) {
	e := New(testConfig(), nil)

	for range 3 {
		e.Decide("bash", []byte("bash:{\"command\":\"a\"}"))
	}
	if got := e.Decide("bash", []byte("bash:{\"command\":\"b\"}")); got != ActionPass {
		t.Errorf("first call on a different key = %v, want ActionPass", got)
	}
}

func TestDecide_ExemptToolAlwaysPasses(t *testing.T) {
	ex := NewExemptions([]string{"get_status"}, nil)
	e := New(testConfig(), ex)
	key := []byte("get_status:{}")

	for i := range 10 {
		if got := e.Decide("get_status", key); got != ActionPass {
			t.Fatalf("call %d = %v, want ActionPass (exempt tool)", i+1, got)
		}
	}
}

func TestDecide_PerToolThresholdOverride(t *testing.T) {
	ex := NewExemptions(nil, map[string]uint32{"get_status": 50})
	e := New(testConfig(), ex) // default threshold 4
	key := []byte("get_status:{}")

	for i := range 10 {
		if got := e.Decide("get_status", key); got != ActionPass {
			t.Fatalf("call %d = %v, want ActionPass (overridden threshold 50)", i+1, got)
		}
	}
}
