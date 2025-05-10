package ratelimit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware(t *testing.T) *Middleware {
	// Create test config
	config := &Config{
		RequestsPerSecond: 10,
		Burst:            5,
		WaitOnLimit:      false,
	}

	// Create middleware
	middleware, err := NewMiddleware(config)
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	return middleware
}

func TestMiddleware_Middleware(t *testing.T) {
	middleware := setupTestMiddleware(t)

	tests := []struct {
		name           string
		path           string
		remoteAddr     string
		expectedStatus int
	}{
		{
			name:           "Skip validation for health check",
			path:           "/health",
			remoteAddr:     "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Skip validation for metrics",
			path:           "/metrics",
			remoteAddr:     "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid request",
			path:           "/api/test",
			remoteAddr:     "127.0.0.1",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			req.RemoteAddr = tt.remoteAddr

			// Create test response recorder
			rr := httptest.NewRecorder()

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Apply middleware
			middleware.Middleware(handler).ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestMiddleware_RateLimit(t *testing.T) {
	// Create test config with low rate limit
	config := &Config{
		RequestsPerSecond: 1,
		Burst:            1,
		WaitOnLimit:      false,
	}

	// Create middleware
	middleware, err := NewMiddleware(config)
	assert.NoError(t, err)

	// Create test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "127.0.0.1"

	// Create test response recorder
	rr := httptest.NewRecorder()

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Make first request (should succeed)
	middleware.Middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Make second request immediately (should fail)
	rr = httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusTooManyRequests, rr.Code)

	// Wait for rate limit to reset
	time.Sleep(time.Second)

	// Make another request (should succeed)
	rr = httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestMiddleware_WaitOnLimit(t *testing.T) {
	// Create test config with wait on limit
	config := &Config{
		RequestsPerSecond: 1,
		Burst:            1,
		WaitOnLimit:      true,
	}

	// Create middleware
	middleware, err := NewMiddleware(config)
	assert.NoError(t, err)

	// Create test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.RemoteAddr = "127.0.0.1"

	// Create test response recorder
	rr := httptest.NewRecorder()

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Make first request (should succeed)
	middleware.Middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)

	// Make second request (should wait and succeed)
	rr = httptest.NewRecorder()
	middleware.Middleware(handler).ServeHTTP(rr, req)
	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestRateLimiter_Allow(t *testing.T) {
	limiter := NewRateLimiter(1, 1)

	// Test allowing first request
	assert.True(t, limiter.Allow(context.Background(), "test-key"))

	// Test rate limiting second request
	assert.False(t, limiter.Allow(context.Background(), "test-key"))

	// Test different keys
	assert.True(t, limiter.Allow(context.Background(), "different-key"))
}

func TestRateLimiter_Wait(t *testing.T) {
	limiter := NewRateLimiter(1, 1)

	// Test waiting for first request
	err := limiter.Wait(context.Background(), "test-key")
	assert.NoError(t, err)

	// Test waiting for second request
	err = limiter.Wait(context.Background(), "test-key")
	assert.NoError(t, err) // Should wait and succeed

	// Test context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	err = limiter.Wait(ctx, "test-key")
	assert.Error(t, err)
}

func TestDefaultKeyFunc(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		expected   string
	}{
		{
			name:       "Use X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "1.2.3.4"},
			remoteAddr: "127.0.0.1",
			expected:   "1.2.3.4",
		},
		{
			name:       "Use X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "1.2.3.4"},
			remoteAddr: "127.0.0.1",
			expected:   "1.2.3.4",
		},
		{
			name:       "Use RemoteAddr",
			headers:    map[string]string{},
			remoteAddr: "127.0.0.1",
			expected:   "127.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/", nil)
			req.RemoteAddr = tt.remoteAddr

			// Add headers
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			// Test key function
			key := DefaultKeyFunc(req)
			assert.Equal(t, tt.expected, key)
		})
	}
} 