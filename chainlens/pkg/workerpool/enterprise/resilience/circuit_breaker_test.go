package resilience

import (
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewCircuitBreaker(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 2,
	})

	if cb == nil {
		t.Fatal("expected non-nil circuit breaker")
	}
	if cb.name != "test" {
		t.Errorf("name = %s, want test", cb.name)
	}
	if cb.failureThreshold != 5 {
		t.Errorf("failureThreshold = %d, want 5", cb.failureThreshold)
	}
	if cb.successThreshold != 3 {
		t.Errorf("successThreshold = %d, want 3", cb.successThreshold)
	}
	if cb.GetState() != StateClosed {
		t.Errorf("initial state = %s, want closed", cb.GetState())
	}
}

func TestCircuitBreaker_Call_Success(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 1,
	})

	called := false
	err := cb.Call(func() error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if !called {
		t.Error("function was not called")
	}
	if cb.GetState() != StateClosed {
		t.Errorf("state = %s, want closed", cb.GetState())
	}
}

func TestCircuitBreaker_Call_FailureThreshold(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 3,
		SuccessThreshold: 2,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 1,
	})

	testErr := errors.New("test error")

	// Fail multiple times to open the circuit
	for i := 0; i < 3; i++ {
		err := cb.Call(func() error {
			return testErr
		})
		if err != testErr {
			t.Errorf("expected test error, got %v", err)
		}
	}

	if cb.GetState() != StateOpen {
		t.Errorf("state = %s, want open", cb.GetState())
	}

	// Next call should fail immediately with ErrCircuitOpen
	err := cb.Call(func() error {
		return nil
	})
	if err != ErrCircuitOpen {
		t.Errorf("expected ErrCircuitOpen, got %v", err)
	}
}

func TestCircuitBreaker_HalfOpen_Transition(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	// Fail to open
	cb.Call(func() error {
		return errors.New("fail")
	})

	if cb.GetState() != StateOpen {
		t.Errorf("state = %s, want open", cb.GetState())
	}

	// Wait for timeout
	time.Sleep(100 * time.Millisecond)

	// Next call should transition to half-open and succeed
	err := cb.Call(func() error {
		return nil
	})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// State should be half-open or closed after success
	state := cb.GetState()
	if state != StateHalfOpen && state != StateClosed {
		t.Errorf("state = %s, want half-open or closed", state)
	}
}

func TestCircuitBreaker_HalfOpen_Success_Closes(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 5,
	})

	// Open the circuit
	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(60 * time.Millisecond)

	// Succeed enough times to close
	for i := 0; i < 2; i++ {
		cb.Call(func() error { return nil })
	}

	if cb.GetState() != StateClosed {
		t.Errorf("state = %s, want closed", cb.GetState())
	}
}

func TestCircuitBreaker_HalfOpen_Failure_Opens(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 5,
	})

	// Open the circuit
	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(60 * time.Millisecond)

	// Fail in half-open state
	cb.Call(func() error { return errors.New("fail again") })

	if cb.GetState() != StateOpen {
		t.Errorf("state = %s, want open", cb.GetState())
	}
}

func TestCircuitBreaker_TooManyRequests(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 2,
		Timeout:          50 * time.Millisecond,
		HalfOpenMaxCalls: 1,
	})

	// Open the circuit
	cb.Call(func() error { return errors.New("fail") })
	time.Sleep(60 * time.Millisecond)

	// Manually set to half-open
	cb.setState(StateHalfOpen)
	cb.halfOpenCalls.Store(1) // Already at max

	err := cb.Call(func() error { return nil })
	if err != ErrTooManyRequests {
		t.Errorf("expected ErrTooManyRequests, got %v", err)
	}
}

func TestCircuitBreaker_GetMetrics(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test-metrics",
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 1,
	})

	// Record some activity
	cb.Call(func() error { return nil })
	cb.Call(func() error { return nil })
	cb.Call(func() error { return errors.New("fail") })

	metrics := cb.GetMetrics()

	if metrics.Name != "test-metrics" {
		t.Errorf("Name = %s, want test-metrics", metrics.Name)
	}
	if metrics.State != string(StateClosed) {
		t.Errorf("State = %s, want closed", metrics.State)
	}
	if metrics.Failures != 1 {
		t.Errorf("Failures = %d, want 1", metrics.Failures)
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 1,
	})

	// Open the circuit
	cb.Call(func() error { return errors.New("fail") })
	if cb.GetState() != StateOpen {
		t.Errorf("state = %s, want open", cb.GetState())
	}

	cb.Reset()

	if cb.GetState() != StateClosed {
		t.Errorf("state after reset = %s, want closed", cb.GetState())
	}
	if cb.failures.Load() != 0 {
		t.Error("failures should be 0 after reset")
	}
	if cb.consecutiveFails.Load() != 0 {
		t.Error("consecutiveFails should be 0 after reset")
	}
}

