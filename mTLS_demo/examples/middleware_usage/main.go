package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"mTLS_demo/auth/apikey"
	"mTLS_demo/auth/common"
	"mTLS_demo/auth/oidc"
	"mTLS_demo/auth/ratelimit"
)

func main() {
	// Create API key store and add some test keys
	apiKeyStore := apikey.NewInMemoryStore()
	testKey := &apikey.Key{
		ID:        "test-service",
		Hash:      "test-hash", // In production, use proper hashing
		Roles:     []string{"service"},
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	if err := apiKeyStore.AddKey(testKey); err != nil {
		log.Fatalf("Failed to add test key: %v", err)
	}

	// Create OIDC middleware with Keycloak configuration
	oidcConfig := &oidc.Config{
		IssuerURL:      "http://localhost:8081/realms/demo",
		ClientID:       "demo-client",
		ClientSecret:   "demo-secret",
		RedirectURL:    "http://localhost:8080/auth/callback",
		Scopes:         []string{"openid", "profile", "email"},
		SkipIssuerCheck: true, // Only for testing
		SkipExpiryCheck: true, // Only for testing
	}
	oidcMiddleware, err := oidc.NewMiddleware(oidcConfig, "example-service")
	if err != nil {
		log.Fatalf("Failed to create OIDC middleware: %v", err)
	}

	// Create API key middleware
	apiKeyConfig := &apikey.Config{
		Store: apiKeyStore,
	}
	apiKeyMiddleware, err := apikey.NewMiddleware(apiKeyConfig, "example-service")
	if err != nil {
		log.Fatalf("Failed to create API key middleware: %v", err)
	}

	// Create rate limiting middleware
	rateLimitConfig := &ratelimit.Config{
		RequestsPerSecond: 10,
		Burst:            5,
		WaitOnLimit:      true,
	}
	rateLimitMiddleware, err := ratelimit.NewMiddleware(rateLimitConfig)
	if err != nil {
		log.Fatalf("Failed to create rate limit middleware: %v", err)
	}

	// Create router
	mux := http.NewServeMux()

	// Public endpoints (no auth required)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	// Service-to-service endpoints (API key auth required)
	mux.Handle("/api/service", apiKeyMiddleware.RequireRole("service")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			serviceID, _ := common.GetServiceIDFromContext(r.Context())
			w.Write([]byte("Hello service: " + serviceID))
		}),
	))

	// User endpoints (OIDC auth required)
	mux.Handle("/api/user", oidcMiddleware.RequireRole("user")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := common.GetServiceIDFromContext(r.Context())
			w.Write([]byte("Hello user: " + userID))
		}),
	))

	// Admin endpoints (OIDC auth with admin role required)
	mux.Handle("/api/admin", oidcMiddleware.RequireRole("admin")(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, _ := common.GetServiceIDFromContext(r.Context())
			w.Write([]byte("Hello admin: " + userID))
		}),
	))

	// Protected endpoints with rate limiting
	mux.Handle("/api/protected", rateLimitMiddleware.Middleware(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte("Rate limited endpoint"))
		}),
	))

	// OIDC callback endpoint
	mux.HandleFunc("/auth/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "Missing code", http.StatusBadRequest)
			return
		}

		// Exchange code for token
		token, err := oidcMiddleware.ExchangeCode(r.Context(), code)
		if err != nil {
			http.Error(w, "Failed to exchange code", http.StatusInternalServerError)
			return
		}

		// In a real application, you would:
		// 1. Store the token securely
		// 2. Set a session cookie
		// 3. Redirect to the appropriate page
		w.Write([]byte("Authentication successful"))
	})

	// Start server
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
} 