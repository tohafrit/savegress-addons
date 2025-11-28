package resilience

import (
	"testing"
	"time"
)

func TestNewTokenBucketLimiter(t *testing.T) {
	rl := NewTokenBucketLimiter(10.0, 20)

	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if rl.rate != 10.0 {
		t.Errorf("rate = %v, want 10.0", rl.rate)
	}
	if rl.burst != 20 {
		t.Errorf("burst = %d, want 20", rl.burst)
	}
	if rl.tokens != 20.0 {
		t.Errorf("initial tokens = %v, want 20.0", rl.tokens)
	}
}

func TestTokenBucketLimiter_Allow(t *testing.T) {
	rl := NewTokenBucketLimiter(100.0, 5) // 100 tokens/sec, burst of 5

	// Should allow burst
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	if rl.Allow() {
		t.Error("request should be denied after burst exhausted")
	}
}

func TestTokenBucketLimiter_Refill(t *testing.T) {
	rl := NewTokenBucketLimiter(100.0, 5) // 100 tokens/sec

	// Exhaust burst
	for i := 0; i < 5; i++ {
		rl.Allow()
	}

	// Wait for refill (at least 1 token)
	time.Sleep(20 * time.Millisecond)

	// Should allow at least one more request
	if !rl.Allow() {
		t.Error("request should be allowed after refill")
	}
}

func TestTokenBucketLimiter_Wait(t *testing.T) {
	rl := NewTokenBucketLimiter(10.0, 2)

	// Exhaust burst
	rl.Allow()
	rl.Allow()

	wait := rl.Wait()
	if wait <= 0 {
		t.Error("should need to wait for next token")
	}

	// Wait should be approximately 100ms for rate of 10/sec
	if wait > 200*time.Millisecond {
		t.Errorf("wait time %v seems too long", wait)
	}
}

func TestTokenBucketLimiter_Wait_Available(t *testing.T) {
	rl := NewTokenBucketLimiter(10.0, 5)

	// Tokens available
	wait := rl.Wait()
	if wait != 0 {
		t.Errorf("wait = %v, want 0 when tokens available", wait)
	}
}

func TestTokenBucketLimiter_SetRate(t *testing.T) {
	rl := NewTokenBucketLimiter(10.0, 5)

	rl.SetRate(50.0)

	if rl.GetRate() != 50.0 {
		t.Errorf("rate = %v, want 50.0", rl.GetRate())
	}
}

func TestTokenBucketLimiter_GetRate(t *testing.T) {
	rl := NewTokenBucketLimiter(25.0, 10)

	if rl.GetRate() != 25.0 {
		t.Errorf("GetRate() = %v, want 25.0", rl.GetRate())
	}
}

func TestNewSlidingWindowLimiter(t *testing.T) {
	rl := NewSlidingWindowLimiter(10, time.Second)

	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
	if rl.limit != 10 {
		t.Errorf("limit = %d, want 10", rl.limit)
	}
	if rl.window != time.Second {
		t.Errorf("window = %v, want 1s", rl.window)
	}
}

func TestSlidingWindowLimiter_Allow(t *testing.T) {
	rl := NewSlidingWindowLimiter(5, time.Second)

	// Should allow up to limit
	for i := 0; i < 5; i++ {
		if !rl.Allow() {
			t.Errorf("request %d should be allowed", i+1)
		}
	}

	// Next request should be denied
	if rl.Allow() {
		t.Error("request should be denied after limit reached")
	}
}

func TestSlidingWindowLimiter_Window_Expiry(t *testing.T) {
	rl := NewSlidingWindowLimiter(3, 50*time.Millisecond)

	// Use up limit
	for i := 0; i < 3; i++ {
		rl.Allow()
	}

	// Should be denied
	if rl.Allow() {
		t.Error("should be denied immediately after limit")
	}

	// Wait for window to expire
	time.Sleep(60 * time.Millisecond)

	// Should be allowed again
	if !rl.Allow() {
		t.Error("should be allowed after window expires")
	}
}

