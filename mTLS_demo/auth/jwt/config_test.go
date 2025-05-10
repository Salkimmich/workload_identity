package jwt

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func setupTestKeys(t *testing.T) (string, string) {
	// Generate private key
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	assert.NoError(t, err)

	// Encode private key
	privateKeyBytes, err := x509.MarshalECPrivateKey(privateKey)
	assert.NoError(t, err)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "EC PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	// Encode public key
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	assert.NoError(t, err)
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	// Write keys to temporary files
	privateKeyFile, err := os.CreateTemp("", "private-*.pem")
	assert.NoError(t, err)
	_, err = privateKeyFile.Write(privateKeyPEM)
	assert.NoError(t, err)
	privateKeyFile.Close()

	publicKeyFile, err := os.CreateTemp("", "public-*.pem")
	assert.NoError(t, err)
	_, err = publicKeyFile.Write(publicKeyPEM)
	assert.NoError(t, err)
	publicKeyFile.Close()

	return privateKeyFile.Name(), publicKeyFile.Name()
}

func TestConfig_DefaultConfig(t *testing.T) {
	config := DefaultConfig()
	assert.NotNil(t, config)
	assert.Equal(t, 1*time.Hour, config.TokenDuration)
	assert.Equal(t, 24*time.Hour, config.RefreshDuration)
	assert.Equal(t, 5, config.MaxRefreshAttempts)
	assert.Equal(t, 100, config.RateLimitRequests)
	assert.Equal(t, 1*time.Minute, config.RateLimitWindow)
	assert.True(t, config.RequireHTTPS)
	assert.Equal(t, "mTLS_demo", config.Issuer)
	assert.True(t, config.RequireServiceID)
	assert.True(t, config.RequireRoles)
	assert.Equal(t, 1, config.MinimumRoleLength)
}

func TestConfig_LoadKeys(t *testing.T) {
	privateKeyPath, publicKeyPath := setupTestKeys(t)
	defer os.Remove(privateKeyPath)
	defer os.Remove(publicKeyPath)

	config := DefaultConfig()
	config.PrivateKeyPath = privateKeyPath
	config.PublicKeyPath = publicKeyPath

	err := config.LoadKeys()
	assert.NoError(t, err)
	assert.NotNil(t, config.PrivateKey)
	assert.NotNil(t, config.PublicKey)
}

func TestConfig_Validate(t *testing.T) {
	privateKeyPath, publicKeyPath := setupTestKeys(t)
	defer os.Remove(privateKeyPath)
	defer os.Remove(publicKeyPath)

	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid config",
			config: func() *Config {
				config := DefaultConfig()
				config.PrivateKeyPath = privateKeyPath
				config.PublicKeyPath = publicKeyPath
				config.LoadKeys()
				return config
			}(),
			wantErr: false,
		},
		{
			name: "Invalid token duration",
			config: func() *Config {
				config := DefaultConfig()
				config.TokenDuration = -1
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Invalid refresh duration",
			config: func() *Config {
				config := DefaultConfig()
				config.RefreshDuration = -1
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Invalid max refresh attempts",
			config: func() *Config {
				config := DefaultConfig()
				config.MaxRefreshAttempts = -1
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Invalid rate limit requests",
			config: func() *Config {
				config := DefaultConfig()
				config.RateLimitRequests = -1
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Invalid rate limit window",
			config: func() *Config {
				config := DefaultConfig()
				config.RateLimitWindow = -1
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Missing issuer with required service ID",
			config: func() *Config {
				config := DefaultConfig()
				config.Issuer = ""
				return config
			}(),
			wantErr: true,
		},
		{
			name: "Invalid minimum role length",
			config: func() *Config {
				config := DefaultConfig()
				config.MinimumRoleLength = -1
				return config
			}(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNewTokenManagerFromConfig(t *testing.T) {
	privateKeyPath, publicKeyPath := setupTestKeys(t)
	defer os.Remove(privateKeyPath)
	defer os.Remove(publicKeyPath)

	config := DefaultConfig()
	config.PrivateKeyPath = privateKeyPath
	config.PublicKeyPath = publicKeyPath

	err := config.LoadKeys()
	assert.NoError(t, err)

	tm, err := NewTokenManagerFromConfig(config)
	assert.NoError(t, err)
	assert.NotNil(t, tm)

	// Test token generation with the manager
	token, err := tm.GenerateToken("test-service", []string{"admin"}, "read:write", time.Hour)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Verify the token
	claims, err := tm.VerifyToken(token)
	assert.NoError(t, err)
	assert.Equal(t, "test-service", claims.ServiceID)
	assert.Equal(t, []string{"admin"}, claims.Roles)
	assert.Equal(t, "read:write", claims.Scope)
} 