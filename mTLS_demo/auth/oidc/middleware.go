package oidc

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"mTLS_demo/auth/common"

	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// Middleware handles OIDC authentication
type Middleware struct {
	provider    *oidc.Provider
	verifier    *oidc.IDTokenVerifier
	config      *oauth2.Config
	metrics     common.AuthMetricsCollector
	serviceName string
}

// Config holds the OIDC configuration
type Config struct {
	// OIDC Provider configuration
	IssuerURL      string
	ClientID       string
	ClientSecret   string
	RedirectURL    string
	Scopes         []string

	// Token validation settings
	SkipIssuerCheck bool
	SkipExpiryCheck bool

	// Additional settings
	AllowedAudiences []string
	AllowedIssuers   []string
}

// NewMiddleware creates a new OIDC middleware
func NewMiddleware(config *Config, serviceName string) (*Middleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	// Create OIDC provider
	provider, err := oidc.NewProvider(context.Background(), config.IssuerURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create OIDC provider: %v", err)
	}

	// Create OAuth2 config
	oauth2Config := &oauth2.Config{
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		RedirectURL:  config.RedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       config.Scopes,
	}

	// Create token verifier
	verifier := provider.Verifier(&oidc.Config{
		SkipClientIDCheck:    true,
		SkipIssuerCheck:      config.SkipIssuerCheck,
		SkipExpiryCheck:      config.SkipExpiryCheck,
		InsecureSkipVerify:   false,
		SupportedSigningAlgs: []string{oidc.RS256, oidc.ES256},
	})

	return &Middleware{
		provider:    provider,
		verifier:    verifier,
		config:      oauth2Config,
		metrics:     common.NewAuthMetricsCollector(),
		serviceName: serviceName,
	}, nil
}

// Middleware returns a middleware function that validates OIDC tokens
func (m *Middleware) Middleware(next http.Handler) http.Handler {
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
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "missing_token")
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		// Verify token
		token, err := m.verifier.Verify(r.Context(), tokenString)
		if err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "invalid_token")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Parse claims
		var claims struct {
			Subject string   `json:"sub"`
			Email   string   `json:"email"`
			Name    string   `json:"name"`
			Groups  []string `json:"groups"`
			Roles   []string `json:"roles"`
		}

		if err := token.Claims(&claims); err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "invalid_claims")
			http.Error(w, "Invalid token claims", http.StatusUnauthorized)
			return
		}

		// Add authentication info to context
		ctx := r.Context()
		ctx = common.WithAuthMethod(ctx, common.AuthMethodOIDC)
		ctx = common.WithServiceID(ctx, claims.Subject)
		ctx = common.WithRoles(ctx, claims.Roles)
		r = r.WithContext(ctx)

		// Record successful authentication
		m.metrics.RecordAuthRequest(m.serviceName, string(common.AuthMethodOIDC), "success", time.Since(start).Seconds())

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
		"/auth/callback", // OIDC callback endpoint
	}

	for _, skipPath := range skipPaths {
		if path == skipPath {
			return true
		}
	}
	return false
}

// ExtractToken extracts the token from the Authorization header
func (m *Middleware) ExtractToken(r *http.Request) (string, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", fmt.Errorf("authorization header required")
	}

	// Check for Bearer token
	if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
		return authHeader[7:], nil
	}

	return "", fmt.Errorf("invalid authorization header format")
}

// RequireRole creates a middleware that checks for required roles
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get authentication method
			method, err := common.GetAuthMethodFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "missing_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify OIDC authentication
			if method != common.AuthMethodOIDC {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "invalid_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get roles from context
			tokenRoles, err := common.GetRolesFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "missing_roles")
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
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodOIDC), "insufficient_roles")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// GetAuthURL returns the authorization URL for the OAuth2 flow
func (m *Middleware) GetAuthURL(state string) string {
	return m.config.AuthCodeURL(state)
}

// ExchangeCode exchanges the authorization code for tokens
func (m *Middleware) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	return m.config.Exchange(ctx, code)
}

// RefreshToken refreshes the access token
func (m *Middleware) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}
	return m.config.TokenSource(ctx, token).Token()
} 