# Developer Guide

This document provides comprehensive guidance for developers working with the workload identity system.

## Table of Contents
1. [Getting Started](#getting-started)
2. [Development Environment](#development-environment)
3. [Core Concepts](#core-concepts)
4. [Implementation Guide](#implementation-guide)
5. [Testing](#testing)
6. [Debugging](#debugging)
7. [Performance Optimization](#performance-optimization)
8. [Security Best Practices](#security-best-practices)
9. [Contributing](#contributing)

## Getting Started

### 1. Prerequisites
```yaml
# Required Tools and Versions
prerequisites:
  tools:
    - name: "go"
      version: ">=1.19"
    - name: "docker"
      version: ">=20.10"
    - name: "kubectl"
      version: ">=1.24"
    - name: "make"
      version: ">=4.0"
  services:
    - name: "kubernetes"
      version: ">=1.24"
    - name: "postgresql"
      version: ">=14.0"
```

### 2. Quick Start
```bash
# Clone the repository
git clone https://github.com/your-org/workload-identity.git
cd workload-identity

# Set up development environment
make setup-dev

# Start local development cluster
make start-local

# Run tests
make test
```

## Development Environment

### 1. Local Development
```yaml
# Development Environment Configuration
development:
  local:
    services:
      - name: "identity-provider"
        port: 8080
        env:
          - "DB_HOST=localhost"
          - "DB_PORT=5432"
      - name: "certificate-authority"
        port: 8081
        env:
          - "CA_KEY_PATH=./certs"
```

### 2. IDE Setup
```yaml
# IDE Configuration
ide:
  vscode:
    extensions:
      - "golang.go"
      - "ms-kubernetes-tools.vscode-kubernetes-tools"
      - "redhat.vscode-yaml"
    settings:
      "go.formatTool": "gofmt"
      "go.lintTool": "golangci-lint"
```

## Core Concepts

### 1. Identity Management
```go
// Example Identity Structure
type Identity struct {
    ID        string            `json:"id"`
    Name      string            `json:"name"`
    Type      string            `json:"type"`
    Metadata  map[string]string `json:"metadata"`
    CreatedAt time.Time         `json:"created_at"`
}

// Example Identity Creation
func CreateIdentity(ctx context.Context, identity *Identity) error {
    // Validate identity
    if err := validateIdentity(identity); err != nil {
        return fmt.Errorf("invalid identity: %w", err)
    }
    
    // Store identity
    if err := storeIdentity(ctx, identity); err != nil {
        return fmt.Errorf("failed to store identity: %w", err)
    }
    
    return nil
}
```

### 2. Certificate Management
```go
// Example Certificate Structure
type Certificate struct {
    SerialNumber string    `json:"serial_number"`
    IdentityID   string    `json:"identity_id"`
    ValidFrom    time.Time `json:"valid_from"`
    ValidTo      time.Time `json:"valid_to"`
    Certificate  string    `json:"certificate"`
    PrivateKey   string    `json:"private_key"`
}

// Example Certificate Issuance
func IssueCertificate(ctx context.Context, req *CertificateRequest) (*Certificate, error) {
    // Generate key pair
    keyPair, err := generateKeyPair(req.KeyType, req.KeySize)
    if err != nil {
        return nil, fmt.Errorf("failed to generate key pair: %w", err)
    }
    
    // Create certificate
    cert, err := createCertificate(keyPair, req)
    if err != nil {
        return nil, fmt.Errorf("failed to create certificate: %w", err)
    }
    
    return cert, nil
}
```

## Implementation Guide

### 1. Authentication Implementation
```go
// Example Authentication Middleware
func AuthMiddleware(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Extract token
        token := extractToken(r)
        if token == "" {
            http.Error(w, "unauthorized", http.StatusUnauthorized)
            return
        }
        
        // Validate token
        claims, err := validateToken(token)
        if err != nil {
            http.Error(w, "invalid token", http.StatusUnauthorized)
            return
        }
        
        // Add claims to context
        ctx := context.WithValue(r.Context(), "claims", claims)
        next.ServeHTTP(w, r.WithContext(ctx))
    })
}
```

### 2. Policy Implementation
```go
// Example Policy Structure
type Policy struct {
    ID          string   `json:"id"`
    Name        string   `json:"name"`
    Description string   `json:"description"`
    Rules       []Rule   `json:"rules"`
    CreatedAt   time.Time `json:"created_at"`
}

// Example Policy Evaluation
func EvaluatePolicy(ctx context.Context, req *PolicyRequest) (*PolicyResponse, error) {
    // Load policies
    policies, err := loadPolicies(ctx, req.IdentityID)
    if err != nil {
        return nil, fmt.Errorf("failed to load policies: %w", err)
    }
    
    // Evaluate policies
    result := evaluatePolicies(policies, req)
    
    return &PolicyResponse{
        Allowed: result.Allowed,
        Reason:  result.Reason,
    }, nil
}
```

## Testing

### 1. Unit Testing
```go
// Example Unit Test
func TestCreateIdentity(t *testing.T) {
    tests := []struct {
        name    string
        input   *Identity
        wantErr bool
    }{
        {
            name: "valid identity",
            input: &Identity{
                Name: "test-identity",
                Type: "service",
            },
            wantErr: false,
        },
        {
            name: "invalid identity",
            input: &Identity{
                Name: "",
                Type: "invalid",
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := CreateIdentity(context.Background(), tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("CreateIdentity() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

### 2. Integration Testing
```yaml
# Example Integration Test Configuration
integration_tests:
  setup:
    - name: "start_services"
      command: "make start-test-services"
    - name: "load_test_data"
      command: "make load-test-data"
  tests:
    - name: "authentication_flow"
      steps:
        - "create_identity"
        - "issue_certificate"
        - "validate_token"
    - name: "policy_evaluation"
      steps:
        - "create_policy"
        - "evaluate_access"
        - "verify_decision"
```

## Debugging

### 1. Logging
```go
// Example Logging Configuration
type Logger struct {
    level  string
    output io.Writer
}

func (l *Logger) Debug(msg string, fields map[string]interface{}) {
    if l.level == "debug" {
        l.log("DEBUG", msg, fields)
    }
}

func (l *Logger) Error(msg string, err error, fields map[string]interface{}) {
    l.log("ERROR", msg, fields)
    if err != nil {
        l.log("ERROR", err.Error(), nil)
    }
}
```

### 2. Tracing
```go
// Example Tracing Configuration
func setupTracing() {
    tracer := opentracing.GlobalTracer()
    
    // Configure sampling
    sampler := jaegercfg.SamplerConfig{
        Type:  "probabilistic",
        Param: 0.1,
    }
    
    // Configure reporter
    reporter := jaegercfg.ReporterConfig{
        LogSpans: true,
    }
    
    // Initialize tracer
    cfg := jaegercfg.Configuration{
        Sampler: &sampler,
        Reporter: &reporter,
    }
    
    tracer, closer, _ := cfg.NewTracer()
    defer closer.Close()
}
```

## Performance Optimization

### 1. Caching
```go
// Example Cache Configuration
type Cache struct {
    store    *cache.Cache
    ttl      time.Duration
}

func NewCache(ttl time.Duration) *Cache {
    return &Cache{
        store: cache.New(ttl, 10*time.Minute),
        ttl:   ttl,
    }
}

func (c *Cache) Get(key string) (interface{}, bool) {
    return c.store.Get(key)
}

func (c *Cache) Set(key string, value interface{}) {
    c.store.Set(key, value, c.ttl)
}
```

### 2. Connection Pooling
```go
// Example Connection Pool Configuration
type Pool struct {
    db *sql.DB
}

func NewPool(dsn string) (*Pool, error) {
    db, err := sql.Open("postgres", dsn)
    if err != nil {
        return nil, err
    }
    
    // Configure pool
    db.SetMaxOpenConns(25)
    db.SetMaxIdleConns(25)
    db.SetConnMaxLifetime(5 * time.Minute)
    
    return &Pool{db: db}, nil
}
```

## Security Best Practices

### 1. Secure Coding
```go
// Example Secure Coding Practices
func secureHandler(w http.ResponseWriter, r *http.Request) {
    // Set security headers
    w.Header().Set("X-Content-Type-Options", "nosniff")
    w.Header().Set("X-Frame-Options", "DENY")
    w.Header().Set("Content-Security-Policy", "default-src 'self'")
    
    // Validate input
    if err := validateInput(r); err != nil {
        http.Error(w, "invalid input", http.StatusBadRequest)
        return
    }
    
    // Process request
    // ...
}
```

### 2. Key Management
```go
// Example Key Management
type KeyManager struct {
    store KeyStore
}

func (km *KeyManager) RotateKey(ctx context.Context, keyID string) error {
    // Generate new key
    newKey, err := generateKey()
    if err != nil {
        return err
    }
    
    // Store new key
    if err := km.store.StoreKey(ctx, keyID, newKey); err != nil {
        return err
    }
    
    // Schedule old key deletion
    go km.scheduleKeyDeletion(ctx, keyID)
    
    return nil
}
```

## Contributing

### 1. Development Workflow
```yaml
# Development Workflow
workflow:
  steps:
    - name: "create_branch"
      command: "git checkout -b feature/your-feature"
    - name: "make_changes"
      command: "make changes"
    - name: "run_tests"
      command: "make test"
    - name: "create_pr"
      command: "git push origin feature/your-feature"
```

### 2. Code Review Guidelines
```yaml
# Code Review Guidelines
code_review:
  required:
    - "unit_tests"
    - "integration_tests"
    - "documentation"
    - "security_review"
  checklist:
    - "code_style"
    - "error_handling"
    - "logging"
    - "performance"
```

## Conclusion

This guide provides comprehensive development instructions for the workload identity system. Remember to:
- Follow coding standards
- Write comprehensive tests
- Document your code
- Review security implications
- Optimize performance

For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [API Reference](api_reference.md)
- [Security Best Practices](security_best_practices.md) 