func TestSlidingWindowLimiter_Wait(t *testing.T) {
	rl := NewSlidingWindowLimiter(2, 100*time.Millisecond)

	// Use up limit
	rl.Allow()
	rl.Allow()

	wait := rl.Wait()
	if wait <= 0 {
		t.Error("should need to wait")
	}
	if wait > 150*time.Millisecond {
		t.Errorf("wait time %v seems too long", wait)
	}
}

func TestSlidingWindowLimiter_Wait_Available(t *testing.T) {
	rl := NewSlidingWindowLimiter(5, time.Second)

	wait := rl.Wait()
	if wait != 0 {
		t.Errorf("wait = %v, want 0 when under limit", wait)
	}
}

func TestNewPerClientRateLimiter(t *testing.T) {
	factory := func() RateLimiter {
		return NewTokenBucketLimiter(10.0, 5)
	}

	rl := NewPerClientRateLimiter(factory)

	if rl == nil {
		t.Fatal("expected non-nil rate limiter")
	}
}

func TestPerClientRateLimiter_Allow(t *testing.T) {
	factory := func() RateLimiter {
		return NewTokenBucketLimiter(100.0, 2)
	}

	rl := NewPerClientRateLimiter(factory)

	// Client A should have its own bucket
	if !rl.Allow("client-a") {
		t.Error("client-a request 1 should be allowed")
	}
	if !rl.Allow("client-a") {
		t.Error("client-a request 2 should be allowed")
	}
	if rl.Allow("client-a") {
		t.Error("client-a request 3 should be denied")
	}

	// Client B should have separate bucket
	if !rl.Allow("client-b") {
		t.Error("client-b request 1 should be allowed")
	}
	if !rl.Allow("client-b") {
		t.Error("client-b request 2 should be allowed")
	}
}

func TestPerClientRateLimiter_Wait(t *testing.T) {
	factory := func() RateLimiter {
		return NewTokenBucketLimiter(10.0, 2)
	}

	rl := NewPerClientRateLimiter(factory)

	// Unknown client should return 0
	wait := rl.Wait("unknown")
	if wait != 0 {
		t.Errorf("wait for unknown client = %v, want 0", wait)
	}

	// Known client after exhausting tokens
	rl.Allow("known")
	rl.Allow("known")

	wait = rl.Wait("known")
	if wait <= 0 {
		t.Error("should need to wait after exhausting tokens")
	}
}

func TestNoOpRateLimiter_Allow(t *testing.T) {
	rl := &NoOpRateLimiter{}

	for i := 0; i < 100; i++ {
		if !rl.Allow() {
			t.Error("NoOpRateLimiter should always allow")
		}
	}
}

func TestNoOpRateLimiter_Wait(t *testing.T) {
	rl := &NoOpRateLimiter{}

	wait := rl.Wait()
	if wait != 0 {
		t.Errorf("NoOpRateLimiter.Wait() = %v, want 0", wait)
	}
}

func TestRateLimiter_Interface(t *testing.T) {
	// Verify all implementations satisfy the interface
	var _ RateLimiter = &TokenBucketLimiter{}
	var _ RateLimiter = &SlidingWindowLimiter{}
	var _ RateLimiter = &NoOpRateLimiter{}
}

func TestTokenBucketLimiter_BurstCap(t *testing.T) {
	rl := NewTokenBucketLimiter(1000.0, 5) // High rate, but low burst

	// Wait to accumulate tokens beyond burst
	time.Sleep(50 * time.Millisecond)

	// Should still only have burst tokens available
	allowed := 0
	for i := 0; i < 10; i++ {
		if rl.Allow() {
			allowed++
		}
	}

	if allowed > 5 {
		t.Errorf("allowed %d requests, burst should cap at 5", allowed)
	}
}
