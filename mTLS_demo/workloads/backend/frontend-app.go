// frontend/main.go
// Example Go application demonstrating SPIFFE mTLS client implementation
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

// SVID file paths provided by SPIFFE Helper
const (
	svidPath   = "/tmp/svid.pem"
	keyPath    = "/tmp/key.pem"
	bundlePath = "/tmp/bundle.pem"
)

// We'll use this to reload certs when they change
type certReloader struct {
	sync.Mutex
	cert     *tls.Certificate
	certPath string
	keyPath  string
}

// Initialize a certificate reloader
func newCertReloader(certPath, keyPath string) (*certReloader, error) {
	cr := &certReloader{
		certPath: certPath,
		keyPath:  keyPath,
	}
	if err := cr.reload(); err != nil {
		return nil, err
	}
	return cr, nil
}

// Reload certificates from disk
func (cr *certReloader) reload() error {
	cert, err := tls.LoadX509KeyPair(cr.certPath, cr.keyPath)
	if err != nil {
		return err
	}
	cr.Lock()
	defer cr.Unlock()
	cr.cert = &cert
	return nil
}

// Get the current certificate
func (cr *certReloader) getCert() *tls.Certificate {
	cr.Lock()
	defer cr.Unlock()
	return cr.cert
}

// Watch for certificate changes
func (cr *certReloader) watchForChanges(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := cr.reload(); err != nil {
				log.Printf("Error reloading certificates: %v", err)
			} else {
				log.Printf("Certificates reloaded successfully")
			}
		}
	}()
}

// Create an mTLS HTTP client using SPIFFE SVIDs
func createMTLSClient() (*http.Client, error) {
	// Wait for SVID certificate to be available
	for {
		if _, err := os.Stat(svidPath); err == nil {
			break
		}
		log.Println("Waiting for SVID certificate...")
		time.Sleep(1 * time.Second)
	}

	// Load CA bundle for server verification
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA bundle: %v", err)
	}
	
	bundlePool := x509.NewCertPool()
	if !bundlePool.AppendCertsFromPEM(bundleData) {
		return nil, fmt.Errorf("failed to parse CA bundle")
	}

	// Create certificate reloader
	reloader, err := newCertReloader(svidPath, keyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificates: %v", err)
	}
	
	// Watch for certificate changes
	reloader.watchForChanges(10 * time.Second)

	// Configure TLS client
	tlsConfig := &tls.Config{
		RootCAs: bundlePool,
		GetClientCertificate: func(*tls.CertificateRequestInfo) (*tls.Certificate, error) {
			return reloader.getCert(), nil
		},
		MinVersion: tls.VersionTLS12,
	}

	// Create HTTP client with custom Transport using our TLS config
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
		Timeout: 10 * time.Second,
	}

	return client, nil
}

// Call backend service using mTLS
func callBackend(client *http.Client, message string) (string, error) {
	// Create request with message body
	req, err := http.NewRequest("POST", "https://backend:8443/echo", strings.NewReader(message))
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}
	
	// Send the request
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()
	
	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response: %v", err)
	}
	
	return string(body), nil
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "frontend"}`))
}

// Handler to proxy requests to backend service
func backendProxyHandler(client *http.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Read request body
		reqBody, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request", http.StatusBadRequest)
			return
		}
		
		// Call backend
		response, err := callBackend(client, string(reqBody))
		if err != nil {
			log.Printf("Backend call failed: %v", err)
			http.Error(w, "Error calling backend service", http.StatusServiceUnavailable)
			return
		}
		
		// Return response
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
		
		log.Printf("Successfully proxied request to backend")
	}
}

func main() {
	// Create mTLS client for backend calls
	client, err := createMTLSClient()
	if err != nil {
		log.Fatalf("Failed to create mTLS client: %v", err)
	}
	
	// Set up routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/proxy", backendProxyHandler(client))
	
	// Start HTTP server
	log.Printf("Frontend server listening on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
