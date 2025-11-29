package resilience

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	if policy.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", policy.MaxRetries)
	}
	if policy.InitialDelay != 100*time.Millisecond {
		t.Errorf("InitialDelay = %v, want 100ms", policy.InitialDelay)
	}
	if policy.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want 30s", policy.MaxDelay)
	}
	if policy.Multiplier != 2.0 {
		t.Errorf("Multiplier = %v, want 2.0", policy.Multiplier)
	}
	if !policy.Jitter {
		t.Error("Jitter should be true by default")
	}
}

func TestRetry_Success_FirstAttempt(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func() error {
		attempts++
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetry_Success_AfterFailures(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	attempts := 0
	err := Retry(context.Background(), policy, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary error")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_AllAttemptsFail(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   2,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	testErr := errors.New("persistent error")
	attempts := 0
	err := Retry(context.Background(), policy, func() error {
		attempts++
		return testErr
	})

	if err != testErr {
		t.Errorf("expected testErr, got %v", err)
	}
	if attempts != 3 { // Initial + 2 retries
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetry_ContextCancellation(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   5,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	attempts := 0

	// Cancel after first attempt
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := Retry(ctx, policy, func() error {
		attempts++
		return errors.New("fail")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestRetryWithResult_Success(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	attempts := 0
	result, err := RetryWithResult(context.Background(), policy, func() (string, error) {
		attempts++
		if attempts < 2 {
			return "", errors.New("temporary")
		}
		return "success", nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != "success" {
		t.Errorf("result = %s, want success", result)
	}
}

func TestRetryWithResult_AllFail(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   1,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	testErr := errors.New("persistent")
	_, err := RetryWithResult(context.Background(), policy, func() (int, error) {
		return 0, testErr
	})

	if err != testErr {
		t.Errorf("expected testErr, got %v", err)
	}
}

func TestRetryWithResult_ContextCancel(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   10,
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := RetryWithResult(ctx, policy, func() (string, error) {
		return "", errors.New("fail")
	})

	if err != context.Canceled {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestCalculateBackoff(t *testing.T) {
	policy := RetryPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
		Jitter:       false,
	}

	// Attempt 0: 100ms
	delay0 := calculateBackoff(policy, 0)
	if delay0 != 100*time.Millisecond {
		t.Errorf("delay at attempt 0 = %v, want 100ms", delay0)
	}

	// Attempt 1: 200ms
	delay1 := calculateBackoff(policy, 1)
	if delay1 != 200*time.Millisecond {
		t.Errorf("delay at attempt 1 = %v, want 200ms", delay1)
	}

	// Attempt 2: 400ms
	delay2 := calculateBackoff(policy, 2)
	if delay2 != 400*time.Millisecond {
		t.Errorf("delay at attempt 2 = %v, want 400ms", delay2)
	}
}

func TestCalculateBackoff_MaxDelay(t *testing.T) {
	policy := RetryPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   10.0,
		Jitter:       false,
	}

	// Should cap at maxDelay
	delay := calculateBackoff(policy, 5)
	if delay > 500*time.Millisecond {
		t.Errorf("delay = %v, should be capped at 500ms", delay)
	}
}

func TestCalculateBackoff_Jitter(t *testing.T) {
	policy := RetryPolicy{
		InitialDelay: 100 * time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
		Jitter:       true,
	}

	// With jitter, delay should vary
	delays := make(map[time.Duration]bool)
	for i := 0; i < 10; i++ {
		delay := calculateBackoff(policy, 1)
		delays[delay] = true
	}

	// Should have some variation
	if len(delays) == 1 {
		t.Error("expected jitter to produce varying delays")
	}
}

func TestRetryableError(t *testing.T) {
	innerErr := errors.New("inner error")
	err := &RetryableError{Err: innerErr}

	if err.Error() != "inner error" {
		t.Errorf("Error() = %s, want 'inner error'", err.Error())
	}

	if err.Unwrap() != innerErr {
		t.Error("Unwrap() should return inner error")
	}
}

func TestIsRetryable(t *testing.T) {
	retryable := &RetryableError{Err: errors.New("retryable")}
	nonRetryable := errors.New("non-retryable")

	if !IsRetryable(retryable) {
		t.Error("RetryableError should be retryable")
	}
	if IsRetryable(nonRetryable) {
		t.Error("regular error should not be retryable")
	}
}

func TestRetryWithCondition(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   3,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	retryableErr := errors.New("retryable")
	nonRetryableErr := errors.New("non-retryable")

	shouldRetry := func(err error) bool {
		return err == retryableErr
	}

	// Test non-retryable error stops immediately
	attempts := 0
	err := RetryWithCondition(context.Background(), policy, shouldRetry, func() error {
		attempts++
		return nonRetryableErr
	})

	if err != nonRetryableErr {
		t.Errorf("expected nonRetryableErr, got %v", err)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 for non-retryable error", attempts)
	}

	// Test retryable error retries
	attempts = 0
	RetryWithCondition(context.Background(), policy, shouldRetry, func() error {
		attempts++
		return retryableErr
	})

	if attempts != 4 { // Initial + 3 retries
		t.Errorf("attempts = %d, want 4 for retryable error", attempts)
	}
}

func TestNewRetryer(t *testing.T) {
	policy := DefaultRetryPolicy()
	shouldRetry := func(err error) bool { return true }

	retryer := NewRetryer(policy, shouldRetry)

	if retryer == nil {
		t.Fatal("expected non-nil retryer")
	}
	if retryer.policy.MaxRetries != policy.MaxRetries {
		t.Error("policy not set correctly")
	}
}

func TestRetryer_Do_WithoutCondition(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   2,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	retryer := NewRetryer(policy, nil)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		if attempts < 2 {
			return errors.New("fail")
		}
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if attempts != 2 {
		t.Errorf("attempts = %d, want 2", attempts)
	}
}

func TestRetryer_Do_WithCondition(t *testing.T) {
	policy := RetryPolicy{
		MaxRetries:   5,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Second,
		Multiplier:   2.0,
	}

	retryableErr := errors.New("retryable")
	shouldRetry := func(err error) bool {
		return err == retryableErr
	}

	retryer := NewRetryer(policy, shouldRetry)

	attempts := 0
	err := retryer.Do(context.Background(), func() error {
		attempts++
		return errors.New("non-retryable")
	})

	if err == nil {
		t.Error("expected error")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryMetrics_Struct(t *testing.T) {
	metrics := RetryMetrics{
		TotalAttempts:     10,
		SuccessfulRetries: 3,
		FailedRetries:     2,
		AverageAttempts:   1.5,
	}

	if metrics.TotalAttempts != 10 {
		t.Error("TotalAttempts not set correctly")
	}
	if metrics.SuccessfulRetries != 3 {
		t.Error("SuccessfulRetries not set correctly")
	}
	if metrics.FailedRetries != 2 {
		t.Error("FailedRetries not set correctly")
	}
	if metrics.AverageAttempts != 1.5 {
		t.Error("AverageAttempts not set correctly")
	}
}

func TestRetryableTask_Struct(t *testing.T) {
	now := time.Now()
	task := RetryableTask{
		ID:          "task-1",
		Fn:          func() error { return nil },
		Attempts:    3,
		LastAttempt: now,
		NextRetry:   now.Add(time.Minute),
		Errors:      []error{errors.New("error1"), errors.New("error2")},
	}

	if task.ID != "task-1" {
		t.Error("ID not set correctly")
	}
	if task.Attempts != 3 {
		t.Error("Attempts not set correctly")
	}
	if len(task.Errors) != 2 {
		t.Error("Errors not set correctly")
	}
}
