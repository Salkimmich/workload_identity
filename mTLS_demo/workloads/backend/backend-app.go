// backend/main.go
// Example Go application demonstrating SPIFFE mTLS server implementation
package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
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

// Basic handler function
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "healthy", "service": "backend"}`))
}

// Echo handler
func echoHandler(w http.ResponseWriter, r *http.Request) {
	// Log details about the client certificate
	if r.TLS != nil && len(r.TLS.PeerCertificates) > 0 {
		clientCert := r.TLS.PeerCertificates[0]
		log.Printf("Client authenticated with cert: Subject=%s, Issuer=%s", 
			clientCert.Subject.CommonName, 
			clientCert.Issuer.CommonName)
		
		// Look for SPIFFE ID in URI SAN
		for _, uri := range clientCert.URIs {
			if uri.Scheme == "spiffe" {
				log.Printf("Client SPIFFE ID: %s", uri.String())
			}
		}
	}

	// Echo back the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write(body)
	
	log.Printf("Processed request from %s", r.RemoteAddr)
}

func main() {
	// Wait for SVID certificate to be available
	for {
		if _, err := os.Stat(svidPath); err == nil {
			break
		}
		log.Println("Waiting for SVID certificate...")
		time.Sleep(1 * time.Second)
	}

	// Load CA bundle for client verification
	bundleData, err := os.ReadFile(bundlePath)
	if err != nil {
		log.Fatalf("Failed to read CA bundle: %v", err)
	}
	
	bundlePool := x509.NewCertPool()
	if !bundlePool.AppendCertsFromPEM(bundleData) {
		log.Fatalf("Failed to parse CA bundle")
	}

	// Create certificate reloader to handle SVID rotation
	reloader, err := newCertReloader(svidPath, keyPath)
	if err != nil {
		log.Fatalf("Failed to load certificates: %v", err)
	}
	
	// Watch for certificate changes every 10 seconds
	reloader.watchForChanges(10 * time.Second)

	// Configure the TLS server
	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  bundlePool,
		// Get the current certificate during TLS handshake
		GetCertificate: func(*tls.ClientHelloInfo) (*tls.Certificate, error) {
			return reloader.getCert(), nil
		},
		MinVersion: tls.VersionTLS12,
	}

	// Set up routes
	http.HandleFunc("/health", healthHandler)
	http.HandleFunc("/echo", echoHandler)
	
	server := &http.Server{
		Addr:      ":8443",
		TLSConfig: tlsConfig,
	}

	// Start the server with empty cert/key arguments since we're using GetCertificate callback
	log.Printf("Backend server listening on :8443 with mTLS enabled")
	if err := server.ListenAndServeTLS("", ""); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
