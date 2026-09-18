// Package reflex decides whether a tool call should pass through or
// trip the circuit breaker, backed by one fixed-memory epochsketch.Sketch.
package reflex

import (
	"time"

	"github.com/shyam-s00/epochsketch"
)

// Config controls the reflex engine's sizing, decay rate, and default
// trip threshold.
type Config struct {
	NumBuckets   int
	TickDuration time.Duration
	Threshold    uint32
}

// DefaultConfig returns Dashpot's shipped defaults.
func DefaultConfig() Config {
	return Config{
		NumBuckets:   8192,
		TickDuration: 10 * time.Second,
		Threshold:    4,
	}
}

// Engine tracks tool-call frequency and decides pass/intercept per call.
type Engine struct {
	sk        *epochsketch.Sketch
	threshold uint32
	exempt    *Exemptions
}

// New builds an Engine from cfg. exempt may be nil for no exemptions.
// epochsketch.New already panics on a non-power-of-two NumBuckets, so
// New doesn't duplicate that check.
func New(cfg Config, exempt *Exemptions) *Engine {
	return &Engine{
		sk: epochsketch.New(epochsketch.Config{
			NumBuckets:   cfg.NumBuckets,
			TickDuration: cfg.TickDuration,
		}),
		threshold: cfg.Threshold,
		exempt:    exempt,
	}
}
