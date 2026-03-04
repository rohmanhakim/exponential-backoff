package exponentialbackoff_test

import (
	"testing"
	"time"

	exponentialbackoff "github.com/rohmanhakim/exponential-backoff"
)

func TestNewConfig(t *testing.T) {
	t.Run("creates valid config", func(t *testing.T) {
		config, err := exponentialbackoff.NewConfig(1*time.Second, 1*time.Minute, 2.0)
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}
		if config.InitialDuration() != 1*time.Second {
			t.Errorf("InitialDuration() = %v, want 1s", config.InitialDuration())
		}
		if config.Multiplier() != 2.0 {
			t.Errorf("Multiplier() = %v, want 2.0", config.Multiplier())
		}
		if config.MaxDuration() != 1*time.Minute {
			t.Errorf("MaxDuration() = %v, want 1m", config.MaxDuration())
		}
	})

	t.Run("rejects negative initial duration", func(t *testing.T) {
		_, err := exponentialbackoff.NewConfig(-1*time.Second, 1*time.Minute, 2.0)
		if err == nil {
			t.Error("NewConfig() should error on negative initial duration")
		}
	})

	t.Run("rejects zero or negative multiplier", func(t *testing.T) {
		_, err := exponentialbackoff.NewConfig(1*time.Second, 1*time.Minute, 0)
		if err == nil {
			t.Error("NewConfig() should error on zero multiplier")
		}
		_, err = exponentialbackoff.NewConfig(1*time.Second, 1*time.Minute, -1)
		if err == nil {
			t.Error("NewConfig() should error on negative multiplier")
		}
	})

	t.Run("rejects zero or negative max duration", func(t *testing.T) {
		_, err := exponentialbackoff.NewConfig(1*time.Second, 0, 2.0)
		if err == nil {
			t.Error("NewConfig() should error on zero max duration")
		}
	})

	t.Run("rejects initial > max duration", func(t *testing.T) {
		_, err := exponentialbackoff.NewConfig(2*time.Minute, 1*time.Minute, 2.0)
		if err == nil {
			t.Error("NewConfig() should error when initial > max")
		}
	})

	t.Run("accepts zero initial duration", func(t *testing.T) {
		config, err := exponentialbackoff.NewConfig(0, 1*time.Minute, 2.0)
		if err != nil {
			t.Errorf("NewConfig() error = %v", err)
		}
		if config.InitialDuration() != 0 {
			t.Errorf("InitialDuration() = %v, want 0", config.InitialDuration())
		}
	})
}

func TestMustConfig(t *testing.T) {
	t.Run("creates valid config without panic", func(t *testing.T) {
		config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)
		if config.InitialDuration() != 1*time.Second {
			t.Errorf("InitialDuration() = %v, want 1s", config.InitialDuration())
		}
	})

	t.Run("panics on invalid config", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("MustConfig() should panic on invalid config")
			}
		}()
		exponentialbackoff.MustConfig(-1*time.Second, 1*time.Minute, 2.0)
	})
}

func TestComputeJitter(t *testing.T) {
	t.Run("returns zero for non-positive input", func(t *testing.T) {
		tests := []time.Duration{-1 * time.Second, 0}
		for _, max := range tests {
			result := exponentialbackoff.ComputeJitter(max)
			if result != 0 {
				t.Errorf("ComputeJitter(%v) = %v, want 0", max, result)
			}
		}
	})

	t.Run("returns value within range", func(t *testing.T) {
		max := 100 * time.Millisecond
		for i := 0; i < 100; i++ {
			result := exponentialbackoff.ComputeJitter(max)
			if result < 0 || result > max {
				t.Errorf("ComputeJitter(%v) = %v, want value in range [0, %v]", max, result, max)
			}
		}
	})
}

