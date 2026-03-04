# exponential-backoff

[![codecov](https://codecov.io/gh/rohmanhakim/exponential-backoff/graph/badge.svg?token=dmnYkGxYKD)](https://codecov.io/gh/rohmanhakim/exponential-backoff)
[![Go Reference](https://pkg.go.dev/badge/github.com/rohmanhakim/exponential-backoff.svg)](https://pkg.go.dev/github.com/rohmanhakim/exponential-backoff)


A Go package for computing exponential backoff delays with optional jitter and server delay support.

## Installation

```bash
go get github.com/rohmanhakim/exponential-backoff
```

## Usage

### Basic Example

```go
package main

import (
    "fmt"
    "time"

    "github.com/rohmanhakim/exponential-backoff"
)

func main() {
    // Create an immutable config (panics on invalid values)
    config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

    // Calculate delays for successive backoff attempts
    for count := 1; count <= 5; count++ {
        delay := exponentialbackoff.CalculateDelay(count, 0, config)
        fmt.Printf("Attempt %d: %v\n", count, delay)
    }
}
```

Output:
```
Attempt 1: 1s
Attempt 2: 2s
Attempt 3: 4s
Attempt 4: 8s
Attempt 5: 16s
```

### Creating Config

The `Config` struct is immutable and must be created using constructors:

```go
// MustConfig panics on invalid values - use for static configurations
config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

// NewConfig returns an error - use for dynamic configurations
config, err := exponentialbackoff.NewConfig(1*time.Second, 1*time.Minute, 2.0)
if err != nil {
    // handle error
}
```

**Validation rules:**
- `initialDuration` must be >= 0
- `maxDuration` must be > 0
- `multiplier` must be > 0
- `initialDuration` must be <= `maxDuration`

### With Jitter

Adding jitter helps avoid thundering herd problems in distributed systems:

```go
config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

jitter := 100 * time.Millisecond
delay := exponentialbackoff.CalculateDelay(1, jitter, config)
// delay will be 1s + random(0, 100ms)
```

### With Server Delay

Use server-suggested delays (e.g., from `Retry-After` headers or whatever):

```go
config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

// Server suggested a 5-second delay
serverDelay := 5 * time.Second
delay := exponentialbackoff.CalculateDelay(1, 0, config,
    exponentialbackoff.WithServerDelay(serverDelay))
// delay will be max(1s, 5s) = 5s
```

## API

### `NewConfig(initialDuration, maxDuration time.Duration, multiplier float64) (Config, error)`

Creates a new immutable Config with validation. Returns an error if values are invalid.

### `MustConfig(initialDuration, maxDuration time.Duration, multiplier float64) Config`

Creates a new Config, panicking on invalid values. Use for static configurations where errors are impossible.

### `Config` Methods

| Method | Return Type | Description |
|--------|-------------|-------------|
| `InitialDuration()` | `time.Duration` | Returns the starting backoff duration |
| `Multiplier()` | `float64` | Returns the exponential growth factor |
| `MaxDuration()` | `time.Duration` | Returns the maximum backoff cap |


### Calculate Delay

```
CalculateDelay(backoffCount int, jitter time.Duration, config Config, opts ...DelayOption) time.Duration
```

Computes the delay for a given backoff count using exponential backoff.

**Formula:** `initial * (multiplier ^ (count - 1)) + jitter`

- First backoff (count=1): `max(serverDelay, InitialDuration)` or `InitialDuration`
- Second backoff (count=2): `initial * multiplier`
- Capped at `MaxDuration`

### `ComputeJitter(max time.Duration) time.Duration`

Returns a random duration between 0 and max (inclusive). Returns 0 if max ≤ 0.

### `WithServerDelay(d time.Duration) DelayOption`

Functional option to set a server-suggested delay. The initial backoff uses `max(serverDelay, InitialDuration)` as the starting point.

## License

MIT License