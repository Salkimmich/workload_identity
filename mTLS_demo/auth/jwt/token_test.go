package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestKeys(t *testing.T) (*ecdsa.PrivateKey, *ecdsa.PublicKey) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("Failed to generate test key: %v", err)
	}
	return privateKey, &privateKey.PublicKey
}

func TestTokenManager_GenerateToken(t *testing.T) {
	privateKey, publicKey := setupTestKeys(t)
	tm, err := NewTokenManager(privateKey, publicKey)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		serviceID string
		roles     []string
		scope     string
		duration  time.Duration
		wantErr   bool
	}{
		{
			name:      "Valid token",
			serviceID: "test-service",
			roles:     []string{"admin"},
			scope:     "read:write",
			duration:  time.Hour,
			wantErr:   false,
		},
		{
			name:      "Empty service ID",
			serviceID: "",
			roles:     []string{"admin"},
			scope:     "read:write",
			duration:  time.Hour,
			wantErr:   true,
		},
		{
			name:      "Empty roles",
			serviceID: "test-service",
			roles:     []string{},
			scope:     "read:write",
			duration:  time.Hour,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token, err := tm.GenerateToken(tt.serviceID, tt.roles, tt.scope, tt.duration)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotEmpty(t, token)

			// Verify the generated token
			claims, err := tm.VerifyToken(token)
			assert.NoError(t, err)
			assert.Equal(t, tt.serviceID, claims.ServiceID)
			assert.Equal(t, tt.roles, claims.Roles)
			assert.Equal(t, tt.scope, claims.Scope)
		})
	}
}

func TestTokenManager_VerifyToken(t *testing.T) {
	privateKey, publicKey := setupTestKeys(t)
	tm, err := NewTokenManager(privateKey, publicKey)
	assert.NoError(t, err)

	// Generate a valid token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name      string
		token     string
		wantErr   bool
		wantValid bool
	}{
		{
			name:      "Valid token",
			token:     token,
			wantErr:   false,
			wantValid: true,
		},
		{
			name:      "Invalid token",
			token:     "invalid.token.here",
			wantErr:   true,
			wantValid: false,
		},
		{
			name:      "Empty token",
			token:     "",
			wantErr:   true,
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := tm.VerifyToken(tt.token)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.NotNil(t, claims)
			assert.Equal(t, tt.wantValid, claims != nil)
		})
	}
}

func TestTokenManager_ValidateClaims(t *testing.T) {
	privateKey, publicKey := setupTestKeys(t)
	tm, err := NewTokenManager(privateKey, publicKey)
	assert.NoError(t, err)

	// Generate tokens with different scenarios
	validToken, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	expiredToken, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", -time.Hour)
	assert.NoError(t, err)

	tests := []struct {
		name    string
		token   string
		wantErr bool
	}{
		{
			name:    "Valid claims",
			token:   validToken,
			wantErr: false,
		},
		{
			name:    "Expired token",
			token:   expiredToken,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			claims, err := tm.VerifyToken(tt.token)
			if err != nil {
				assert.True(t, tt.wantErr)
				return
			}

			err = tm.ValidateClaims(claims)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTokenManager_RefreshToken(t *testing.T) {
	privateKey, publicKey := setupTestKeys(t)
	tm, err := NewTokenManager(privateKey, publicKey)
	assert.NoError(t, err)

	// Generate initial token
	initialToken, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	// Test refresh
	newToken, err := tm.RefreshToken(initialToken, time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, initialToken, newToken)

	// Verify new token
	claims, err := tm.VerifyToken(newToken)
	assert.NoError(t, err)
	assert.Equal(t, "test-service", claims.ServiceID)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, "read:write", claims.Scope)

	// Test refresh with invalid token
	_, err = tm.RefreshToken("invalid.token.here", time.Hour)
	assert.Error(t, err)
}

func TestTokenManager_GetTokenMetadata(t *testing.T) {
	privateKey, publicKey := setupTestKeys(t)
	tm, err := NewTokenManager(privateKey, publicKey)
	assert.NoError(t, err)

	// Generate token
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)

	// Get metadata
	metadata, err := tm.GetTokenMetadata(token)
	assert.NoError(t, err)
	assert.NotNil(t, metadata)

	// Verify metadata fields
	assert.Equal(t, "test-service", metadata["service_id"])
	assert.Contains(t, metadata["roles"], "admin")
	assert.Equal(t, "read:write", metadata["scope"])
} 