package apikey

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mTLS_demo/auth/common"

	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware(t *testing.T) (*Middleware, *InMemoryStore) {
	// Create test store
	store := NewInMemoryStore()

	// Add test key
	key := &Key{
		ID:        "test-key",
		Hash:      "test-hash",
		Roles:     []string{"user", "admin"},
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	err := store.AddKey(key)
	if err != nil {
		t.Fatalf("Failed to add test key: %v", err)
	}

	// Create test config
	config := &Config{
		Store: store,
	}

	// Create middleware
	middleware, err := NewMiddleware(config, "test-service")
	if err != nil {
		t.Fatalf("Failed to create middleware: %v", err)
	}

	return middleware, store
}

func TestMiddleware_Middleware(t *testing.T) {
	middleware, _ := setupTestMiddleware(t)

	tests := []struct {
		name           string
		path           string
		apiKey         string
		expectedStatus int
	}{
		{
			name:           "Skip validation for health check",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Skip validation for metrics",
			path:           "/metrics",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing API key",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid API key",
			path:           "/api/test",
			apiKey:         "invalid-key",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Valid API key in X-API-Key header",
			path:           "/api/test",
			apiKey:         "test-key",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid API key in Authorization header",
			path:           "/api/test",
			apiKey:         "Bearer test-key",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.apiKey != "" {
				if len(tt.apiKey) > 7 && tt.apiKey[:7] == "Bearer " {
					req.Header.Set("Authorization", tt.apiKey)
				} else {
					req.Header.Set("X-API-Key", tt.apiKey)
				}
			}

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

func TestMiddleware_RequireRole(t *testing.T) {
	middleware, _ := setupTestMiddleware(t)

	tests := []struct {
		name           string
		roles          []string
		contextRoles   []string
		expectedStatus int
	}{
		{
			name:           "No roles required",
			roles:          []string{},
			contextRoles:   []string{"user"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Required role present",
			roles:          []string{"admin"},
			contextRoles:   []string{"user", "admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Required role not present",
			roles:          []string{"admin"},
			contextRoles:   []string{"user"},
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", "/api/test", nil)

			// Add authentication info to context
			ctx := req.Context()
			ctx = common.WithAuthMethod(ctx, common.AuthMethodAPIKey)
			ctx = common.WithServiceID(ctx, "test-key")
			ctx = common.WithRoles(ctx, tt.contextRoles)
			req = req.WithContext(ctx)

			// Create test response recorder
			rr := httptest.NewRecorder()

			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Apply role middleware
			middleware.RequireRole(tt.roles...)(handler).ServeHTTP(rr, req)

			// Check status code
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestInMemoryStore_GetKey(t *testing.T) {
	store := NewInMemoryStore()

	// Add test key
	key := &Key{
		ID:        "test-key",
		Hash:      "test-hash",
		Roles:     []string{"user"},
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	err := store.AddKey(key)
	assert.NoError(t, err)

	// Test getting existing key
	retrievedKey, err := store.GetKey(context.Background(), "test-hash")
	assert.NoError(t, err)
	assert.Equal(t, key.ID, retrievedKey.ID)

	// Test getting non-existent key
	_, err = store.GetKey(context.Background(), "non-existent")
	assert.Error(t, err)

	// Test expired key
	expiredKey := &Key{
		ID:        "expired-key",
		Hash:      "expired-hash",
		Roles:     []string{"user"},
		ExpiresAt: time.Now().Add(-time.Hour),
		CreatedAt: time.Now().Add(-2 * time.Hour),
	}
	err = store.AddKey(expiredKey)
	assert.NoError(t, err)

	_, err = store.GetKey(context.Background(), "expired-hash")
	assert.Error(t, err)
}

func TestInMemoryStore_UpdateLastUsed(t *testing.T) {
	store := NewInMemoryStore()

	// Add test key
	key := &Key{
		ID:        "test-key",
		Hash:      "test-hash",
		Roles:     []string{"user"},
		CreatedAt: time.Now(),
	}
	err := store.AddKey(key)
	assert.NoError(t, err)

	// Test updating last used
	err = store.UpdateLastUsed(context.Background(), "test-key")
	assert.NoError(t, err)

	// Verify last used was updated
	retrievedKey, err := store.GetKey(context.Background(), "test-hash")
	assert.NoError(t, err)
	assert.True(t, retrievedKey.LastUsed.After(key.CreatedAt))

	// Test updating non-existent key
	err = store.UpdateLastUsed(context.Background(), "non-existent")
	assert.Error(t, err)
} 