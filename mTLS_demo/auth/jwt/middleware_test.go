package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware(t *testing.T) (*JWTMiddleware, *TokenManager) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	tm, err := NewTokenManager(privateKey, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create token manager: %v", err)
	}

	middleware := NewJWTMiddleware(tm, "test-service")
	return middleware, tm
}

func TestJWTMiddleware_Middleware(t *testing.T) {
	middleware, tm := setupTestMiddleware(t)

	// Generate a valid token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		token          string
		expectedStatus int
	}{
		{
			name:           "Valid token",
			path:           "/api/test",
			token:          "Bearer " + token,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing token",
			path:           "/api/test",
			token:          "",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Invalid token",
			path:           "/api/test",
			token:          "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name:           "Skip validation path",
			path:           "/health",
			token:          "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create test request
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Apply middleware
			middleware.Middleware(handler).ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestJWTMiddleware_RequireRole(t *testing.T) {
	middleware, tm := setupTestMiddleware(t)

	// Generate tokens with different roles
	adminToken, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	userToken, err := tm.GenerateToken("test-service", []string{"user"}, "read", time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		token          string
		requiredRoles  []string
		expectedStatus int
	}{
		{
			name:           "Has required role",
			token:          "Bearer " + adminToken,
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing required role",
			token:          "Bearer " + userToken,
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "Multiple roles - has one",
			token:          "Bearer " + adminToken,
			requiredRoles:  []string{"user", "admin"},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Missing token",
			token:          "",
			requiredRoles:  []string{"admin"},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test handler
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusOK)
			})

			// Create test request
			req := httptest.NewRequest("GET", "/api/test", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Apply middleware chain
			middleware.Middleware(middleware.RequireRole(tt.requiredRoles...)(handler)).ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}

func TestJWTMiddleware_ContextValues(t *testing.T) {
	middleware, tm := setupTestMiddleware(t)

	// Generate a valid token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	// Create test handler that checks context values
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check token in context
		ctxToken, err := GetTokenFromContext(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, token, ctxToken)

		// Check claims in context
		claims, err := GetClaimsFromContext(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "test-service", claims.ServiceID)
		assert.Equal(t, []string{"admin"}, claims.Roles)
		assert.Equal(t, "read:write", claims.Scope)

		w.WriteHeader(http.StatusOK)
	})

	// Create test request
	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	// Create response recorder
	rr := httptest.NewRecorder()

	// Apply middleware
	middleware.Middleware(handler).ServeHTTP(rr, req)

	// Check response
	assert.Equal(t, http.StatusOK, rr.Code)
} 