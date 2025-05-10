package jwt

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestHandlers(t *testing.T) (*TokenHandler, *RefreshHandler, *TokenManager) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}

	tm, err := NewTokenManager(privateKey, &privateKey.PublicKey)
	if err != nil {
		t.Fatalf("Failed to create token manager: %v", err)
	}

	tokenHandler := NewTokenHandler(tm, "test-service")
	refreshHandler := NewRefreshHandler(tm, "test-service")
	return tokenHandler, refreshHandler, tm
}

func TestTokenHandler_ServeHTTP(t *testing.T) {
	tokenHandler, _, _ := setupTestHandlers(t)

	tests := []struct {
		name           string
		method         string
		request        TokenRequest
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:   "Valid request",
			method: http.MethodPost,
			request: TokenRequest{
				ServiceID: "test-service",
				Roles:     []string{"admin"},
				Scope:     "read:write",
			},
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:   "Invalid method",
			method: http.MethodGet,
			request: TokenRequest{
				ServiceID: "test-service",
				Roles:     []string{"admin"},
			},
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
		{
			name:   "Missing service ID",
			method: http.MethodPost,
			request: TokenRequest{
				Roles: []string{"admin"},
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  false,
		},
		{
			name:   "Missing roles",
			method: http.MethodPost,
			request: TokenRequest{
				ServiceID: "test-service",
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request body
			body, err := json.Marshal(tt.request)
			assert.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(tt.method, "/auth/token", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			tokenHandler.ServeHTTP(rr, req)

			// Check response status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check response body if needed
			if tt.checkResponse {
				var response TokenResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.AccessToken)
				assert.Equal(t, "Bearer", response.TokenType)
				assert.Greater(t, response.ExpiresIn, int64(0))
				assert.True(t, response.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestRefreshHandler_ServeHTTP(t *testing.T) {
	_, refreshHandler, tm := setupTestHandlers(t)

	// Generate a valid token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name           string
		method         string
		token          string
		expectedStatus int
		checkResponse  bool
	}{
		{
			name:           "Valid refresh",
			method:         http.MethodPost,
			token:          "Bearer " + token,
			expectedStatus: http.StatusOK,
			checkResponse:  true,
		},
		{
			name:           "Invalid method",
			method:         http.MethodGet,
			token:          "Bearer " + token,
			expectedStatus: http.StatusMethodNotAllowed,
			checkResponse:  false,
		},
		{
			name:           "Missing token",
			method:         http.MethodPost,
			token:          "",
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
		{
			name:           "Invalid token",
			method:         http.MethodPost,
			token:          "Bearer invalid.token.here",
			expectedStatus: http.StatusUnauthorized,
			checkResponse:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test request
			req := httptest.NewRequest(tt.method, "/auth/refresh", nil)
			if tt.token != "" {
				req.Header.Set("Authorization", tt.token)
			}

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			refreshHandler.ServeHTTP(rr, req)

			// Check response status
			assert.Equal(t, tt.expectedStatus, rr.Code)

			// Check response body if needed
			if tt.checkResponse {
				var response TokenResponse
				err := json.NewDecoder(rr.Body).Decode(&response)
				assert.NoError(t, err)
				assert.NotEmpty(t, response.AccessToken)
				assert.Equal(t, "Bearer", response.TokenType)
				assert.Greater(t, response.ExpiresIn, int64(0))
				assert.True(t, response.ExpiresAt.After(time.Now()))
			}
		})
	}
}

func TestTokenHandler_ConcurrentRequests(t *testing.T) {
	tokenHandler, _, _ := setupTestHandlers(t)
	concurrentRequests := 10
	done := make(chan bool)

	for i := 0; i < concurrentRequests; i++ {
		go func() {
			// Create request body
			request := TokenRequest{
				ServiceID: "test-service",
				Roles:     []string{"admin"},
				Scope:     "read:write",
			}
			body, err := json.Marshal(request)
			assert.NoError(t, err)

			// Create test request
			req := httptest.NewRequest(http.MethodPost, "/auth/token", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			// Create response recorder
			rr := httptest.NewRecorder()

			// Call handler
			tokenHandler.ServeHTTP(rr, req)

			// Check response
			assert.Equal(t, http.StatusOK, rr.Code)

			var response TokenResponse
			err = json.NewDecoder(rr.Body).Decode(&response)
			assert.NoError(t, err)
			assert.NotEmpty(t, response.AccessToken)

			done <- true
		}()
	}

	// Wait for all requests to complete
	for i := 0; i < concurrentRequests; i++ {
		<-done
	}
} 