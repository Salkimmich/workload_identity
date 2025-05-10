package jwt

import (
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"time"
)

// Config holds the JWT configuration
type Config struct {
	// Token settings
	TokenDuration     time.Duration
	RefreshDuration   time.Duration
	MaxRefreshAttempts int

	// Key settings
	PrivateKeyPath string
	PublicKeyPath  string
	PrivateKey     *ecdsa.PrivateKey
	PublicKey      *ecdsa.PublicKey

	// Rate limiting
	RateLimitRequests int
	RateLimitWindow   time.Duration

	// Security settings
	RequireHTTPS      bool
	AllowedOrigins    []string
	AllowedAudiences  []string
	Issuer            string
	RequireServiceID  bool
	RequireRoles      bool
	MinimumRoleLength int
}

// DefaultConfig returns a default configuration
func DefaultConfig() *Config {
	return &Config{
		TokenDuration:     1 * time.Hour,
		RefreshDuration:   24 * time.Hour,
		MaxRefreshAttempts: 5,
		RateLimitRequests: 100,
		RateLimitWindow:   1 * time.Minute,
		RequireHTTPS:      true,
		Issuer:            "mTLS_demo",
		RequireServiceID:  true,
		RequireRoles:      true,
		MinimumRoleLength: 1,
	}
}

// LoadKeys loads the private and public keys from files
func (c *Config) LoadKeys() error {
	// Load private key
	privateKeyData, err := os.ReadFile(c.PrivateKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read private key: %v", err)
	}

	block, _ := pem.Decode(privateKeyData)
	if block == nil {
		return fmt.Errorf("failed to decode private key PEM")
	}

	privateKey, err := x509.ParseECPrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %v", err)
	}
	c.PrivateKey = privateKey

	// Load public key
	publicKeyData, err := os.ReadFile(c.PublicKeyPath)
	if err != nil {
		return fmt.Errorf("failed to read public key: %v", err)
	}

	block, _ = pem.Decode(publicKeyData)
	if block == nil {
		return fmt.Errorf("failed to decode public key PEM")
	}

	publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %v", err)
	}

	publicKey, ok := publicKeyInterface.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("public key is not ECDSA")
	}
	c.PublicKey = publicKey

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.TokenDuration <= 0 {
		return fmt.Errorf("token duration must be positive")
	}

	if c.RefreshDuration <= 0 {
		return fmt.Errorf("refresh duration must be positive")
	}

	if c.MaxRefreshAttempts <= 0 {
		return fmt.Errorf("max refresh attempts must be positive")
	}

	if c.RateLimitRequests <= 0 {
		return fmt.Errorf("rate limit requests must be positive")
	}

	if c.RateLimitWindow <= 0 {
		return fmt.Errorf("rate limit window must be positive")
	}

	if c.RequireServiceID && c.Issuer == "" {
		return fmt.Errorf("issuer is required when service ID is required")
	}

	if c.RequireRoles && c.MinimumRoleLength <= 0 {
		return fmt.Errorf("minimum role length must be positive when roles are required")
	}

	if c.PrivateKey == nil || c.PublicKey == nil {
		return fmt.Errorf("keys must be loaded before validation")
	}

	return nil
}

// NewTokenManagerFromConfig creates a new TokenManager from the configuration
func NewTokenManagerFromConfig(config *Config) (*TokenManager, error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %v", err)
	}

	return NewTokenManager(config.PrivateKey, config.PublicKey)
} 