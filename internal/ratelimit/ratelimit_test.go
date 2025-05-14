package ratelimit

import (
	"testing"
	"time"
)

func TestRateLimiterInitialization(t *testing.T) {
	limiter := New(10, 100)
	
	// Test default values
	rpm, rph := limiter.GetLimits()
	if rpm != 10 {
		t.Errorf("Expected requests per minute to be 10, got %d", rpm)
	}
	if rph != 100 {
		t.Errorf("Expected requests per hour to be 100, got %d", rph)
	}
	
	// Test negative values are corrected to defaults
	limiter = New(-5, -20)
	rpm, rph = limiter.GetLimits()
	if rpm != 10 {
		t.Errorf("Expected negative requests per minute to be corrected to 10, got %d", rpm)
	}
	if rph != 100 {
		t.Errorf("Expected negative requests per hour to be corrected to 100, got %d", rph)
	}
}

func TestRateLimiterAllow(t *testing.T) {
	// Create a limiter with a small limit for testing
	limiter := New(3, 5)
	
	// Test user IDs
	user1 := int64(1001)
	user2 := int64(1002)
	
	// First 3 requests for user1 should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(user1) {
			t.Errorf("Expected request %d to be allowed for user1", i+1)
		}
	}
	
	// 4th request for user1 should be denied (minute limit)
	if limiter.Allow(user1) {
		t.Errorf("Expected 4th request to be denied for user1")
	}
	
	// But user2 should still be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(user2) {
			t.Errorf("Expected request %d to be allowed for user2", i+1)
		}
	}
	
	// Reset for user1
	limiter.ResetFor(user1)
	
	// Now user1 should be allowed again
	if !limiter.Allow(user1) {
		t.Errorf("Expected request to be allowed for user1 after reset")
	}
}

func TestRateLimiterHourlyLimit(t *testing.T) {
	// Create a limiter with high minute limit but low hour limit
	limiter := New(100, 3)
	
	user := int64(2001)
	
	// First 3 requests should be allowed
	for i := 0; i < 3; i++ {
		if !limiter.Allow(user) {
			t.Errorf("Expected request %d to be allowed", i+1)
		}
	}
	
	// 4th request should be denied (hourly limit)
	if limiter.Allow(user) {
		t.Errorf("Expected 4th request to be denied due to hourly limit")
	}
}

func TestRateLimiterRemaining(t *testing.T) {
	limiter := New(5, 50)
	
	user := int64(3001)
	
	// Check initial remaining counts
	if remaining := limiter.RemainingMinute(user); remaining != 5 {
		t.Errorf("Expected 5 requests remaining for new user minute, got %d", remaining)
	}
	
	if remaining := limiter.RemainingHour(user); remaining != 50 {
		t.Errorf("Expected 50 requests remaining for new user hour, got %d", remaining)
	}
	
	// Make some requests
	for i := 0; i < 3; i++ {
		limiter.Allow(user)
	}
	
	// Check updated remaining counts
	if remaining := limiter.RemainingMinute(user); remaining != 2 {
		t.Errorf("Expected 2 requests remaining after 3 requests, got %d", remaining)
	}
	
	if remaining := limiter.RemainingHour(user); remaining != 47 {
		t.Errorf("Expected 47 requests remaining for hour, got %d", remaining)
	}
}

func TestRateLimiterCleanup(t *testing.T) {
	limiter := New(3, 10)
	user := int64(4001)
	
	// Make 3 requests (hitting the minute limit)
	for i := 0; i < 3; i++ {
		limiter.Allow(user)
	}
	
	// Additional request should be denied
	if limiter.Allow(user) {
		t.Errorf("Expected request to be denied after limit reached")
	}
	
	// Manually force cleanup with time in the future
	// This is a direct test of the cleaning logic
	limiter.mu.Lock()
	data := limiter.userRequests[user]
	
	// Create minute timestamps older than one minute
	now := time.Now()
	oldTime := now.Add(-2 * time.Minute)
	data.minuteRequests = []time.Time{oldTime, oldTime, oldTime}
	
	limiter.cleanupUserData(user, now)
	limiter.mu.Unlock()
	
	// Now should be allowed again because old requests were cleaned
	if !limiter.Allow(user) {
		t.Errorf("Expected request to be allowed after cleanup")
	}
}

func TestRateLimiterConcurrentUsers(t *testing.T) {
	// Test with lots of different users to ensure map works correctly
	limiter := New(5, 20)
	
	// Test with 100 different users
	for i := int64(1); i <= 100; i++ {
		// Each user should be able to make 5 requests
		for j := 0; j < 5; j++ {
			if !limiter.Allow(i) {
				t.Errorf("Expected request %d to be allowed for user %d", j+1, i)
			}
		}
		
		// 6th request should be denied
		if limiter.Allow(i) {
			t.Errorf("Expected 6th request to be denied for user %d", i)
		}
	}
	
	// Check we have entries for all users
	limiter.mu.Lock()
	if len(limiter.userRequests) != 100 {
		t.Errorf("Expected 100 users in map, got %d", len(limiter.userRequests))
	}
	limiter.mu.Unlock()
}

func TestRateLimiterTimeWindow(t *testing.T) {
	// This test is time-dependent and might be flaky
	// Best practice would be to inject a clock, but keeping it simple for this example
	limiter := New(5, 10)
	user := int64(5001)
	
	// Create a sequence of requests in the past that should have expired
	limiter.mu.Lock()
	data := &userRequestData{
		minuteRequests: make([]time.Time, 0),
		hourRequests:   make([]time.Time, 0),
		lastCleanup:    time.Now().Add(-30 * time.Minute),
	}
	
	// Add one recent request that shouldn't expire
	now := time.Now()
	data.minuteRequests = append(data.minuteRequests, now.Add(-30*time.Second))
	data.hourRequests = append(data.hourRequests, now.Add(-30*time.Second))
	
	// Add some expired requests (older than 1 minute for minute window)
	for i := 0; i < 10; i++ {
		oldTime := now.Add(-time.Duration(90+i) * time.Second)
		data.minuteRequests = append(data.minuteRequests, oldTime)
		data.hourRequests = append(data.hourRequests, oldTime)
	}
	
	limiter.userRequests[user] = data
	limiter.mu.Unlock()
	
	// Now only the recent request should count, and we should have 4 remaining
	if remaining := limiter.RemainingMinute(user); remaining != 4 {
		t.Errorf("Expected 4 requests remaining after cleanup, got %d", remaining)
	}
}

func TestRateLimiterClose(t *testing.T) {
	// Test that Close doesn't panic
	limiter := New(10, 100)
	limiter.Close()
}