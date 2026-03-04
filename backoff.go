// Package exponentialbackoff provides utilities for computing exponential backoff delays.
package exponentialbackoff

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// Config holds the immutable configuration for exponential backoff.
// Create instances using NewConfig or MustConfig.
type Config struct {
	initialDuration time.Duration
	multiplier      float64
	maxDuration     time.Duration
}

// NewConfig creates a new Config with validation.
// Returns error if values are invalid:
//   - initialDuration must be >= 0
//   - multiplier must be > 0
//   - maxDuration must be > 0
//   - initialDuration must be <= maxDuration
func NewConfig(initialDuration, maxDuration time.Duration, multiplier float64) (Config, error) {
	if initialDuration < 0 {
		return Config{}, errors.New("initial duration must be non-negative")
	}
	if multiplier <= 0 {
		return Config{}, errors.New("multiplier must be positive")
	}
	if maxDuration <= 0 {
		return Config{}, errors.New("max duration must be positive")
	}
	if initialDuration > maxDuration {
		return Config{}, errors.New("initial duration cannot exceed max duration")
	}

	return Config{
		initialDuration: initialDuration,
		multiplier:      multiplier,
		maxDuration:     maxDuration,
	}, nil
}

// MustConfig creates a new Config, panicking on invalid values.
// Use for static configurations where errors are impossible.
func MustConfig(initialDuration, maxDuration time.Duration, multiplier float64) Config {
	config, err := NewConfig(initialDuration, maxDuration, multiplier)
	if err != nil {
		panic(err)
	}
	return config
}

// InitialDuration returns the starting backoff duration.
func (c Config) InitialDuration() time.Duration {
	return c.initialDuration
}

// Multiplier returns the factor by which the backoff increases each attempt.
func (c Config) Multiplier() float64 {
	return c.multiplier
}

// MaxDuration returns the maximum backoff duration cap.
func (c Config) MaxDuration() time.Duration {
	return c.maxDuration
}

// DelayOption is a functional option for CalculateDelay.
type DelayOption func(*delayOptions)

type delayOptions struct {
	serverDelay time.Duration
}

// WithServerDelay sets a server-suggested delay (e.g., from Retry-After header).
// The initial backoff will use max(serverDelay, InitialDuration) as the starting point.
func WithServerDelay(d time.Duration) DelayOption {
	return func(o *delayOptions) {
		o.serverDelay = d
	}
}

// ComputeJitter returns a pseudo-random duration between 0 and max (inclusive).
// Uses the global rand which is automatically seeded with a random value
// at startup (Go 1.20+), ensuring different jitter values across concurrent calls.
// Returns 0 if max is less than or equal to 0.
func ComputeJitter(max time.Duration) time.Duration {
	if max <= 0 {
		return 0
	}
	return time.Duration(rand.Int63n(int64(max)))
}

// CalculateDelay computes the delay for a given backoff count using exponential backoff
// with optional jitter.
//
// The formula is: initial * (multiplier ^ (count - 1)) + jitter
//   - First backoff (count=1): max(serverDelay, InitialDuration) or InitialDuration
//   - Second backoff (count=2): initial * multiplier
//   - And so on, capped at MaxDuration
//
// Parameters:
//   - backoffCount: The current backoff attempt number (1-indexed)
//   - jitter: Maximum random duration to add to the delay (0 for no jitter)
//   - config: Backoff configuration created with NewConfig or MustConfig
//   - opts: Optional functional options (e.g., WithServerDelay)
//
// Example:
//
//	config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)
//
//	// Basic usage
//	delay := exponentialbackoff.CalculateDelay(1, 100*time.Millisecond, config)
//
//	// With server delay from Retry-After header
//	delay = exponentialbackoff.CalculateDelay(1, 100*time.Millisecond, config,
//	    exponentialbackoff.WithServerDelay(5*time.Second))
func CalculateDelay(backoffCount int, jitter time.Duration, config Config, opts ...DelayOption) time.Duration {
	// Apply options
	options := &delayOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Use server delay as floor for initial backoff if provided
	initialBackoff := config.InitialDuration()
	if options.serverDelay > initialBackoff {
		initialBackoff = options.serverDelay
	}

	// Compute exponential: initial * (multiplier ^ (count - 1))
	exponent := float64(backoffCount - 1)
	delay := float64(initialBackoff) * math.Pow(config.Multiplier(), exponent)
	if delay > float64(config.MaxDuration()) {
		delay = float64(config.MaxDuration())
	}

	// Add jitter only if jitter > 0
	if jitter > 0 {
		jitterValue := ComputeJitter(jitter)
		delay += float64(jitterValue)
	}

	return time.Duration(delay)
}
