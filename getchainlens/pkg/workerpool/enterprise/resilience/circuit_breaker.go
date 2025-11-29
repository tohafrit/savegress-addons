package resilience

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

var (
	// ErrCircuitOpen is returned when circuit is open
	ErrCircuitOpen = errors.New("circuit breaker is open")
	// ErrTooManyRequests is returned in half-open state with too many requests
	ErrTooManyRequests = errors.New("too many requests in half-open state")
)

// CircuitState represents the circuit breaker state
type CircuitState string

const (
	StateClosed   CircuitState = "closed"
	StateOpen     CircuitState = "open"
	StateHalfOpen CircuitState = "half-open"
)

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	name             string
	state            atomic.Value // CircuitState
	failures         atomic.Int32
	successes        atomic.Int32
	consecutiveFails atomic.Int32
	lastFailTime     atomic.Value // time.Time

	failureThreshold  int
	successThreshold  int
	timeout           time.Duration
	halfOpenMaxCalls  int
	halfOpenCalls     atomic.Int32

	onStateChange func(from, to CircuitState)
	mu            sync.RWMutex
}

// CircuitBreakerConfig configures a circuit breaker
type CircuitBreakerConfig struct {
	Name             string
	FailureThreshold int           // Open after N consecutive failures
	SuccessThreshold int           // Close after N consecutive successes
	Timeout          time.Duration // Half-open after timeout
	HalfOpenMaxCalls int           // Max concurrent calls in half-open
	OnStateChange    func(from, to CircuitState)
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	cb := &CircuitBreaker{
		name:              config.Name,
		failureThreshold:  config.FailureThreshold,
		successThreshold:  config.SuccessThreshold,
		timeout:           config.Timeout,
		halfOpenMaxCalls:  config.HalfOpenMaxCalls,
		onStateChange:     config.OnStateChange,
	}

	cb.state.Store(StateClosed)
	cb.lastFailTime.Store(time.Time{})

	return cb
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	state := cb.getState()

	switch state {
	case StateOpen:
		// Check if timeout elapsed
		lastFail := cb.lastFailTime.Load().(time.Time)
		if !lastFail.IsZero() && time.Since(lastFail) > cb.timeout {
			cb.setState(StateHalfOpen)
			return cb.Call(fn) // Retry in half-open state
		}
		return ErrCircuitOpen

	case StateHalfOpen:
		// Check concurrent calls limit
		calls := cb.halfOpenCalls.Load()
		if calls >= int32(cb.halfOpenMaxCalls) {
			return ErrTooManyRequests
		}

		cb.halfOpenCalls.Add(1)
		defer cb.halfOpenCalls.Add(-1)

		err := fn()
		if err != nil {
			cb.recordFailure()
			cb.setState(StateOpen)
			return err
		}

		cb.recordSuccess()
		// Check if we should close
		if cb.successes.Load() >= int32(cb.successThreshold) {
			cb.setState(StateClosed)
		}
		return nil

	case StateClosed:
		err := fn()
		if err != nil {
			cb.recordFailure()
			// Check if we should open
			if cb.consecutiveFails.Load() >= int32(cb.failureThreshold) {
				cb.setState(StateOpen)
			}
			return err
		}

		cb.recordSuccess()
		return nil

	default:
		return errors.New("invalid circuit breaker state")
	}
}

// getState gets the current state
func (cb *CircuitBreaker) getState() CircuitState {
	return cb.state.Load().(CircuitState)
}

// setState sets the state and triggers callbacks
func (cb *CircuitBreaker) setState(newState CircuitState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	oldState := cb.state.Load().(CircuitState)
	if oldState == newState {
		return
	}

	cb.state.Store(newState)

	// Reset counters on state change
	switch newState {
	case StateClosed:
		cb.consecutiveFails.Store(0)
		cb.successes.Store(0)
		cb.halfOpenCalls.Store(0)
	case StateOpen:
		cb.successes.Store(0)
		cb.lastFailTime.Store(time.Now())
	case StateHalfOpen:
		cb.consecutiveFails.Store(0)
		cb.successes.Store(0)
		cb.halfOpenCalls.Store(0)
	}

	if cb.onStateChange != nil {
		go cb.onStateChange(oldState, newState)
	}
}

// recordFailure records a failure
func (cb *CircuitBreaker) recordFailure() {
	cb.failures.Add(1)
	cb.consecutiveFails.Add(1)
	cb.successes.Store(0) // Reset consecutive successes
	cb.lastFailTime.Store(time.Now())
}

// recordSuccess records a success
func (cb *CircuitBreaker) recordSuccess() {
	cb.successes.Add(1)
	cb.consecutiveFails.Store(0) // Reset consecutive failures
}

// GetState returns the current state
func (cb *CircuitBreaker) GetState() CircuitState {
	return cb.getState()
}

// GetMetrics returns circuit breaker metrics
func (cb *CircuitBreaker) GetMetrics() CircuitBreakerMetrics {
	return CircuitBreakerMetrics{
		Name:             cb.name,
		State:            string(cb.getState()),
		Failures:         int(cb.failures.Load()),
		Successes:        int(cb.successes.Load()),
		ConsecutiveFails: int(cb.consecutiveFails.Load()),
	}
}

// CircuitBreakerMetrics contains circuit breaker metrics
type CircuitBreakerMetrics struct {
	Name             string
	State            string
	Failures         int
	Successes        int
	ConsecutiveFails int
}

// Reset resets the circuit breaker to closed state
func (cb *CircuitBreaker) Reset() {
	cb.setState(StateClosed)
	cb.failures.Store(0)
	cb.successes.Store(0)
	cb.consecutiveFails.Store(0)
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers sync.Map // map[string]*CircuitBreaker
	config   CircuitBreakerConfig
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager(config CircuitBreakerConfig) *CircuitBreakerManager {
	return &CircuitBreakerManager{
		config: config,
	}
}

// Get gets or creates a circuit breaker for a task type
func (cbm *CircuitBreakerManager) Get(taskType string) *CircuitBreaker {
	cbI, _ := cbm.breakers.LoadOrStore(taskType, NewCircuitBreaker(CircuitBreakerConfig{
		Name:             taskType,
		FailureThreshold: cbm.config.FailureThreshold,
		SuccessThreshold: cbm.config.SuccessThreshold,
		Timeout:          cbm.config.Timeout,
		HalfOpenMaxCalls: cbm.config.HalfOpenMaxCalls,
		OnStateChange:    cbm.config.OnStateChange,
	}))
	return cbI.(*CircuitBreaker)
}

// GetAll returns all circuit breakers
func (cbm *CircuitBreakerManager) GetAll() map[string]*CircuitBreaker {
	result := make(map[string]*CircuitBreaker)
	cbm.breakers.Range(func(key, value interface{}) bool {
		result[key.(string)] = value.(*CircuitBreaker)
		return true
	})
	return result
}
