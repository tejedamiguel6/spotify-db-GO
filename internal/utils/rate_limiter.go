package utils

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// RateLimiter handles Spotify API rate limiting
type RateLimiter struct {
	requestCount     int
	windowStart      time.Time
	maxRequestsPerMinute int
	backoffMultiplier   float64
	maxBackoffSeconds   int
}

// NewRateLimiter creates a new rate limiter
// Spotify allows ~100 requests per minute, we'll be conservative with 60
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		requestCount:         0,
		windowStart:         time.Now(),
		maxRequestsPerMinute: 60, // Conservative limit
		backoffMultiplier:   2.0,
		maxBackoffSeconds:   60,
	}
}

// Wait blocks if we're approaching rate limits
func (rl *RateLimiter) Wait() {
	now := time.Now()
	
	// Reset counter if we've moved to a new minute
	if now.Sub(rl.windowStart) >= time.Minute {
		rl.requestCount = 0
		rl.windowStart = now
	}
	
	// If we're approaching the limit, wait
	if rl.requestCount >= rl.maxRequestsPerMinute {
		waitTime := time.Until(rl.windowStart.Add(time.Minute))
		if waitTime > 0 {
			fmt.Printf("🐌 Rate limit protection: waiting %v before next request\n", waitTime.Round(time.Second))
			time.Sleep(waitTime)
			// Reset after waiting
			rl.requestCount = 0
			rl.windowStart = time.Now()
		}
	}
	
	rl.requestCount++
	
	// Add small delay between requests regardless
	time.Sleep(100 * time.Millisecond)
}

// HandleRateLimit handles 429 responses with exponential backoff
func (rl *RateLimiter) HandleRateLimit(retryAfterHeader string, attempt int) time.Duration {
	var waitTime time.Duration
	
	// Try to parse Retry-After header
	if retryAfterHeader != "" {
		if seconds, err := strconv.Atoi(retryAfterHeader); err == nil {
			waitTime = time.Duration(seconds) * time.Second
		}
	}
	
	// If no Retry-After header, use exponential backoff
	if waitTime == 0 {
		backoffSeconds := int(math.Min(
			math.Pow(rl.backoffMultiplier, float64(attempt)),
			float64(rl.maxBackoffSeconds),
		))
		waitTime = time.Duration(backoffSeconds) * time.Second
	}
	
	fmt.Printf("⏳ Rate limited! Waiting %v (attempt %d)\n", waitTime.Round(time.Second), attempt+1)
	return waitTime
}

// IsRateLimitError checks if an error is a rate limit error
func IsRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "429") || 
		   strings.Contains(errStr, "rate limit") ||
		   strings.Contains(errStr, "too many requests")
}

// RetryWithBackoff executes a function with retry logic for rate limits
func (rl *RateLimiter) RetryWithBackoff(operation func() error, maxRetries int) error {
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Wait for rate limiting before each attempt
		rl.Wait()
		
		err := operation()
		if err == nil {
			return nil // Success!
		}
		
		lastErr = err
		
		// If it's a rate limit error, wait and retry
		if IsRateLimitError(err) && attempt < maxRetries {
			waitTime := rl.HandleRateLimit("", attempt)
			time.Sleep(waitTime)
			continue
		}
		
		// If it's not a rate limit error, don't retry
		if !IsRateLimitError(err) {
			return err
		}
	}
	
	return fmt.Errorf("max retries exceeded: %v", lastErr)
}