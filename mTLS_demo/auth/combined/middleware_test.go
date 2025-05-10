package combined

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"mTLS_demo/auth/jwt"
	"mTLS_demo/auth/mtls"

	"github.com/stretchr/testify/assert"
)

func setupTestMiddleware(t *testing.T) (*CombinedMiddleware, *jwt.TokenManager) {
	// Generate test keys
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// Create mTLS config
	mtlsConfig := &mtls.Config{
		TrustBundle: []*x509.Certificate{},
		AllowedCNs:  []string{"test-service"},
	}

	// Create JWT config
	jwtConfig := &jwt.Config{
		TokenManager: jwt.NewTokenManager(privateKey, &privateKey.PublicKey),
	}

	// Create combined config
	config := &Config{
		MTLSConfig:  mtlsConfig,
		JWTConfig:   jwtConfig,
		RequireMTLS: false,
		RequireJWT:  false,
		AllowBoth:   true,
	}

	// Create middleware
	middleware, err := NewCombinedMiddleware(config, "test-service")
	assert.NoError(t, err)

	return middleware, jwtConfig.TokenManager
}

func TestCombinedMiddleware_Middleware(t *testing.T) {
	middleware, tm := setupTestMiddleware(t)

	// Generate a valid JWT token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		path           string
		token          string
		tls            *tls.ConnectionState
		expectedStatus int
	}{
		{
			name:           "Valid JWT token",
			path:           "/api/test",
			token:          "Bearer " + token,
			tls:            nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Valid mTLS",
			path:           "/api/test",
			token:          "",
			tls:            &tls.ConnectionState{},
			expectedStatus: http.StatusUnauthorized, // Will fail because we don't have a valid cert
		},
		{
			name:           "Skip validation path",
			path:           "/health",
			token:          "",
			tls:            nil,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Invalid token",
			path:           "/api/test",
			token:          "Bearer invalid.token.here",
			tls:            nil,
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
			req := httptest.NewRequest("GET", tt.path, nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}
			if tt.tls != nil {
				req.TLS = tt.tls
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

func TestCombinedMiddleware_RequireRole(t *testing.T) {
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

func TestCombinedMiddleware_ContextValues(t *testing.T) {
	middleware, tm := setupTestMiddleware(t)

	// Generate a valid token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	// Create test handler that checks context values
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check auth method
		method, err := GetAuthMethodFromContext(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, AuthMethodJWT, method)

		// Check service ID
		serviceID, err := GetServiceIDFromContext(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, "test-service", serviceID)

		// Check roles
		roles, err := GetRolesFromContext(r.Context())
		assert.NoError(t, err)
		assert.Equal(t, []string{"admin"}, roles)

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

func TestCombinedMiddleware_ConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config - allow both",
			config: &Config{
				MTLSConfig:  &mtls.Config{},
				JWTConfig:   &jwt.Config{},
				RequireMTLS: false,
				RequireJWT:  false,
				AllowBoth:   true,
			},
			wantErr: false,
		},
		{
			name: "Valid config - require mTLS",
			config: &Config{
				MTLSConfig:  &mtls.Config{},
				JWTConfig:   &jwt.Config{},
				RequireMTLS: true,
				RequireJWT:  false,
				AllowBoth:   false,
			},
			wantErr: false,
		},
		{
			name: "Valid config - require JWT",
			config: &Config{
				MTLSConfig:  &mtls.Config{},
				JWTConfig:   &jwt.Config{},
				RequireMTLS: false,
				RequireJWT:  true,
				AllowBoth:   false,
			},
			wantErr: false,
		},
		{
			name: "Invalid config - missing mTLS config",
			config: &Config{
				JWTConfig:   &jwt.Config{},
				RequireMTLS: true,
				RequireJWT:  false,
				AllowBoth:   false,
			},
			wantErr: true,
		},
		{
			name: "Invalid config - missing JWT config",
			config: &Config{
				MTLSConfig:  &mtls.Config{},
				RequireMTLS: false,
				RequireJWT:  true,
				AllowBoth:   false,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewCombinedMiddleware(tt.config, "test-service")
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
} 