func TestCalculateDelay(t *testing.T) {
	config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

	t.Run("first backoff returns initial duration", func(t *testing.T) {
		delay := exponentialbackoff.CalculateDelay(1, 0, config)
		if delay != config.InitialDuration() {
			t.Errorf("CalculateDelay(1, 0, config) = %v, want %v", delay, config.InitialDuration())
		}
	})

	t.Run("exponential growth", func(t *testing.T) {
		tests := []struct {
			count    int
			expected time.Duration
		}{
			{1, 1 * time.Second},
			{2, 2 * time.Second},
			{3, 4 * time.Second},
			{4, 8 * time.Second},
			{5, 16 * time.Second},
			{6, 32 * time.Second},
		}

		for _, tt := range tests {
			delay := exponentialbackoff.CalculateDelay(tt.count, 0, config)
			if delay != tt.expected {
				t.Errorf("CalculateDelay(%d, 0, config) = %v, want %v", tt.count, delay, tt.expected)
			}
		}
	})

	t.Run("caps at max duration", func(t *testing.T) {
		// With 1s initial, 2x multiplier, 60s max:
		// count=7: 64s -> capped to 60s
		delay := exponentialbackoff.CalculateDelay(7, 0, config)
		if delay != config.MaxDuration() {
			t.Errorf("CalculateDelay(7, 0, config) = %v, want %v", delay, config.MaxDuration())
		}
	})

	t.Run("with jitter adds random value", func(t *testing.T) {
		jitter := 100 * time.Millisecond
		baseDelay := exponentialbackoff.CalculateDelay(1, 0, config)

		// Run multiple times to ensure jitter is applied
		hasJitter := false
		for i := 0; i < 100; i++ {
			delay := exponentialbackoff.CalculateDelay(1, jitter, config)
			if delay > baseDelay {
				hasJitter = true
				break
			}
		}

		if !hasJitter {
			t.Error("Expected jitter to be added at least once in 100 attempts")
		}
	})

	t.Run("zero jitter returns exact value", func(t *testing.T) {
		delay := exponentialbackoff.CalculateDelay(1, 0, config)
		if delay != config.InitialDuration() {
			t.Errorf("CalculateDelay with zero jitter = %v, want %v", delay, config.InitialDuration())
		}
	})
}

func TestCalculateDelayWithServerDelay(t *testing.T) {
	config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

	t.Run("uses server delay when greater than initial", func(t *testing.T) {
		serverDelay := 5 * time.Second
		delay := exponentialbackoff.CalculateDelay(1, 0, config, exponentialbackoff.WithServerDelay(serverDelay))
		if delay != serverDelay {
			t.Errorf("CalculateDelay with serverDelay = %v, want %v", delay, serverDelay)
		}
	})

	t.Run("uses initial duration when server delay is smaller", func(t *testing.T) {
		serverDelay := 500 * time.Millisecond
		delay := exponentialbackoff.CalculateDelay(1, 0, config, exponentialbackoff.WithServerDelay(serverDelay))
		if delay != config.InitialDuration() {
			t.Errorf("CalculateDelay with smaller serverDelay = %v, want %v", delay, config.InitialDuration())
		}
	})

	t.Run("server delay affects only first backoff", func(t *testing.T) {
		serverDelay := 4 * time.Second
		// count=1: 4s (server delay)
		// count=2: 4s * 2 = 8s
		delay := exponentialbackoff.CalculateDelay(2, 0, config, exponentialbackoff.WithServerDelay(serverDelay))
		expected := 8 * time.Second
		if delay != expected {
			t.Errorf("CalculateDelay(2) with serverDelay = %v, want %v", delay, expected)
		}
	})

	t.Run("zero server delay is ignored", func(t *testing.T) {
		delay := exponentialbackoff.CalculateDelay(1, 0, config, exponentialbackoff.WithServerDelay(0))
		if delay != config.InitialDuration() {
			t.Errorf("CalculateDelay with zero serverDelay = %v, want %v", delay, config.InitialDuration())
		}
	})
}

func TestWithServerDelay(t *testing.T) {
	t.Run("sets server delay in options", func(t *testing.T) {
		config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)
		serverDelay := 5 * time.Second
		delay := exponentialbackoff.CalculateDelay(1, 0, config,
			exponentialbackoff.WithServerDelay(serverDelay))

		if delay != 5*time.Second {
			t.Errorf("CalculateDelay() WithServerDelay() = %v, want %v", delay, 5*time.Second)
		}
	})
}

func TestConfigDefaults(t *testing.T) {
	// This test documents expected default values
	// Users should set these explicitly in production
	config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

	delay := exponentialbackoff.CalculateDelay(1, 0, config)
	if delay != 1*time.Second {
		t.Errorf("Default initial duration not 1 second")
	}
}

func TestEdgeCases(t *testing.T) {
	t.Run("zero initial duration", func(t *testing.T) {
		config := exponentialbackoff.MustConfig(0, 1*time.Minute, 2.0)
		delay := exponentialbackoff.CalculateDelay(1, 0, config)
		if delay != 0 {
			t.Errorf("CalculateDelay with zero initial = %v, want 0", delay)
		}
	})

	t.Run("multiplier of 1 keeps delay constant", func(t *testing.T) {
		config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 1.0)
		for count := 1; count <= 5; count++ {
			delay := exponentialbackoff.CalculateDelay(count, 0, config)
			if delay != 1*time.Second {
				t.Errorf("CalculateDelay(%d) with multiplier 1 = %v, want 1s", count, delay)
			}
		}
	})

	t.Run("fractional multiplier", func(t *testing.T) {
		config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 1.5)
		// count=2: 1s * 1.5 = 1.5s
		delay := exponentialbackoff.CalculateDelay(2, 0, config)
		expected := 1500 * time.Millisecond
		if delay != expected {
			t.Errorf("CalculateDelay(2) with 1.5 multiplier = %v, want %v", delay, expected)
		}
	})
}
