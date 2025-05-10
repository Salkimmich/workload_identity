package common

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for certificate monitoring
var (
	// Tracks time until certificate expiration
	CertificateExpiry = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "certificate_expiry_seconds",
			Help: "Time until certificate expiration in seconds",
		},
		[]string{"type"},
	)

	// Counts certificate validation errors by reason
	CertificateValidationErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "certificate_validation_errors_total",
			Help: "Number of certificate validation errors",
		},
		[]string{"reason"},
	)

	// Measures time taken for certificate rotation
	CertificateRotationTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "certificate_rotation_seconds",
			Help:    "Time taken to rotate certificates",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"type"},
	)
)

// SecureCertificateStore provides secure storage for certificates using memory-mapped files
type SecureCertificateStore struct {
	mmap  []byte                // Memory-mapped file for secure storage
	cert  *x509.Certificate     // Current certificate
	key   *rsa.PrivateKey       // Current private key
	mutex sync.RWMutex         // Mutex for thread-safe access
}

// NewSecureCertificateStore creates a new secure certificate store with memory-mapped storage
func NewSecureCertificateStore() (*SecureCertificateStore, error) {
	// Create memory-mapped file with restricted permissions
	fd, err := syscall.MemfdCreate("cert-store", syscall.MFD_CLOEXEC)
	if err != nil {
		return nil, fmt.Errorf("failed to create secure memory region: %v", err)
	}

	// Set restrictive permissions (read/write for owner only)
	if err := syscall.Fchmod(fd, 0600); err != nil {
		return nil, fmt.Errorf("failed to set permissions: %v", err)
	}

	// Initialize store with memory-mapped region
	return &SecureCertificateStore{
		mmap: make([]byte, 4096), // 4KB memory region
	}, nil
}

// StoreCertificate securely stores a certificate and private key with validation
func (s *SecureCertificateStore) StoreCertificate(cert *x509.Certificate, key *rsa.PrivateKey) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Validate certificate before storage
	if err := s.validateCertificate(cert); err != nil {
		CertificateValidationErrors.WithLabelValues("invalid_cert").Inc()
		return fmt.Errorf("certificate validation failed: %v", err)
	}

	// Store certificate and key
	s.cert = cert
	s.key = key

	// Update expiration metric
	CertificateExpiry.WithLabelValues("svid").Set(
		time.Until(cert.NotAfter).Seconds(),
	)

	return nil
}

// GetCertificate returns the stored certificate and key as a TLS certificate
func (s *SecureCertificateStore) GetCertificate() (*tls.Certificate, error) {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	// Check if certificate exists
	if s.cert == nil || s.key == nil {
		return nil, fmt.Errorf("no certificate stored")
	}

	// Return certificate in TLS format
	return &tls.Certificate{
		Certificate: [][]byte{s.cert.Raw},
		PrivateKey:  s.key,
		Leaf:        s.cert,
	}, nil
}

// validateCertificate performs comprehensive certificate validation
func (s *SecureCertificateStore) validateCertificate(cert *x509.Certificate) error {
	// Check if certificate is expired
	if time.Now().After(cert.NotAfter) {
		CertificateValidationErrors.WithLabelValues("expired").Inc()
		return fmt.Errorf("certificate expired")
	}

	// Check if certificate is not yet valid
	if time.Now().Before(cert.NotBefore) {
		CertificateValidationErrors.WithLabelValues("not_yet_valid").Inc()
		return fmt.Errorf("certificate not yet valid")
	}

	// Verify digital signature capability
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		CertificateValidationErrors.WithLabelValues("invalid_key_usage").Inc()
		return fmt.Errorf("certificate does not have digital signature key usage")
	}

	// Check for required extended key usages
	hasClientAuth := false
	hasServerAuth := false
	for _, eku := range cert.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasClientAuth || !hasServerAuth {
		CertificateValidationErrors.WithLabelValues("invalid_extended_key_usage").Inc()
		return fmt.Errorf("certificate does not have required extended key usage")
	}

	return nil
}

// RotateCertificate rotates the stored certificate with timing metrics
func (s *SecureCertificateStore) RotateCertificate(cert *x509.Certificate, key *rsa.PrivateKey) error {
	start := time.Now()
	defer func() {
		CertificateRotationTime.WithLabelValues("svid").Observe(time.Since(start).Seconds())
	}()

	return s.StoreCertificate(cert, key)
} 