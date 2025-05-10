package mtls

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"time"

	"mTLS_demo/auth/common"
)

// Middleware handles mTLS authentication
type Middleware struct {
	config     *Config
	metrics    common.AuthMetricsCollector
	serviceName string
}

// Config holds the mTLS configuration
type Config struct {
	TrustBundle []*x509.Certificate
	AllowedCNs  []string
}

// NewMiddleware creates a new mTLS middleware
func NewMiddleware(config *Config, serviceName string) (*Middleware, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	return &Middleware{
		config:     config,
		metrics:    common.NewAuthMetricsCollector(),
		serviceName: serviceName,
	}, nil
}

// Middleware returns a middleware function that validates mTLS
func (m *Middleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Skip validation for certain paths
		if m.shouldSkipValidation(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Check if TLS connection exists
		if r.TLS == nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "no_tls")
			http.Error(w, "TLS required", http.StatusUnauthorized)
			return
		}

		// Verify client certificate
		if len(r.TLS.PeerCertificates) == 0 {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "no_certificate")
			http.Error(w, "Client certificate required", http.StatusUnauthorized)
			return
		}

		// Get client certificate
		clientCert := r.TLS.PeerCertificates[0]

		// Verify certificate
		if err := m.VerifyCertificate(clientCert); err != nil {
			m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "invalid_certificate")
			http.Error(w, "Invalid certificate", http.StatusUnauthorized)
			return
		}

		// Add authentication info to context
		ctx := r.Context()
		ctx = common.WithAuthMethod(ctx, common.AuthMethodMTLS)
		ctx = common.WithServiceID(ctx, clientCert.Subject.CommonName)
		ctx = common.WithRoles(ctx, m.GetCertificateRoles(clientCert))
		r = r.WithContext(ctx)

		// Record successful authentication
		m.metrics.RecordAuthRequest(m.serviceName, string(common.AuthMethodMTLS), "success", time.Since(start).Seconds())

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

// VerifyCertificate verifies the client certificate
func (m *Middleware) VerifyCertificate(cert *x509.Certificate) error {
	// Check if CN is allowed
	cnAllowed := false
	for _, allowedCN := range m.config.AllowedCNs {
		if cert.Subject.CommonName == allowedCN {
			cnAllowed = true
			break
		}
	}
	if !cnAllowed {
		return fmt.Errorf("certificate CN not allowed")
	}

	// Verify certificate against trust bundle
	opts := x509.VerifyOptions{
		Roots:         x509.NewCertPool(),
		CurrentTime:   time.Now(),
		KeyUsages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		Intermediates: x509.NewCertPool(),
	}

	for _, root := range m.config.TrustBundle {
		opts.Roots.AddCert(root)
	}

	_, err := cert.Verify(opts)
	return err
}

// GetCertificateRoles extracts roles from the certificate
func (m *Middleware) GetCertificateRoles(cert *x509.Certificate) []string {
	// Extract roles from certificate extensions or SAN
	// This is a placeholder - implement according to your certificate structure
	return []string{"service"}
}

// RequireRole creates a middleware that checks for required roles
func (m *Middleware) RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get authentication method
			method, err := common.GetAuthMethodFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "missing_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Verify mTLS authentication
			if method != common.AuthMethodMTLS {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "invalid_auth_method")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Get roles from context
			certRoles, err := common.GetRolesFromContext(r.Context())
			if err != nil {
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "missing_roles")
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			// Check if any required role is present
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
				m.metrics.RecordAuthError(m.serviceName, string(common.AuthMethodMTLS), "insufficient_roles")
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
} 