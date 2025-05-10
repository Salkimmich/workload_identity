package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"sync"
	"time"

	"mTLS_demo/auth/common"
)

// Key represents an API key with its metadata
type Key struct {
	ID        string
	Hash      string
	Roles     []string
	ExpiresAt time.Time
	CreatedAt time.Time
	LastUsed  time.Time
}

// Store defines the interface for API key storage
type Store interface {
	GetKey(ctx context.Context, keyHash string) (*Key, error)
	UpdateLastUsed(ctx context.Context, keyID string) error
}

// InMemoryStore is a simple in-memory implementation of Store
type InMemoryStore struct {
	mu    sync.RWMutex
	keys  map[string]*Key
	byID  map[string]*Key
}

// NewInMemoryStore creates a new in-memory key store
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{
		keys: make(map[string]*Key),
		byID: make(map[string]*Key),
	}
}

// GetKey retrieves a key by its hash
func (s *InMemoryStore) GetKey(ctx context.Context, keyHash string) (*Key, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key, exists := s.keys[keyHash]
	if !exists {
		return nil, fmt.Errorf("key not found")
	}

	if !key.ExpiresAt.IsZero() && key.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("key expired")
	}

	return key, nil
}

// UpdateLastUsed updates the last used timestamp for a key
func (s *InMemoryStore) UpdateLastUsed(ctx context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.byID[keyID]
	if !exists {
		return fmt.Errorf("key not found")
	}

	key.LastUsed = time.Now()
	return nil
}

// AddKey adds a new key to the store
func (s *InMemoryStore) AddKey(key *Key) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.keys[key.Hash] = key
	s.byID[key.ID] = key
	return nil
}

// Middleware handles API key authentication
type Middleware struct {
	store       Store
	metrics     common.AuthMetricsCollector
	serviceName string
}

// Config holds the API key configuration
type Config struct {
	Store Store
}

// NewMiddleware creates a new API key middleware
func NewMiddleware(config *Config, serviceName string) (*Middleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if config.Store == nil {
		return nil, fmt.Errorf("store cannot be nil")
	}

	return &Middleware{
		store:       config.Store,
		metrics:     common.NewAuthMetricsCollector(),
		serviceName: serviceName,
	}, nil
}

// Middleware returns a middleware function that validates API keys
func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip validation for certain paths
		if m.shouldSkipValidation(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key from header
		keyString, err := m.ExtractKey(r)
		if err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "missing_key")
			http.Error(w, "API key required", http.StatusUnauthorized)
			return
		}

		// Hash the key
		keyHash := m.hashKey(keyString)

		// Look up the key
		key, err := m.store.GetKey(r.Context(), keyHash)
		if err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "invalid_key")
			http.Error(w, "Invalid API key", http.StatusUnauthorized)
			return
		}

		// Update last used timestamp
		if err := m.store.UpdateLastUsed(r.Context(), key.ID); err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "update_failed")
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		// Add authentication info to context
		ctx := r.Context()
		ctx = common.WithAuthMethod(ctx, common.AuthMethodAPIKey)
		ctx = common.WithServiceID(ctx, key.ID)
		ctx = common.WithRoles(ctx, key.Roles)
		r = r.WithContext(ctx)

		// Record successful authentication
		m.metrics.RecordAuthRequest(m.serviceName, string(common.AuthMethodAPIKey), "success", time.Since(start).Seconds())

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// shouldSkipValidation determines if authentication should be skipped
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

// ExtractKey extracts the API key from the request
func (m *Middleware) ExtractKey(r *http.Request) (string, error) {
	// Try X-API-Key header first
	key := r.Header.Get("X-API-Key")
	if key != "" {
		return key, nil
	}

	// Try Authorization header with Bearer scheme
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:], nil
	}

	return "", fmt.Errorf("API key not found")
}

// hashKey creates a SHA-256 hash of the API key
func (m *Middleware) hashKey(key string) string {
	hash := sha256.Sum256([]byte(key))
	return hex.EncodeToString(hash[:])
}

// RequireRole creates a middleware that checks for required roles
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get authentication method
			method, err := common.GetAuthMethodFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "missing_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify API key authentication
			if method != common.AuthMethodAPIKey {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "invalid_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get roles from context
			tokenRoles, err := common.GetRolesFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "missing_roles")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if any required role is present
			hasRole := false
			for _, requiredRole := range roles {
				for _, tokenRole := range tokenRoles {
					if requiredRole == tokenRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodAPIKey), "insufficient_roles")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
} 