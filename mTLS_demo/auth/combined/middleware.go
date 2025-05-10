package combined

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"mTLS_demo/auth/common"
	"mTLS_demo/auth/jwt"
	"mTLS_demo/auth/mtls"
)

// ContextKey is a type for context keys
type ContextKey string

const (
	// AuthMethodContextKey is the key for storing the authentication method in context
	AuthMethodContextKey ContextKey = "auth_method"
	// ServiceIDContextKey is the key for storing the service ID in context
	ServiceIDContextKey ContextKey = "service_id"
	// RolesContextKey is the key for storing the roles in context
	RolesContextKey ContextKey = "roles"
)

// AuthMethod represents the authentication method used
type AuthMethod string

const (
	// AuthMethodMTLS represents mTLS authentication
	AuthMethodMTLS AuthMethod = "mtls"
	// AuthMethodJWT represents JWT authentication
	AuthMethodJWT AuthMethod = "jwt"
)

// CombinedMiddleware handles both mTLS and JWT authentication
type CombinedMiddleware struct {
	mtlsMiddleware *mtls.Middleware
	jwtMiddleware  *jwt.JWTMiddleware
	metrics        *common.AuthMetricsCollector
	serviceName    string
	config         *Config
}

// Config holds the combined authentication configuration
type Config struct {
	// mTLS configuration
	MTLSConfig *mtls.Config
	// JWT configuration
	JWTConfig *jwt.Config
	// Combined settings
	RequireMTLS bool
	RequireJWT  bool
	AllowBoth   bool
}

// NewCombinedMiddleware creates a new combined authentication middleware
func NewCombinedMiddleware(config *Config, serviceName string) (*CombinedMiddleware, error) {
	mtlsMiddleware, err := mtls.NewMiddleware(config.MTLSConfig, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create mTLS middleware: %v", err)
	}

	jwtMiddleware := jwt.NewJWTMiddleware(config.JWTConfig.TokenManager, serviceName)

	return &CombinedMiddleware{
		mtlsMiddleware: mtlsMiddleware,
		jwtMiddleware:  jwtMiddleware,
		metrics:        common.NewAuthMetricsCollector(),
		serviceName:    serviceName,
		config:         config,
	}, nil
}

// Middleware returns a middleware function that validates both mTLS and JWT
func (m *CombinedMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip validation for certain paths
		if m.shouldSkipValidation(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Try mTLS authentication first
		mtlsSuccess := false
		if m.config.RequireMTLS || m.config.AllowBoth {
			mtlsSuccess = m.tryMTLSAuth(r)
		}

		// Try JWT authentication if needed
		jwtSuccess := false
		if m.config.RequireJWT || m.config.AllowBoth {
			jwtSuccess = m.tryJWTAuth(r)
		}

		// Check if authentication requirements are met
		if !m.isAuthenticationValid(mtlsSuccess, jwtSuccess) {
			m.metrics.RecordAuthError(m.serviceName, "combined", "authentication_failed")
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		// Record successful authentication
		m.metrics.RecordAuthRequest(m.serviceName, "combined", "success", time.Since(start))

		// Call next handler
		next.ServeHTTP(w, r)
	})
}

