package jwt

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"mTLS_demo/auth/common"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// TokenContextKey is the key for storing the token in context
	TokenContextKey ContextKey = "token"
	// ClaimsContextKey is the key for storing the claims in context
	ClaimsContextKey ContextKey = "claims"
)

// JWTMiddleware handles JWT authentication
type JWTMiddleware struct {
	tokenManager *TokenManager
	metrics      common.AuthMetricsCollector
	serviceName  string
}

// NewJWTMiddleware creates a new JWT middleware
func NewJWTMiddleware(tokenManager *TokenManager, serviceName string) *JWTMiddleware {
	return &JWTMiddleware{
		tokenManager: tokenManager,
		metrics:      common.NewAuthMetricsCollector(),
		serviceName:  serviceName,
	}
}

// Middleware returns a middleware function that validates JWT
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip validation for certain paths
		if m.shouldSkipValidation(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		tokenString, err := m.ExtractToken(r)
		if err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "missing_token")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Verify token
		claims, err := m.tokenManager.VerifyToken(tokenString)
		if err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "invalid_token")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Validate claims
		if err := m.tokenManager.ValidateClaims(claims); err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "invalid_claims")
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Add authentication info to context
		ctx := r.Context()
		ctx = common.WithAuthMethod(ctx, common.AuthMethodJWT)
		ctx = common.WithServiceID(ctx, claims.ServiceID)
		ctx = common.WithRoles(ctx, claims.Roles)
		r = r.WithContext(ctx)

		// Record successful authentication
		m.metrics.RecordAuthRequest(m.serviceName, string(common.AuthMethodJWT), "success", time.Since(start).Seconds())

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// shouldSkipValidation determines if authentication should be skipped
func (m *JWTMiddleware) shouldSkipValidation(path string) bool {
	// Add paths that should skip validation
	skipPaths := []string{
		"/health",
		"/metrics",
		"/auth/token", // Token endpoint
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// ExtractToken extracts the JWT from the Authorization header
func (m *JWTMiddleware) ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", fmt.Errorf("invalid authorization header format")
	}

	return parts[1], nil
}

// GetTokenFromContext extracts the token from context
func GetTokenFromContext(ctx context.Context) (string, error) {
	token, ok := ctx.Value(TokenContextKey).(string)
	if !ok {
		return "", fmt.Errorf("token not found in context")
	}
	return token, nil
}

// GetClaimsFromContext extracts the claims from context
func GetClaimsFromContext(ctx context.Context) (*TokenClaims, error) {
	claims, ok := ctx.Value(ClaimsContextKey).(*TokenClaims)
	if !ok {
		return nil, fmt.Errorf("claims not found in context")
	}
	return claims, nil
}

// RequireRole creates a middleware that checks for required roles
func (m *JWTMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get authentication method
			method, err := common.GetAuthMethodFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "missing_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify JWT authentication
			if method != common.AuthMethodJWT {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "invalid_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get roles from context
			tokenRoles, err := common.GetRolesFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "missing_roles")
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
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodJWT), "insufficient_roles")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
} 