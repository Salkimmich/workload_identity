package mtls

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Certificate metrics
	certExpirationTime = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "mtls_cert_expiration_time",
		Help: "Time until certificate expiration in seconds",
	})

	certRotationCount = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mtls_cert_rotations_total",
		Help: "Total number of certificate rotations",
	})

	certValidationErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "mtls_cert_validation_errors_total",
		Help: "Total number of certificate validation errors",
	})
)

// SecureCertificateStore manages TLS certificates with secure storage
type SecureCertificateStore struct {
	certPath     string
	keyPath      string
	trustPath    string
	cert         *tls.Certificate
	trustBundle  *x509.CertPool
	mutex        sync.RWMutex
	lastRotation time.Time
}

// NewSecureCertificateStore creates a new secure certificate store
func NewSecureCertificateStore(certPath, keyPath, trustPath string) *SecureCertificateStore {
	return &SecureCertificateStore{
		certPath:    certPath,
		keyPath:     keyPath,
		trustPath:   trustPath,
		trustBundle: x509.NewCertPool(),
	}
}

// StoreCertificate stores a new certificate and key
func (s *SecureCertificateStore) StoreCertificate(certPEM, keyPEM []byte) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Create memory-mapped files for sensitive data
	certFile, err := ioutil.TempFile("", "cert-*")
	if err != nil {
		return fmt.Errorf("failed to create temp cert file: %v", err)
	}
	defer os.Remove(certFile.Name())

	keyFile, err := ioutil.TempFile("", "key-*")
	if err != nil {
		return fmt.Errorf("failed to create temp key file: %v", err)
	}
	defer os.Remove(keyFile.Name())

	// Write certificate and key to temp files
	if err := ioutil.WriteFile(certFile.Name(), certPEM, 0600); err != nil {
		return fmt.Errorf("failed to write cert: %v", err)
	}
	if err := ioutil.WriteFile(keyFile.Name(), keyPEM, 0600); err != nil {
		return fmt.Errorf("failed to write key: %v", err)
	}

	// Load the certificate
	cert, err := tls.LoadX509KeyPair(certFile.Name(), keyFile.Name())
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	// Validate the certificate
	if err := s.validateCertificate(&cert); err != nil {
		certValidationErrors.Inc()
		return fmt.Errorf("certificate validation failed: %v", err)
	}

	s.cert = &cert
	s.lastRotation = time.Now()
	certRotationCount.Inc()

	// Update metrics
	if cert.Leaf != nil {
		certExpirationTime.Set(time.Until(cert.Leaf.NotAfter).Seconds())
	}

	return nil
}

// GetCertificate returns the current certificate
func (s *SecureCertificateStore) GetCertificate() *tls.Certificate {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.cert
}

// GetTrustBundle returns the current trust bundle
func (s *SecureCertificateStore) GetTrustBundle() *x509.CertPool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.trustBundle
}

// validateCertificate performs security checks on the certificate
func (s *SecureCertificateStore) validateCertificate(cert *tls.Certificate) error {
	if cert == nil || cert.Leaf == nil {
		return fmt.Errorf("invalid certificate")
	}

	// Check expiration
	if time.Now().After(cert.Leaf.NotAfter) {
		return fmt.Errorf("certificate has expired")
	}

	// Check if certificate is about to expire (80% of lifetime)
	lifetime := cert.Leaf.NotAfter.Sub(cert.Leaf.NotBefore)
	threshold := cert.Leaf.NotAfter.Add(-lifetime * 20 / 100)
	if time.Now().After(threshold) {
		return fmt.Errorf("certificate is approaching expiration")
	}

	// Verify key usage
	if cert.Leaf.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		return fmt.Errorf("certificate missing digital signature key usage")
	}

	return nil
}

// RotateCertificate rotates the certificate if needed
func (s *SecureCertificateStore) RotateCertificate() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cert == nil || s.cert.Leaf == nil {
		return fmt.Errorf("no certificate to rotate")
	}

	// Check if rotation is needed (80% of lifetime)
	lifetime := s.cert.Leaf.NotAfter.Sub(s.cert.Leaf.NotBefore)
	threshold := s.cert.Leaf.NotAfter.Add(-lifetime * 20 / 100)
	if time.Now().Before(threshold) {
		return nil // No rotation needed
	}

	// Read new certificate and key
	certPEM, err := ioutil.ReadFile(s.certPath)
	if err != nil {
		return fmt.Errorf("failed to read certificate: %v", err)
	}

	keyPEM, err := ioutil.ReadFile(s.keyPath)
	if err != nil {
		return fmt.Errorf("failed to read key: %v", err)
	}

	// Store new certificate
	return s.StoreCertificate(certPEM, keyPEM)
}

// LoadTrustBundle loads the trust bundle from file
func (s *SecureCertificateStore) LoadTrustBundle() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	trustPEM, err := ioutil.ReadFile(s.trustPath)
	if err != nil {
		return fmt.Errorf("failed to read trust bundle: %v", err)
	}

	if !s.trustBundle.AppendCertsFromPEM(trustPEM) {
		return fmt.Errorf("failed to parse trust bundle")
	}

	return nil
} 