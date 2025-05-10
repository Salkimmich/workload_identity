package jwt

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mTLS_demo/auth/common"
)

// TokenRequest represents a token request
type TokenRequest struct {
	ServiceID string   `json:"service_id"`
	Roles     []string `json:"roles"`
	Scope     string   `json:"scope,omitempty"`
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresIn   int64     `json:"expires_in"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// TokenHandler handles token requests
type TokenHandler struct {
	tokenManager *TokenManager
	metrics      *common.AuthMetricsCollector
	serviceName  string
}

// NewTokenHandler creates a new token handler
func NewTokenHandler(tokenManager *TokenManager, serviceName string) *TokenHandler {
	return &TokenHandler{
		tokenManager: tokenManager,
		metrics:      common.NewAuthMetricsCollector(),
		serviceName:  serviceName,
	}
}

// ServeHTTP handles token requests
func (h *TokenHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only allow POST requests
	if r.Method != http.MethodPost {
		h.metrics.RecordAuthError(h.serviceName, "token", "invalid_method")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var req TokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.metrics.RecordAuthError(h.serviceName, "token", "invalid_request")
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Validate request
	if req.ServiceID == "" {
		h.metrics.RecordAuthError(h.serviceName, "token", "missing_service_id")
		http.Error(w, "Service ID is required", http.StatusBadRequest)
		return
	}

	if len(req.Roles) == 0 {
		h.metrics.RecordAuthError(h.serviceName, "token", "missing_roles")
		http.Error(w, "At least one role is required", http.StatusBadRequest)
		return
	}

	// Generate token
	tokenDuration := 1 * time.Hour // Configurable
	token, err := h.tokenManager.GenerateToken(req.ServiceID, req.Roles, req.Scope, tokenDuration)
	if err != nil {
		h.metrics.RecordAuthError(h.serviceName, "token", "generation_failed")
		http.Error(w, "Failed to generate token", http.StatusInternalServerError)
		return
	}

	// Create response
	expiresAt := time.Now().Add(tokenDuration)
	resp := TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenDuration.Seconds()),
		ExpiresAt:   expiresAt,
	}

	// Record metrics
	h.metrics.RecordAuthRequest(h.serviceName, "token", "success", time.Since(start))
	h.metrics.RecordNewToken(h.serviceName)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// RefreshHandler handles token refresh requests
type RefreshHandler struct {
	tokenManager *TokenManager
	metrics      *common.AuthMetricsCollector
	serviceName  string
}

// NewRefreshHandler creates a new refresh handler
func NewRefreshHandler(tokenManager *TokenManager, serviceName string) *RefreshHandler {
	return &RefreshHandler{
		tokenManager: tokenManager,
		metrics:      common.NewAuthMetricsCollector(),
		serviceName:  serviceName,
	}
}

// ServeHTTP handles token refresh requests
func (h *RefreshHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// Only allow POST requests
	if r.Method != http.MethodPost {
		h.metrics.RecordAuthError(h.serviceName, "refresh", "invalid_method")
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get token from Authorization header
	tokenString, err := extractTokenFromHeader(r)
	if err != nil {
		h.metrics.RecordAuthError(h.serviceName, "refresh", "missing_token")
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Refresh token
	tokenDuration := 1 * time.Hour // Configurable
	newToken, err := h.tokenManager.RefreshToken(tokenString, tokenDuration)
	if err != nil {
		h.metrics.RecordAuthError(h.serviceName, "refresh", "refresh_failed")
		http.Error(w, "Failed to refresh token", http.StatusUnauthorized)
		return
	}

	// Create response
	expiresAt := time.Now().Add(tokenDuration)
	resp := TokenResponse{
		AccessToken: newToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(tokenDuration.Seconds()),
		ExpiresAt:   expiresAt,
	}

	// Record metrics
	h.metrics.RecordAuthRequest(h.serviceName, "refresh", "success", time.Since(start))
	h.metrics.RecordNewToken(h.serviceName)

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// extractTokenFromHeader extracts the token from the Authorization header
func extractTokenFromHeader(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("missing Authorization header")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid Authorization header format")
	}

	return parts[1], nil
} 