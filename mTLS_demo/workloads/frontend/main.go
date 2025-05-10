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
	backendURL       = "https://backend:8443/hello" // Backend service URL
	serverPort       = ":8080"                      // Frontend server port
	certCheckInterval = 30 * time.Second            // Certificate check interval
)

// FrontendServer represents the frontend service
type FrontendServer struct {
	server         *http.Server              // Main HTTP server
	certStore      *common.SecureCertificateStore // Certificate store
	metrics        *common.MetricsCollector  // Metrics collector
	logger         *zap.Logger              // Structured logger
	circuitBreaker *common.CircuitBreaker    // Circuit breaker for fault tolerance
	retryPolicy    *common.RetryPolicy      // Retry policy for failed requests
	client         *http.Client             // HTTP client for backend requests
}

// NewFrontendServer creates a new frontend server instance
func NewFrontendServer() (*FrontendServer, error) {
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
	circuitBreaker := common.NewCircuitBreaker("frontend", 5, 30*time.Second)

	// Initialize retry policy with exponential backoff
	retryPolicy := common.NewRetryPolicy("frontend", 3, 100*time.Millisecond, 1*time.Second, 0.1)

	return &FrontendServer{
		certStore:      certStore,
		metrics:        metrics,
		logger:         logger,
		circuitBreaker: circuitBreaker,
		retryPolicy:    retryPolicy,
	}, nil
}

// Start initializes and starts the frontend server
func (s *FrontendServer) Start() error {
	// Load initial certificate
	if err := s.loadCertificate(); err != nil {
		return fmt.Errorf("failed to load initial certificate: %v", err)
	}

	// Create TLS configuration
	tlsConfig, err := s.createTLSConfig()
	if err != nil {
		return fmt.Errorf("failed to create TLS config: %v", err)
	}

	// Initialize HTTP client with TLS config
	s.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}

	// Initialize server
	s.server = &http.Server{
		Addr:    serverPort,
		Handler: s.createRouter(),
	}

	// Start certificate reload goroutine
	go s.startCertificateReload()

	// Start request loop
	go s.startRequestLoop()

	// Start server
	s.logger.Info("Starting frontend server", zap.String("port", serverPort))
	return s.server.ListenAndServe()
}

// createRouter sets up the HTTP router with endpoints
func (s *FrontendServer) createRouter() http.Handler {
	mux := http.NewServeMux()

	// Register endpoints
	mux.HandleFunc("/health", s.handleHealth)
	mux.Handle("/metrics", promhttp.Handler())

	return mux
}

// handleHealth processes health check requests
func (s *FrontendServer) handleHealth(w http.ResponseWriter, r *http.Request) {
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
	fmt.Fprint(w, "Frontend service is healthy")
}

// loadCertificate loads and validates the certificate
func (s *FrontendServer) loadCertificate() error {
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
func (s *FrontendServer) createTLSConfig() (*tls.Config, error) {
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
func (s *FrontendServer) startCertificateReload() {
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

		// Update client transport with new TLS config
		s.client.Transport.(*http.Transport).TLSClientConfig = tlsConfig
		s.logger.Info("Certificates reloaded successfully")
	}
}

// startRequestLoop starts the periodic request loop to backend
func (s *FrontendServer) startRequestLoop() {
	for {
		if err := s.makeRequest(); err != nil {
			s.logger.Error("Request failed", zap.Error(err))
			time.Sleep(5 * time.Second)
			continue
		}
		time.Sleep(10 * time.Second)
	}
}

// makeRequest makes a request to the backend service
func (s *FrontendServer) makeRequest() error {
	start := time.Now()
	defer func() {
		s.metrics.RecordHTTPRequest("/hello", "GET", http.StatusOK, time.Since(start))
	}()

	// Create request
	req, err := http.NewRequest("GET", backendURL, nil)
	if err != nil {
		s.metrics.RecordHTTPError("/hello", "GET", "request_creation")
		return fmt.Errorf("failed to create request: %v", err)
	}

	// Add request ID header
	requestID := uuid.New().String()
	req.Header.Set("X-Request-ID", requestID)

	// Execute request with circuit breaker and retry
	result, err := s.circuitBreaker.Execute(func() (interface{}, error) {
		return s.retryPolicy.Do(context.Background(), func() (interface{}, error) {
			return s.client.Do(req)
		})
	})

	if err != nil {
		s.metrics.RecordHTTPError("/hello", "GET", "request_failed")
		return fmt.Errorf("request failed: %v", err)
	}

	// Process response
	resp := result.(*http.Response)
	defer resp.Body.Close()

	// Check response status
	if resp.StatusCode != http.StatusOK {
		s.metrics.RecordHTTPError("/hello", "GET", fmt.Sprintf("status_%d", resp.StatusCode))
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	// Parse response body
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		s.metrics.RecordHTTPError("/hello", "GET", "decode_failed")
		return fmt.Errorf("failed to decode response: %v", err)
	}

	// Log successful response
	s.logger.Info("Received response",
		zap.String("request_id", response.RequestID),
		zap.String("message", response.Message),
		zap.Time("timestamp", response.Timestamp),
	)

	return nil
}

// main is the entry point of the application
func main() {
	// Create server instance
	server, err := NewFrontendServer()
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

		// Shutdown server
		if err := server.server.Shutdown(ctx); err != nil {
			server.logger.Error("Error during server shutdown", zap.Error(err))
		}
	}()

	// Start server
	if err := server.Start(); err != nil {
		server.logger.Fatal("Server failed", zap.Error(err))
	}
} 