package reflex

import (
	"testing"
	"time"
)

func testConfig() Config {
	return Config{NumBuckets: 1024, TickDuration: time.Second, Threshold: 4}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.NumBuckets != 8192 {
		t.Errorf("NumBuckets = %d, want 8192", cfg.NumBuckets)
	}
	if cfg.Threshold != 4 {
		t.Errorf("Threshold = %d, want 4", cfg.Threshold)
	}
}

func TestNew_NonPowerOfTwoPanics(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("expected a panic for a non-power-of-two NumBuckets")
		}
	}()
	New(Config{NumBuckets: 1000, TickDuration: time.Second, Threshold: 4}, nil)
}

func TestNew_NilExemptionsIsSafe(t *testing.T) {
	e := New(testConfig(), nil)
	if got := e.Decide("bash", []byte("bash:{}")); got != ActionPass {
		t.Errorf("first call = %v, want ActionPass", got)
	}
}
