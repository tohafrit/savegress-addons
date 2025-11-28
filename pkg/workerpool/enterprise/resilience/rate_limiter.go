package resilience

import (
	"math"
	"sync"
	"time"
)

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow() bool
	Wait() time.Duration
}

// TokenBucketLimiter implements token bucket algorithm
type TokenBucketLimiter struct {
	rate       float64   // Tokens per second
	burst      int       // Max burst size
	tokens     float64   // Current tokens
	lastUpdate time.Time
	mu         sync.Mutex
}

// NewTokenBucketLimiter creates a new token bucket rate limiter
func NewTokenBucketLimiter(rate float64, burst int) *TokenBucketLimiter {
	return &TokenBucketLimiter{
		rate:       rate,
		burst:      burst,
		tokens:     float64(burst),
		lastUpdate: time.Now(),
	}
}

// Allow checks if a request is allowed
func (rl *TokenBucketLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(rl.lastUpdate).Seconds()

	// Refill tokens
	rl.tokens = math.Min(float64(rl.burst), rl.tokens+elapsed*rl.rate)
	rl.lastUpdate = now

	if rl.tokens >= 1.0 {
		rl.tokens -= 1.0
		return true
	}
	return false
}

// Wait returns how long to wait before next token is available
func (rl *TokenBucketLimiter) Wait() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.tokens >= 1.0 {
		return 0
	}

	tokensNeeded := 1.0 - rl.tokens
	return time.Duration(tokensNeeded / rl.rate * float64(time.Second))
}

// SetRate updates the rate limit
func (rl *TokenBucketLimiter) SetRate(rate float64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	rl.rate = rate
}

// GetRate returns the current rate
func (rl *TokenBucketLimiter) GetRate() float64 {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	return rl.rate
}

// SlidingWindowLimiter implements sliding window algorithm
type SlidingWindowLimiter struct {
	limit      int
	window     time.Duration
	requests   []time.Time
	mu         sync.Mutex
}

// NewSlidingWindowLimiter creates a new sliding window rate limiter
func NewSlidingWindowLimiter(limit int, window time.Duration) *SlidingWindowLimiter {
	return &SlidingWindowLimiter{
		limit:    limit,
		window:   window,
		requests: make([]time.Time, 0, limit),
	}
}

// Allow checks if a request is allowed
func (rl *SlidingWindowLimiter) Allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Remove old requests
	validRequests := make([]time.Time, 0, len(rl.requests))
	for _, req := range rl.requests {
		if req.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	rl.requests = validRequests

	// Check limit
	if len(rl.requests) < rl.limit {
		rl.requests = append(rl.requests, now)
		return true
	}
	return false
}

// Wait returns how long to wait before next request is allowed
func (rl *SlidingWindowLimiter) Wait() time.Duration {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if len(rl.requests) < rl.limit {
		return 0
	}

	// Wait until oldest request expires
	oldest := rl.requests[0]
	expiry := oldest.Add(rl.window)
	return time.Until(expiry)
}

// PerClientRateLimiter implements per-client rate limiting
type PerClientRateLimiter struct {
	limiters sync.Map // map[string]RateLimiter
	factory  func() RateLimiter
}

// NewPerClientRateLimiter creates a new per-client rate limiter
func NewPerClientRateLimiter(factory func() RateLimiter) *PerClientRateLimiter {
	return &PerClientRateLimiter{
		factory: factory,
	}
}

// Allow checks if a request from clientID is allowed
func (rl *PerClientRateLimiter) Allow(clientID string) bool {
	limiterI, _ := rl.limiters.LoadOrStore(clientID, rl.factory())
	limiter := limiterI.(RateLimiter)
	return limiter.Allow()
}

// Wait returns how long clientID needs to wait
func (rl *PerClientRateLimiter) Wait(clientID string) time.Duration {
	limiterI, ok := rl.limiters.Load(clientID)
	if !ok {
		return 0
	}
	limiter := limiterI.(RateLimiter)
	return limiter.Wait()
}

// NoOpRateLimiter is a rate limiter that allows everything
type NoOpRateLimiter struct{}

func (n *NoOpRateLimiter) Allow() bool          { return true }
func (n *NoOpRateLimiter) Wait() time.Duration  { return 0 }
