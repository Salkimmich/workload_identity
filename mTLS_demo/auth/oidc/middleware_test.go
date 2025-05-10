package oidc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mTLS_demo/auth/common"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
)

func setupTestMiddleware(t *testing.T) *Middleware {
	// Create a test OIDC provider
	provider, err := oidc.NewProvider(context.Background(), "https://test-issuer.com")
	if err != nil {
		t.Fatalf("Failed to create test provider: %v", err)
	}

	// Create test config
	config := &Config{
		IssuerURL:      "https://test-issuer.com",
		ClientID:       "test-client",
		ClientSecret:   "test-secret",
		RedirectURL:    "https://test-client.com/callback",
		Scopes:         []string{"openid", "profile", "email"},
		SkipIssuerCheck: true,
		SkipExpiryCheck: true,
	}

	// Create middleware
	middleware, err := NewMiddleware(config, "test-service")
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
		authHeader     string
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
			name:           "Missing authorization header",
			path:           "/api/test",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid authorization header format",
			path:           "/api/test",
			authHeader:     "Invalid",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			path:           "/api/test",
			authHeader:     "Bearer invalid-token",
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
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
	middleware := setupTestMiddleware(t)

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
			ctx = common.WithAuthMethod(ctx, common.AuthMethodOIDC)
			ctx = common.WithServiceID(ctx, "test-user")
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

func TestMiddleware_GetAuthURL(t *testing.T) {
	middleware := setupTestMiddleware(t)

	// Test getting auth URL
	state := "test-state"
	url := middleware.GetAuthURL(state)

	// Check that URL contains state
	assert.Contains(t, url, state)
}

func TestMiddleware_ExchangeCode(t *testing.T) {
	middleware := setupTestMiddleware(t)

	// Test exchanging code
	code := "test-code"
	token, err := middleware.ExchangeCode(context.Background(), code)

	// Check error (should fail in test environment)
	assert.Error(t, err)
	assert.Nil(t, token)
}

func TestMiddleware_RefreshToken(t *testing.T) {
	middleware := setupTestMiddleware(t)

	// Test refreshing token
	refreshToken := "test-refresh-token"
	token, err := middleware.RefreshToken(context.Background(), refreshToken)

	// Check error (should fail in test environment)
	assert.Error(t, err)
	assert.Nil(t, token)
} 