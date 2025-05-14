package ratelimit

import (
	"sync"
	"time"
)

// Limiter provides rate limiting functionality to prevent API abuse.
// It implements a sliding window algorithm for tracking requests
// with configurable limits at minute and hour levels.
type Limiter struct {
	requestsPerMinute int
	requestsPerHour   int
	userRequests      map[int64]*userRequestData
	mu                sync.RWMutex
	cleanupInterval   time.Duration
	stopCleanup       chan struct{}
}

// userRequestData tracks request timing for a specific user.
// It maintains separate slices for minute and hour windows,
// each containing timestamps of requests within those windows.
type userRequestData struct {
	minuteRequests []time.Time
	hourRequests   []time.Time
	lastCleanup    time.Time
}

// New creates a new rate limiter with the specified limits.
//
// Parameters:
//   - requestsPerMinute: Maximum number of requests allowed per minute per user
//   - requestsPerHour: Maximum number of requests allowed per hour per user
//
// If any limit is <= 0, sensible defaults (10 req/min, 100 req/hour) will be used.
// The limiter starts a background goroutine to clean up expired entries.
func New(requestsPerMinute, requestsPerHour int) *Limiter {
	// Ensure sensible defaults if invalid values are provided
	if requestsPerMinute <= 0 {
		requestsPerMinute = 10
	}
	if requestsPerHour <= 0 {
		requestsPerHour = 100
	}

	limiter := &Limiter{
		requestsPerMinute: requestsPerMinute,
		requestsPerHour:   requestsPerHour,
		userRequests:      make(map[int64]*userRequestData),
		cleanupInterval:   10 * time.Minute,
		stopCleanup:       make(chan struct{}),
	}

	// Start background cleanup
	go limiter.startCleanupRoutine()

	return limiter
}

// Allow checks if a user is allowed to make a request based on their usage history.
// It returns true if the request is allowed, or false if either the per-minute
// or per-hour limit has been exceeded.
//
// This method is thread-safe and can be called concurrently from multiple goroutines.
//
// If allowed, the request is recorded and counts against future rate limits.
// The userID parameter should be a unique identifier for the user or client
// (e.g., Telegram user ID, IP address hash, etc.)
func (l *Limiter) Allow(userID int64) bool {
	now := time.Now()
	
	l.mu.Lock()
	defer l.mu.Unlock()

	// Get or create user data
	data, exists := l.userRequests[userID]
	if !exists {
		data = &userRequestData{
			minuteRequests: make([]time.Time, 0, l.requestsPerMinute),
			hourRequests:   make([]time.Time, 0, l.requestsPerHour),
			lastCleanup:    now,
		}
		l.userRequests[userID] = data
	}

	// Clean up old requests for this user
	l.cleanupUserData(userID, now)

	// Check minute limit
	if len(data.minuteRequests) >= l.requestsPerMinute {
		return false
	}

	// Check hour limit
	if len(data.hourRequests) >= l.requestsPerHour {
		return false
	}

	// Record the request
	data.minuteRequests = append(data.minuteRequests, now)
	data.hourRequests = append(data.hourRequests, now)
	
	return true
}

// cleanupUserData removes expired requests from the user's history.
// This is an internal method and should be called while holding the mutex lock.
//
// It filters out request timestamps older than one minute from minuteRequests
// and older than one hour from hourRequests. This implements the sliding window
// approach for rate limiting.
func (l *Limiter) cleanupUserData(userID int64, now time.Time) {
	data := l.userRequests[userID]
	
	// Remove requests older than one minute
	minuteAgo := now.Add(-time.Minute)
	var newMinuteRequests []time.Time
	
	for _, t := range data.minuteRequests {
		if t.After(minuteAgo) {
			newMinuteRequests = append(newMinuteRequests, t)
		}
	}
	data.minuteRequests = newMinuteRequests
	
	// Remove requests older than one hour
	hourAgo := now.Add(-time.Hour)
	var newHourRequests []time.Time
	
	for _, t := range data.hourRequests {
		if t.After(hourAgo) {
			newHourRequests = append(newHourRequests, t)
		}
	}
	data.hourRequests = newHourRequests
	
	data.lastCleanup = now
}

// RemainingMinute returns the number of requests remaining in the current minute
// for the specified user. If the user hasn't made any requests yet, it returns
// the full per-minute limit.
//
// This method is thread-safe and can be used to display rate limit information
// to users or for making decisions about when to retry requests.
func (l *Limiter) RemainingMinute(userID int64) int {
	now := time.Now()
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	data, exists := l.userRequests[userID]
	if !exists {
		return l.requestsPerMinute
	}
	
	// Clean up first to get accurate count
	l.cleanupUserData(userID, now)
	
	return max(0, l.requestsPerMinute - len(data.minuteRequests))
}

// RemainingHour returns the number of requests remaining in the current hour
// for the specified user. If the user hasn't made any requests yet, it returns
// the full per-hour limit.
//
// This method is thread-safe and useful for displaying hourly rate limit information
// to users or for making decisions about retry strategies.
func (l *Limiter) RemainingHour(userID int64) int {
	now := time.Now()
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	data, exists := l.userRequests[userID]
	if !exists {
		return l.requestsPerHour
	}
	
	// Clean up first to get accurate count
	l.cleanupUserData(userID, now)
	
	return max(0, l.requestsPerHour - len(data.hourRequests))
}

// ResetFor resets all rate limits for a specific user, effectively clearing
// their request history. This can be useful for administrative purposes,
// testing, or when a user's circumstances change (e.g., upgrading to a premium tier).
//
// This method is thread-safe.
func (l *Limiter) ResetFor(userID int64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	
	delete(l.userRequests, userID)
}

// GetLimits returns the configured rate limits (requests per minute and per hour).
// This can be useful for displaying limits to users or for logging purposes.
//
// This method is concurrent-safe as it only returns immutable configuration values.
func (l *Limiter) GetLimits() (requestsPerMinute, requestsPerHour int) {
	return l.requestsPerMinute, l.requestsPerHour
}

// startCleanupRoutine periodically cleans up the rate limiter data.
// This runs as a background goroutine to prevent the map from growing
// unbounded as users come and go. It cleans up expired requests and
// removes users who haven't made requests for an extended period.
func (l *Limiter) startCleanupRoutine() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			l.cleanup()
		case <-l.stopCleanup:
			return
		}
	}
}

// cleanup removes expired requests and inactive users.
// This method is called periodically by the background cleanup routine.
// It removes:
// - Individual expired requests from active users
// - Entire user entries for users who haven't made requests in over 24 hours
func (l *Limiter) cleanup() {
	now := time.Now()
	
	l.mu.Lock()
	defer l.mu.Unlock()
	
	for userID := range l.userRequests {
		// Clean up user data
		l.cleanupUserData(userID, now)
		
		// Remove users that haven't made requests in over 24 hours
		if now.Sub(l.userRequests[userID].lastCleanup) > 24*time.Hour {
			delete(l.userRequests, userID)
		}
	}
}

// Close stops the background cleanup routine.
// This should be called when the rate limiter is no longer needed
// to free up resources and prevent goroutine leaks.
func (l *Limiter) Close() {
	close(l.stopCleanup)
}

// max returns the greater of two integers.
// This is a helper function used to ensure remaining request counts
// never go below zero.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}