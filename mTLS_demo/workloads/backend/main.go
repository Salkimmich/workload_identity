package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"mTLS_demo/workloads/common"
)

// Response represents the structure of API responses
type Response struct {
	Message   string    `json:"message"`    // Response message
	RequestID string    `json:"request_id"` // Unique request identifier
	Timestamp time.Time `json:"timestamp"`  // Response timestamp
}

// Application constants
const (
	serverPort        = ":8443"              // HTTPS server port
	metricsPort       = ":8080"              // Metrics server port
	certCheckInterval = 30 * time.Second     // Certificate check interval
)

// BackendServer represents the backend service
type BackendServer struct {
	server         *http.Server              // Main HTTPS server
	metricsServer  *http.Server              // Metrics server
	certStore      *common.SecureCertificateStore // Certificate store
	metrics        *common.MetricsCollector  // Metrics collector
	logger         *zap.Logger              // Structured logger
	circuitBreaker *common.CircuitBreaker    // Circuit breaker for fault tolerance
}

// NewBackendServer creates a new backend server instance
func NewBackendServer() (*BackendServer, error) {
	// Initialize structured logger
	logger, err := zap.NewProduction()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %v", err)
	}

	// Initialize secure certificate store
	certStore, err := common.NewSecureCertificateStore()
	if err != nil {
		return nil, fmt.Errorf("failed to create certificate store: %v", err)
	}

	// Initialize metrics collector
	metrics := common.NewMetricsCollector()

	// Initialize circuit breaker with 5 failure threshold and 30s timeout
	circuitBreaker := common.NewCircuitBreaker("backend", 5, 30*time.Second)

	return &BackendServer{
		certStore:      certStore,
		metrics:        metrics,
		logger:         logger,
		circuitBreaker: circuitBreaker,
	}, nil
}

// Start initializes and starts the backend server
func (s *BackendServer) Start() error {
	// Load initial certificate
	if err := s.loadCertificate(); err != nil {
		return fmt.Errorf("failed to load initial certificate: %v", err)
	}

	// Create TLS configuration
	tlsConfig, err := s.createTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %v", err)
	}

	// Initialize main HTTPS server
	s.server = &http.Server{
		Addr:      serverPort,
		TLSConfig: tlsConfig,
		Handler:   s.createRouter(),
	}

	// Initialize metrics server
	s.metricsServer = &http.Server{
		Addr:    metricsPort,
		Handler: s.createMetricsRouter(),
	}

	// Start certificate reload goroutine
	go s.startCertificateReload()

	// Start metrics server
	go func() {
		s.logger.Info("Starting metrics server", zap.String("port", metricsPort))
		if err := s.metricsServer.ListenAndServe(); err != http.ErrServerClosed {
			s.logger.Fatal("Metrics server failed", zap.Error(err))
		}
	}()

	// Start main server
	s.logger.Info("Starting backend server", zap.String("port", serverPort))
	return s.server.ListenAndServeTLS("", "")
}

// createRouter sets up the HTTP router with endpoints
func (s *BackendServer) createRouter() http.Handler {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/hello", s.handleHello)
	mux.HandleFunc("/health", s.handleHealth)

	return mux
}

// createMetricsRouter sets up the metrics endpoint
func (s *BackendServer) createMetricsRouter() http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	return mux
}

// handleHello processes hello endpoint requests
func (s *BackendServer) handleHello(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		s.metrics.RecordHTTPRequest("/hello", r.Method, http.StatusOK, time.Since(start))
	}()

	// Create response
	response := Response{
		Message:   "Hello from backend!",
		RequestID: uuid.New().String(),
		Timestamp: time.Now(),
	}

	// Send response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHealth processes health check requests
func (s *BackendServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	defer func() {
		s.metrics.RecordHTTPRequest("/health", r.Method, http.StatusOK, time.Since(start))
	}()

	// Check certificate validity
	cert, err := s.certStore.GetCertificate()
	if err != nil {
		s.logger.Error("Health check failed: certificate error", zap.Error(err))
		http.Error(w, "Certificate error", http.StatusInternalServerError)
		return
	}

	// Check circuit breaker state
	if s.circuitBreaker.GetState() == common.StateOpen {
		s.logger.Warn("Health check: circuit breaker is open")
		http.Error(w, "Circuit breaker open", http.StatusServiceUnavailable)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "Backend service is healthy")
}

// loadCertificate loads and validates the certificate
func (s *BackendServer) loadCertificate() error {
	// Load certificate and key from files
	cert, err := tls.LoadX509KeyPair("/tmp/svid.pem", "/tmp/key.pem")
	if err != nil {
		return fmt.Errorf("failed to load certificate: %v", err)
	}

	// Parse certificate
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to parse certificate: %v", err)
	}

	// Store certificate in secure store
	return s.certStore.StoreCertificate(x509Cert, cert.PrivateKey.(*rsa.PrivateKey))
}

// createTLSConfig creates TLS configuration with certificate and trust bundle
func (s *BackendServer) createTLSConfig() (*tls.Config, error) {
	// Get certificate from store
	cert, err := s.certStore.GetCertificate()
	if err != nil {
		return nil, err
	}

	// Load trust bundle
	rootCAs := x509.NewCertPool()
	caCert, err := ioutil.ReadFile("/tmp/bundle.pem")
	if err != nil {
		return nil, fmt.Errorf("failed to load CA certificate: %v", err)
	}
	if !rootCAs.AppendCertsFromPEM(caCert) {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	// Create TLS config with secure defaults
	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		RootCAs:     rootCAs,
		MinVersion:  tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		},
	}, nil
}

// startCertificateReload starts the certificate reload goroutine
func (s *BackendServer) startCertificateReload() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP)

	for range sigChan {
		s.logger.Info("Received SIGHUP, reloading certificates")
		if err := s.loadCertificate(); err != nil {
			s.logger.Error("Failed to reload certificates", zap.Error(err))
			continue
		}

		// Update TLS config with new certificate
		tlsConfig, err := s.createTLSConfig()
		if err != nil {
			s.logger.Error("Failed to update TLS config", zap.Error(err))
			continue
		}
		s.server.TLSConfig = tlsConfig
		s.logger.Info("Certificates reloaded successfully")
	}
}

// main is the entry point of the application
func main() {
	// Create server instance
	server, err := NewBackendServer()
	if err != nil {
		log.Fatalf("Failed to create server: %v", err)
	}

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		server.logger.Info("Shutting down server...")

		// Create shutdown context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		// Shutdown servers
		if err := server.server.Shutdown(ctx); err != nil {
			server.logger.Error("Error during server shutdown", zap.Error(err))
		}

		if err := server.metricsServer.Shutdown(ctx); err != nil {
			server.logger.Error("Error during metrics server shutdown", zap.Error(err))
		}
	}()

	// Start server
	if err := server.Start(); err != nil {
		server.logger.Fatal("Server failed", zap.Error(err))
	}
} 