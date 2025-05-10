package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/ed25519"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// TokenManager handles JWT token operations
type TokenManager struct {
	signingKey    crypto.PrivateKey
	verifyingKey  crypto.PublicKey
	signingMethod jwt.SigningMethod
}

// TokenClaims represents the JWT claims
type TokenClaims struct {
	jwt.RegisteredClaims
	ServiceID string   `json:"service_id"`
	Roles     []string `json:"roles"`
	Scope     string   `json:"scope,omitempty"`
}

// NewTokenManager creates a new token manager
func NewTokenManager(signingKey crypto.PrivateKey, verifyingKey crypto.PublicKey) (*TokenManager, error) {
	var method jwt.SigningMethod

	switch signingKey.(type) {
	case *rsa.PrivateKey:
		method = jwt.SigningMethodRS256
	case *ecdsa.PrivateKey:
		method = jwt.SigningMethodES256
	case ed25519.PrivateKey:
		method = jwt.SigningMethodEdDSA
	default:
		return nil, fmt.Errorf("unsupported key type")
	}

	return &TokenManager{
		signingKey:    signingKey,
		verifyingKey:  verifyingKey,
		signingMethod: method,
	}, nil
}

// GenerateToken creates a new JWT token
func (tm *TokenManager) GenerateToken(serviceID string, roles []string, scope string, duration time.Duration) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(duration)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "mtls-demo",
			Subject:   serviceID,
		},
		ServiceID: serviceID,
		Roles:     roles,
		Scope:     scope,
	}

	token := jwt.NewWithClaims(tm.signingMethod, claims)
	return token.SignedString(tm.signingKey)
}

// VerifyToken verifies and parses a JWT token
func (tm *TokenManager) VerifyToken(tokenString string) (*TokenClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (interface{}, error) {
		if token.Method != tm.signingMethod {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return tm.verifyingKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(*TokenClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token claims")
}

// ValidateClaims performs additional validation on token claims
func (tm *TokenManager) ValidateClaims(claims *TokenClaims) error {
	// Check if token is expired
	if time.Now().After(claims.ExpiresAt.Time) {
		return fmt.Errorf("token has expired")
	}

	// Check if token is not yet valid
	if time.Now().Before(claims.NotBefore.Time) {
		return fmt.Errorf("token is not yet valid")
	}

	// Validate required claims
	if claims.ServiceID == "" {
		return fmt.Errorf("missing service ID")
	}

	if len(claims.Roles) == 0 {
		return fmt.Errorf("missing roles")
	}

	return nil
}

// GetTokenMetadata returns token metadata without verification
func (tm *TokenManager) GetTokenMetadata(tokenString string) (map[string]interface{}, error) {
	// Parse token without verification
	parser := jwt.Parser{SkipClaimsValidation: true}
	token, _, err := parser.ParseUnverified(tokenString, jwt.MapClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	// Extract claims
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// RefreshToken creates a new token with extended expiration
func (tm *TokenManager) RefreshToken(tokenString string, duration time.Duration) (string, error) {
	claims, err := tm.VerifyToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("failed to verify token: %v", err)
	}

	// Create new token with extended expiration
	return tm.GenerateToken(claims.ServiceID, claims.Roles, claims.Scope, duration)
} 