package ratelimit

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter defines the interface for rate limiting
type Limiter interface {
	Allow(ctx context.Context, key string) bool
	Wait(ctx context.Context, key string) error
}

// RateLimiter implements the Limiter interface using golang.org/x/time/rate
type RateLimiter struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
	rate     rate.Limit
	burst    int
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond float64, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[string]*rate.Limiter),
		rate:     rate.Limit(requestsPerSecond),
		burst:    burst,
	}
}

// Allow checks if a request is allowed
func (r *RateLimiter) Allow(ctx context.Context, key string) bool {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if !exists {
		r.mu.Lock()
		limiter = rate.NewLimiter(r.rate, r.burst)
		r.limiters[key] = limiter
		r.mu.Unlock()
	}

	return limiter.Allow()
}

// Wait waits for a request to be allowed
func (r *RateLimiter) Wait(ctx context.Context, key string) error {
	r.mu.RLock()
	limiter, exists := r.limiters[key]
	r.mu.RUnlock()

	if !exists {
		r.mu.Lock()
		limiter = rate.NewLimiter(r.rate, r.burst)
		r.limiters[key] = limiter
		r.mu.Unlock()
	}

	return limiter.Wait(ctx)
}

// Middleware handles rate limiting
type Middleware struct {
	limiter     Limiter
	keyFunc     func(*http.Request) string
	waitOnLimit bool
}

// Config holds the rate limiting configuration
type Config struct {
	// Rate limiting settings
	RequestsPerSecond float64
	Burst            int

	// Key function to identify clients
	KeyFunc func(*http.Request) string

	// Whether to wait when rate limit is exceeded
	WaitOnLimit bool
}

// DefaultKeyFunc returns a key based on the client's IP address
func DefaultKeyFunc(r *http.Request) string {
	// Try to get the real IP address
	ip := r.Header.Get("X-Real-IP")
	if ip == "" {
		ip = r.Header.Get("X-Forwarded-For")
	}
	if ip == "" {
		ip = r.RemoteAddr
	}
	return ip
}

// NewMiddleware creates a new rate limiting middleware
func NewMiddleware(config *Config) (*Middleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.RequestsPerSecond <= 0 {
		return nil, fmt.Errorf("requests per second must be positive")
	}

	if config.Burst <= 0 {
		return nil, fmt.Errorf("burst must be positive")
	}

	keyFunc := config.KeyFunc
	if keyFunc == nil {
		keyFunc = DefaultKeyFunc
	}

	return &Middleware{
		limiter:     NewRateLimiter(config.RequestsPerSecond, config.Burst),
		keyFunc:     keyFunc,
		waitOnLimit: config.WaitOnLimit,
	}, nil
}

// Middleware returns a middleware function that enforces rate limits
func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip rate limiting for certain paths
		if m.shouldSkipValidation(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Get client key
		key := m.keyFunc(r)

		// Check rate limit
		if m.waitOnLimit {
			// Wait for rate limit
			if err := m.limiter.Wait(r.Context(), key); err != nil {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		} else {
			// Check if request is allowed
			if !m.limiter.Allow(r.Context(), key) {
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
				return
			}
		}

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// shouldSkipValidation determines if rate limiting should be skipped
func (m *Middleware) shouldSkipValidation(path string) bool {
	// Add paths that should skip validation
	skipPaths := []string{
		"/health",
		"/metrics",
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
} 