// shouldSkipValidation determines if authentication should be skipped
func (m *CombinedMiddleware) shouldSkipValidation(path string) bool {
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

// tryMTLSAuth attempts mTLS authentication
func (m *CombinedMiddleware) tryMTLSAuth(r *http.Request) bool {
	// Check if TLS connection exists
	if r.TLS == nil {
		return false
	}

	// Verify client certificate
	if len(r.TLS.PeerCertificates) == 0 {
		return false
	}

	// Get client certificate
	clientCert := r.TLS.PeerCertificates[0]

	// Verify certificate
	if err := m.mtlsMiddleware.VerifyCertificate(clientCert); err != nil {
		return false
	}

	// Add authentication info to context
	ctx := context.WithValue(r.Context(), AuthMethodContextKey, AuthMethodMTLS)
	ctx = context.WithValue(ctx, ServiceIDContextKey, clientCert.Subject.CommonName)
	*r = *r.WithContext(ctx)

	return true
}

// tryJWTAuth attempts JWT authentication
func (m *CombinedMiddleware) tryJWTAuth(r *http.Request) bool {
	// Extract token from Authorization header
	tokenString, err := m.jwtMiddleware.ExtractToken(r)
	if err != nil {
		return false
	}

	// Verify token
	claims, err := m.jwtMiddleware.TokenManager.VerifyToken(tokenString)
	if err != nil {
		return false
	}

	// Validate claims
	if err := m.jwtMiddleware.TokenManager.ValidateClaims(claims); err != nil {
		return false
	}

	// Add authentication info to context
	ctx := context.WithValue(r.Context(), AuthMethodContextKey, AuthMethodJWT)
	ctx = context.WithValue(ctx, ServiceIDContextKey, claims.ServiceID)
	ctx = context.WithValue(ctx, RolesContextKey, claims.Roles)
	*r = *r.WithContext(ctx)

	return true
}

// isAuthenticationValid checks if the authentication requirements are met
func (m *CombinedMiddleware) isAuthenticationValid(mtlsSuccess, jwtSuccess bool) bool {
	if m.config.RequireMTLS && !mtlsSuccess {
		return false
	}

	if m.config.RequireJWT && !jwtSuccess {
		return false
	}

	if m.config.AllowBoth {
		return mtlsSuccess || jwtSuccess
	}

	return true
}

// GetAuthMethodFromContext extracts the authentication method from context
func GetAuthMethodFromContext(ctx context.Context) (AuthMethod, error) {
	method, ok := ctx.Value(AuthMethodContextKey).(AuthMethod)
	if !ok {
		return "", fmt.Errorf("auth method not found in context")
	}
	return method, nil
}

// GetServiceIDFromContext extracts the service ID from context
func GetServiceIDFromContext(ctx context.Context) (string, error) {
	serviceID, ok := ctx.Value(ServiceIDContextKey).(string)
	if !ok {
		return "", fmt.Errorf("service ID not found in context")
	}
	return serviceID, nil
}

// GetRolesFromContext extracts the roles from context
func GetRolesFromContext(ctx context.Context) ([]string, error) {
	roles, ok := ctx.Value(RolesContextKey).([]string)
	if !ok {
		return nil, fmt.Errorf("roles not found in context")
	}
	return roles, nil
}

// RequireRole creates a middleware that checks for required roles
func (m *CombinedMiddleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get authentication method
			method, err := GetAuthMethodFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, "role_check", "missing_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// For mTLS, check certificate roles
			if method == AuthMethodMTLS {
				if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
					m.metrics.RecordAuthError(m.serviceName, "role_check", "missing_certificate")
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

				certRoles := m.mtlsMiddleware.GetCertificateRoles(r.TLS.PeerCertificates[0])
				hasRole := false
				for _, requiredRole := range roles {
					for _, certRole := range certRoles {
						if requiredRole == certRole {
							hasRole = true
							break
						}
					}
					if hasRole {
						break
					}
				}

				if !hasRole {
					m.metrics.RecordAuthError(m.serviceName, "role_check", "insufficient_roles")
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			// For JWT, check token roles
			if method == AuthMethodJWT {
				tokenRoles, err := GetRolesFromContext(r.Context())
				if err != nil {
					m.metrics.RecordAuthError(m.serviceName, "role_check", "missing_roles")
					http.Error(w, "Unauthorized", http.StatusUnauthorized)
					return
				}

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
					m.metrics.RecordAuthError(m.serviceName, "role_check", "insufficient_roles")
					http.Error(w, "Forbidden", http.StatusForbidden)
					return
				}
			}

			next.ServeHTTP(w, r)
		})
	}
} 