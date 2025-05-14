package main

import (
	"flag"
	"fmt"
	"sync"
	"time"

	"github.com/ceesaxp/cocktail-bot/internal/ratelimit"
)

// requestResult stores the result of a rate limit check
type requestResult struct {
	userID        int64
	requestNumber int
	allowed       bool
	timestamp     time.Time
}

func main() {
	fmt.Println("Cocktail Bot Rate Limiter Demo")
	fmt.Println("==============================")

	// Parse command line arguments
	requestsPerMin := flag.Int("rpm", 10, "Requests per minute")
	requestsPerHour := flag.Int("rph", 100, "Requests per hour")
	concurrentUsers := flag.Int("users", 5, "Number of concurrent users to simulate")
	requestsPerUser := flag.Int("requests", 15, "Number of requests per user")
	delay := flag.Int("delay", 100, "Delay between requests in milliseconds")
	flag.Parse()

	// Create rate limiter
	limiter := ratelimit.New(*requestsPerMin, *requestsPerHour)
	defer limiter.Close()

	fmt.Printf("Rate limits: %d req/min, %d req/hour\n\n", *requestsPerMin, *requestsPerHour)

	// Display test scenario
	fmt.Printf("Simulating %d users making %d requests each with %dms delay between requests\n\n",
		*concurrentUsers, *requestsPerUser, *delay)

	// Create wait group to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(*concurrentUsers)

	// Start time
	startTime := time.Now()

	// Create channels for collecting results
	resultChan := make(chan requestResult, *concurrentUsers**requestsPerUser)

	// Simulate concurrent users
	for i := 0; i < *concurrentUsers; i++ {
		userID := int64(1000 + i)
		go func(id int64) {
			defer wg.Done()
			simulateUser(id, *requestsPerUser, *delay, limiter, resultChan)
		}(userID)
	}

	// Wait for all users to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect and process results
	results := make([]requestResult, 0, *concurrentUsers**requestsPerUser)
	for r := range resultChan {
		results = append(results, r)
	}

	// Display summary
	elapsed := time.Since(startTime)
	fmt.Printf("\nTest completed in %.2f seconds\n", elapsed.Seconds())

	// Count allowed and denied requests
	allowed := 0
	denied := 0
	for _, r := range results {
		if r.allowed {
			allowed++
		} else {
			denied++
		}
	}

	fmt.Printf("Total requests: %d\n", len(results))
	fmt.Printf("Allowed: %d (%.1f%%)\n", allowed, float64(allowed)/float64(len(results))*100)
	fmt.Printf("Denied: %d (%.1f%%)\n", denied, float64(denied)/float64(len(results))*100)

	// Display per-user statistics
	fmt.Println("\nPer-user statistics:")
	userCounts := make(map[int64]struct{ allowed, denied int })
	for _, r := range results {
		counts := userCounts[r.userID]
		if r.allowed {
			counts.allowed++
		} else {
			counts.denied++
		}
		userCounts[r.userID] = counts
	}

	for i := 0; i < *concurrentUsers; i++ {
		userID := int64(1000 + i)
		counts := userCounts[userID]
		fmt.Printf("User %d: %d allowed, %d denied\n", userID, counts.allowed, counts.denied)
	}

	fmt.Println("\nRate Limiter Analysis:")
	for i := 0; i < *concurrentUsers; i++ {
		userID := int64(1000 + i)
		fmt.Printf("User %d: %d req/min remaining, %d req/hour remaining\n",
			userID, limiter.RemainingMinute(userID), limiter.RemainingHour(userID))
	}

	fmt.Println("\nDemo completed successfully!")
}

// simulateUser simulates a user making requests to the rate limiter
func simulateUser(userID int64, numRequests, delayMs int, limiter *ratelimit.Limiter, resultChan chan<- requestResult) {
	for i := 0; i < numRequests; i++ {
		// Make request to rate limiter
		allowed := limiter.Allow(userID)

		// Send result
		resultChan <- requestResult{
			userID:        userID,
			requestNumber: i + 1,
			allowed:       allowed,
			timestamp:     time.Now(),
		}

		// Display result
		status := "✓"
		if !allowed {
			status = "✗"
		}
		fmt.Printf("User %d Request %2d: %s (Remaining: %d/min, %d/hr)\n",
			userID, i+1, status, limiter.RemainingMinute(userID), limiter.RemainingHour(userID))

		// Delay between requests
		time.Sleep(time.Duration(delayMs) * time.Millisecond)
	}
}