func TestCircuitBreaker_StateChangeCallback(t *testing.T) {
	var fromState, toState CircuitState
	var callbackCalled bool
	callbackDone := make(chan struct{}, 1)

	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 1,
		SuccessThreshold: 1,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 1,
		OnStateChange: func(from, to CircuitState) {
			fromState = from
			toState = to
			callbackCalled = true
			select {
			case callbackDone <- struct{}{}:
			default:
			}
		},
	})

	// Trigger state change
	cb.Call(func() error { return errors.New("fail") })

	// Wait for callback
	select {
	case <-callbackDone:
	case <-time.After(100 * time.Millisecond):
	}

	if !callbackCalled {
		t.Error("callback was not called")
	}
	if fromState != StateClosed {
		t.Errorf("fromState = %s, want closed", fromState)
	}
	if toState != StateOpen {
		t.Errorf("toState = %s, want open", toState)
	}
}

func TestCircuitBreaker_Concurrent(t *testing.T) {
	cb := NewCircuitBreaker(CircuitBreakerConfig{
		Name:             "test",
		FailureThreshold: 100,
		SuccessThreshold: 10,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 10,
	})

	var wg sync.WaitGroup
	successCount := 0
	var mu sync.Mutex

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := cb.Call(func() error { return nil })
			if err == nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	if successCount != 100 {
		t.Errorf("successCount = %d, want 100", successCount)
	}
}

func TestCircuitBreakerManager_Get(t *testing.T) {
	cbm := NewCircuitBreakerManager(CircuitBreakerConfig{
		Name:             "default",
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 2,
	})

	cb1 := cbm.Get("task-type-1")
	cb2 := cbm.Get("task-type-2")
	cb1Again := cbm.Get("task-type-1")

	if cb1 == nil || cb2 == nil {
		t.Fatal("expected non-nil circuit breakers")
	}

	if cb1 != cb1Again {
		t.Error("expected same circuit breaker for same task type")
	}

	if cb1.name != "task-type-1" {
		t.Errorf("cb1.name = %s, want task-type-1", cb1.name)
	}
	if cb2.name != "task-type-2" {
		t.Errorf("cb2.name = %s, want task-type-2", cb2.name)
	}
}

func TestCircuitBreakerManager_GetAll(t *testing.T) {
	cbm := NewCircuitBreakerManager(CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 3,
		Timeout:          time.Second,
		HalfOpenMaxCalls: 2,
	})

	cbm.Get("type-a")
	cbm.Get("type-b")
	cbm.Get("type-c")

	all := cbm.GetAll()

	if len(all) != 3 {
		t.Errorf("expected 3 circuit breakers, got %d", len(all))
	}

	for _, name := range []string{"type-a", "type-b", "type-c"} {
		if _, ok := all[name]; !ok {
			t.Errorf("expected circuit breaker for %s", name)
		}
	}
}

func TestCircuitState_Constants(t *testing.T) {
	if StateClosed != "closed" {
		t.Errorf("StateClosed = %s, want closed", StateClosed)
	}
	if StateOpen != "open" {
		t.Errorf("StateOpen = %s, want open", StateOpen)
	}
	if StateHalfOpen != "half-open" {
		t.Errorf("StateHalfOpen = %s, want half-open", StateHalfOpen)
	}
}

func TestErrCircuitOpen(t *testing.T) {
	if ErrCircuitOpen.Error() != "circuit breaker is open" {
		t.Errorf("ErrCircuitOpen = %v", ErrCircuitOpen)
	}
}

func TestErrTooManyRequests(t *testing.T) {
	if ErrTooManyRequests.Error() != "too many requests in half-open state" {
		t.Errorf("ErrTooManyRequests = %v", ErrTooManyRequests)
	}
}
