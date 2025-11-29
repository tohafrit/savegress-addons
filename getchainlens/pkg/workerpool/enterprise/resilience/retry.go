package resilience

import (
	"context"
	"errors"
	"math"
	"math/rand"
	"time"
)

// RetryPolicy defines retry behavior
type RetryPolicy struct {
	MaxRetries   int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	Jitter       bool
}

// DefaultRetryPolicy returns a default retry policy
func DefaultRetryPolicy() RetryPolicy {
	return RetryPolicy{
		MaxRetries:   3,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}
}

// RetryableTask wraps a task with retry information
type RetryableTask struct {
	ID          string
	Fn          func() error
	Attempts    int
	LastAttempt time.Time
	NextRetry   time.Time
	Errors      []error
}

// Retry executes a function with retry logic
func Retry(ctx context.Context, policy RetryPolicy, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute function
		err := fn()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Don't retry on last attempt
		if attempt >= policy.MaxRetries {
			break
		}

		// Calculate backoff
		delay := calculateBackoff(policy, attempt)

		// Wait before retry
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	return lastErr
}

// RetryWithResult executes a function that returns a value and error
func RetryWithResult[T any](ctx context.Context, policy RetryPolicy, fn func() (T, error)) (T, error) {
	var result T
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		res, err := fn()
		if err == nil {
			return res, nil // Success
		}

		result = res
		lastErr = err

		if attempt >= policy.MaxRetries {
			break
		}

		delay := calculateBackoff(policy, attempt)

		select {
		case <-ctx.Done():
			return result, ctx.Err()
		case <-time.After(delay):
		}
	}

	return result, lastErr
}

// calculateBackoff calculates the backoff duration for an attempt
func calculateBackoff(policy RetryPolicy, attempt int) time.Duration {
	// Exponential backoff
	delay := float64(policy.InitialDelay) * math.Pow(policy.Multiplier, float64(attempt))

	// Cap at max delay
	if delay > float64(policy.MaxDelay) {
		delay = float64(policy.MaxDelay)
	}

	// Add jitter if enabled
	if policy.Jitter {
		// Add ±25% jitter
		jitter := delay * 0.25 * (rand.Float64()*2 - 1)
		delay += jitter
	}

	return time.Duration(delay)
}

// ShouldRetry determines if an error should be retried
type ShouldRetry func(error) bool

// RetryableError wraps an error indicating it can be retried
type RetryableError struct {
	Err error
}

func (e *RetryableError) Error() string {
	return e.Err.Error()
}

func (e *RetryableError) Unwrap() error {
	return e.Err
}

// IsRetryable checks if an error is retryable
func IsRetryable(err error) bool {
	var retryableErr *RetryableError
	return errors.As(err, &retryableErr)
}

// RetryWithCondition retries only if the error matches the condition
func RetryWithCondition(ctx context.Context, policy RetryPolicy, shouldRetry ShouldRetry, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt <= policy.MaxRetries; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		err := fn()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry this error
		if !shouldRetry(err) {
			return err
		}

		if attempt >= policy.MaxRetries {
			break
		}

		delay := calculateBackoff(policy, attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
	}

	return lastErr
}

// Retryer manages retry operations
type Retryer struct {
	policy      RetryPolicy
	shouldRetry ShouldRetry
}

// NewRetryer creates a new retryer
func NewRetryer(policy RetryPolicy, shouldRetry ShouldRetry) *Retryer {
	return &Retryer{
		policy:      policy,
		shouldRetry: shouldRetry,
	}
}

// Do executes a function with retry
func (r *Retryer) Do(ctx context.Context, fn func() error) error {
	if r.shouldRetry == nil {
		return Retry(ctx, r.policy, fn)
	}
	return RetryWithCondition(ctx, r.policy, r.shouldRetry, fn)
}

// RetryMetrics tracks retry statistics
type RetryMetrics struct {
	TotalAttempts    int64
	SuccessfulRetries int64
	FailedRetries     int64
	AverageAttempts   float64
}
