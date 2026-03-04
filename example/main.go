package main

import (
	"fmt"
	"time"

	exponentialbackoff "github.com/rohmanhakim/exponential-backoff"
)

func main() {
	config := exponentialbackoff.MustConfig(1*time.Second, 1*time.Minute, 2.0)

	fmt.Println("Basic exponential backoff:")
	fmt.Println("=========================")
	for count := 1; count <= 5; count++ {
		delay := exponentialbackoff.CalculateDelay(count, 0, config)
		fmt.Printf("Attempt %d: %v\n", count, delay)
	}

	fmt.Println("\nWith jitter (100ms max):")
	fmt.Println("=========================")
	jitter := 100 * time.Millisecond
	for count := 1; count <= 3; count++ {
		delay := exponentialbackoff.CalculateDelay(count, jitter, config)
		fmt.Printf("Attempt %d: %v (with random jitter)\n", count, delay)
	}

	fmt.Println("\nWith server delay (5s):")
	fmt.Println("=========================")
	serverDelay := 5 * time.Second
	for count := 1; count <= 3; count++ {
		delay := exponentialbackoff.CalculateDelay(count, 0, config,
			exponentialbackoff.WithServerDelay(serverDelay))
		fmt.Printf("Attempt %d: %v\n", count, delay)
	}

	fmt.Println("\nCompute jitter utility:")
	fmt.Println("=========================")
	for i := 0; i < 5; i++ {
		jitterValue := exponentialbackoff.ComputeJitter(100 * time.Millisecond)
		fmt.Printf("Random jitter: %v\n", jitterValue)
	}
}
