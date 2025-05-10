package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for circuit breaker monitoring
var (
	// Tracks current state of circuit breaker (0: closed, 1: open, 2: half-open)
	CircuitBreakerState = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "circuit_breaker_state",
			Help: "Current state of the circuit breaker (0: closed, 1: open, 2: half-open)",
		},
		[]string{"service"},
	)

	// Counts failures that triggered circuit breaker
	CircuitBreakerFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_failures_total",
			Help: "Number of failures that triggered the circuit breaker",
		},
		[]string{"service"},
	)

	// Counts successful calls after circuit breaker was triggered
	CircuitBreakerSuccesses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "circuit_breaker_successes_total",
			Help: "Number of successful calls after circuit breaker was triggered",
		},
		[]string{"service"},
	)

	// Measures number of retry attempts
	RetryAttempts = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_attempts_total",
			Help:    "Number of retry attempts made",
			Buckets: prometheus.LinearBuckets(0, 1, 10),
		},
		[]string{"service"},
	)

	// Measures time taken for retries
	RetryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "retry_latency_seconds",
			Help:    "Time taken for retries",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service"},
	)
)

// Circuit breaker states
const (
	StateClosed = iota    // Normal operation, requests allowed
	StateOpen            // Circuit open, requests blocked
	StateHalfOpen        // Testing if service recovered
)

// CircuitBreaker implements the circuit breaker pattern for fault tolerance
type CircuitBreaker struct {
	service     string        // Service name for metrics
	state       int          // Current state (closed/open/half-open)
	failures    int          // Consecutive failure count
	successes   int          // Success count in half-open state
	threshold   int          // Failure threshold to open circuit
	timeout     time.Duration // Time to wait before half-open
	lastFailure time.Time    // Time of last failure
	mutex       sync.RWMutex // Mutex for thread-safe state changes
}

// NewCircuitBreaker creates a new circuit breaker instance
func NewCircuitBreaker(service string, threshold int, timeout time.Duration) *CircuitBreaker {
	cb := &CircuitBreaker{
		service:   service,
		state:     StateClosed,
		threshold: threshold,
		timeout:   timeout,
	}
	CircuitBreakerState.WithLabelValues(service).Set(float64(StateClosed))
	return cb
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(f func() (interface{}, error)) (interface{}, error) {
	// Check current state
	cb.mutex.RLock()
	state := cb.state
	cb.mutex.RUnlock()

	// Handle open state with timeout
	switch state {
	case StateOpen:
		if time.Since(cb.lastFailure) > cb.timeout {
			// Move to half-open state after timeout
			cb.mutex.Lock()
			cb.state = StateHalfOpen
			cb.mutex.Unlock()
			CircuitBreakerState.WithLabelValues(cb.service).Set(float64(StateHalfOpen))
		} else {
			return nil, fmt.Errorf("circuit breaker is open")
		}
	}

	// Execute the function
	result, err := f()

	// Update state based on result
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if err != nil {
		// Handle failure
		cb.failures++
		CircuitBreakerFailures.WithLabelValues(cb.service).Inc()
		if cb.failures >= cb.threshold {
			// Open circuit if threshold reached
			cb.state = StateOpen
			cb.lastFailure = time.Now()
			CircuitBreakerState.WithLabelValues(cb.service).Set(float64(StateOpen))
		}
		return nil, err
	}

	if cb.state == StateHalfOpen {
		// Handle success in half-open state
		cb.successes++
		CircuitBreakerSuccesses.WithLabelValues(cb.service).Inc()
		if cb.successes >= cb.threshold/2 {
			// Close circuit if enough successes
			cb.state = StateClosed
			cb.failures = 0
			cb.successes = 0
			CircuitBreakerState.WithLabelValues(cb.service).Set(float64(StateClosed))
		}
	}

	return result, nil
}

// RetryPolicy implements retry logic with exponential backoff
type RetryPolicy struct {
	service     string        // Service name for metrics
	maxRetries  int          // Maximum number of retry attempts
	baseDelay   time.Duration // Initial delay between retries
	maxDelay    time.Duration // Maximum delay between retries
	jitter      float64       // Random jitter factor (0-1)
}

// NewRetryPolicy creates a new retry policy instance
func NewRetryPolicy(service string, maxRetries int, baseDelay, maxDelay time.Duration, jitter float64) *RetryPolicy {
	return &RetryPolicy{
		service:    service,
		maxRetries: maxRetries,
		baseDelay:  baseDelay,
		maxDelay:   maxDelay,
		jitter:     jitter,
	}
}

// Do executes the given function with retry logic
func (p *RetryPolicy) Do(ctx context.Context, f func() (interface{}, error)) (interface{}, error) {
	var lastErr error
	start := time.Now()

	// Attempt execution with retries
	for attempt := 0; attempt <= p.maxRetries; attempt++ {
		RetryAttempts.WithLabelValues(p.service).Observe(float64(attempt))

		// Try execution
		result, err := f()
		if err == nil {
			RetryLatency.WithLabelValues(p.service).Observe(time.Since(start).Seconds())
			return result, nil
		}

		lastErr = err

		// Break if max retries reached
		if attempt == p.maxRetries {
			break
		}

		// Calculate and wait for next retry
		delay := p.calculateDelay(attempt)
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(delay):
			continue
		}
	}

	RetryLatency.WithLabelValues(p.service).Observe(time.Since(start).Seconds())
	return nil, fmt.Errorf("max retries exceeded: %v", lastErr)
}

// calculateDelay calculates the delay for the next retry attempt with exponential backoff
func (p *RetryPolicy) calculateDelay(attempt int) time.Duration {
	// Calculate exponential delay
	delay := p.baseDelay * time.Duration(1<<uint(attempt))
	if delay > p.maxDelay {
		delay = p.maxDelay
	}

	// Add random jitter if configured
	if p.jitter > 0 {
		jitter := float64(delay) * p.jitter
		delay = delay + time.Duration(jitter)
	}

	return delay
} 