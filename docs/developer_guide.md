# Developer Guide

This guide provides comprehensive instructions for developers working on the workload identity system.

## Table of Contents
1. [Getting Started](#getting-started)
2. [Development Environment](#development-environment)
3. [Code Structure](#code-structure)
4. [Testing](#testing)
5. [Security](#security)
6. [Contributing](#contributing)

## Getting Started

### Prerequisites
```bash
# Required tools
go >= 1.20
docker >= 20.10
kubectl >= 1.24
kind >= 0.20
openssl >= 3.0

# Required services
kubernetes >= 1.24
postgresql >= 14
```

### Quick Start
```bash
# Clone repository
git clone https://github.com/your-org/workload-identity.git
cd workload-identity

# Setup development environment
make setup-dev

# Start local cluster
make start-local-cluster

# Run tests
make test
```

## Development Environment

### 1. Local Setup
```bash
# Setup git hooks
make setup-git-hooks

# Generate development certificates
make generate-dev-certs

# Start local cluster
make start-local-cluster
```

### 2. IDE Setup (VSCode)
```json
{
  "go.useLanguageServer": true,
  "go.lintTool": "golangci-lint",
  "go.lintFlags": [
    "--fast"
  ],
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.organizeImports": true
  }
}
```

### 3. Common Commands
```bash
# Run tests
make test

# Run linter
make lint

# Run security checks
make security-check

# Build binaries
make build

# Clean build artifacts
make clean
```

## Code Structure

### 1. Project Layout
```
.
├── cmd/                    # Command-line applications
│   ├── server/            # SPIRE server
│   └── agent/             # SPIRE agent
├── pkg/                   # Library code
│   ├── auth/             # Authentication
│   ├── cert/             # Certificate management
│   ├── identity/         # Identity management
│   └── security/         # Security utilities
├── internal/             # Private application code
├── api/                  # API definitions
├── configs/             # Configuration files
├── scripts/             # Build and utility scripts
└── test/                # Test utilities
```

### 2. Key Components
```go
// Example: Identity Provider Interface
type IdentityProvider interface {
    // IssueIdentity issues a new identity for a workload
    IssueIdentity(ctx context.Context, workload *Workload) (*Identity, error)
    
    // ValidateIdentity validates an existing identity
    ValidateIdentity(ctx context.Context, identity *Identity) error
    
    // RevokeIdentity revokes an existing identity
    RevokeIdentity(ctx context.Context, identity *Identity) error
}

// Example: Certificate Manager Interface
type CertificateManager interface {
    // GenerateCertificate generates a new certificate
    GenerateCertificate(ctx context.Context, request *CertificateRequest) (*Certificate, error)
    
    // ValidateCertificate validates an existing certificate
    ValidateCertificate(ctx context.Context, cert *Certificate) error
    
    // RevokeCertificate revokes an existing certificate
    RevokeCertificate(ctx context.Context, cert *Certificate) error
}
```

## Testing

### 1. Unit Tests
```go
// Example: Unit Test
func TestIdentityProvider_IssueIdentity(t *testing.T) {
    tests := []struct {
        name     string
        workload *Workload
        want     *Identity
        wantErr  bool
    }{
        {
            name: "valid workload",
            workload: &Workload{
                ID:   "test-workload",
                Type: "kubernetes",
            },
            want: &Identity{
                ID:     "test-workload",
                Status: "active",
            },
            wantErr: false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := provider.IssueIdentity(context.Background(), tt.workload)
            if (err != nil) != tt.wantErr {
                t.Errorf("IssueIdentity() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("IssueIdentity() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### 2. Integration Tests
```go
// Example: Integration Test
func TestIdentityFlow(t *testing.T) {
    // Setup test environment
    env := setupTestEnv(t)
    defer env.Cleanup()
    
    // Create test workload
    workload := createTestWorkload(t, env)
    
    // Issue identity
    identity, err := env.Provider.IssueIdentity(context.Background(), workload)
    require.NoError(t, err)
    require.NotNil(t, identity)
    
    // Validate identity
    err = env.Provider.ValidateIdentity(context.Background(), identity)
    require.NoError(t, err)
    
    // Revoke identity
    err = env.Provider.RevokeIdentity(context.Background(), identity)
    require.NoError(t, err)
}
```

### 3. Security Tests
```go
// Example: Security Test
func TestCertificateSecurity(t *testing.T) {
    tests := []struct {
        name    string
        cert    *Certificate
        wantErr bool
    }{
        {
            name: "valid certificate",
            cert: &Certificate{
                Subject: "test-workload",
                NotBefore: time.Now(),
                NotAfter:  time.Now().Add(24 * time.Hour),
            },
            wantErr: false,
        },
        {
            name: "expired certificate",
            cert: &Certificate{
                Subject: "test-workload",
                NotBefore: time.Now().Add(-48 * time.Hour),
                NotAfter:  time.Now().Add(-24 * time.Hour),
            },
            wantErr: true,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := validateCertificate(tt.cert)
            if (err != nil) != tt.wantErr {
                t.Errorf("validateCertificate() error = %v, wantErr %v", err, tt.wantErr)
            }
        })
    }
}
```

## Security

### 1. Secure Coding Practices
```go
// Example: Secure Input Validation
func validateInput(input string) error {
    // Check for SQL injection
    if strings.Contains(input, ";") {
        return errors.New("invalid input: contains SQL injection attempt")
    }
    
    // Check for XSS
    if strings.Contains(input, "<script>") {
        return errors.New("invalid input: contains XSS attempt")
    }
    
    return nil
}

// Example: Secure Error Handling
func handleError(err error) {
    // Log error securely
    log.Printf("error: %v", err)
    
    // Don't expose internal errors
    if isInternalError(err) {
        err = errors.New("internal server error")
    }
    
    return err
}
```

### 2. Security Checks
```bash
# Run security checks
make security-check

# Run dependency checks
make dependency-check

# Run SAST
make sast

# Run SCA
make sca
```

## Contributing

### 1. Development Workflow
1. Create feature branch
2. Make changes
3. Run tests
4. Submit PR
5. Address review comments
6. Merge changes

### 2. Code Review Guidelines
- Follow Go best practices
- Write comprehensive tests
- Update documentation
- Check security implications
- Verify performance impact

### 3. Release Process
```bash
# Create release branch
git checkout -b release/v1.0.0

# Update version
make version VERSION=1.0.0

# Run release checks
make release-check

# Create release
make release
```

## Testing Infrastructure

### Directory Structure
```
tests/
├── unit/                    # Unit tests
├── integration/            # Integration tests
├── e2e/                    # End-to-end tests
├── security/              # Security tests
└── fixtures/              # Test fixtures and mocks
```

### Test Categories

1. **Unit Tests**
   - Core functionality tests
   - Identity management tests
   - Security-related tests
   - Coverage requirement: > 80%

2. **Integration Tests**
   - Kubernetes integration tests
   - Cloud provider integration tests
   - SPIRE integration tests
   - Coverage requirement: > 70%

3. **End-to-End Tests**
   - Common usage scenarios
   - Performance test scenarios
   - Critical paths only

4. **Security Tests**
   - Penetration tests
   - Fuzzing tests
   - All security-critical paths

### Test Fixtures

The project includes comprehensive test fixtures in the `tests/fixtures` directory:

1. **Kubernetes Fixtures**
   ```yaml
   # Example: test-pod.yaml
   apiVersion: v1
   kind: Pod
   metadata:
     annotations:
       workload-identity.io/cloud-provider: "aws"
       workload-identity.io/role-arn: "arn:aws:iam::123456789012:role/test-role"
   ```

2. **Cloud Provider Fixtures**
   ```json
   // Example: aws-test-config.json
   {
     "Version": "2012-10-17",
     "Statement": [
       {
         "Effect": "Allow",
         "Action": ["sts:AssumeRoleWithWebIdentity"],
         "Resource": "arn:aws:iam::123456789012:role/test-role"
       }
     ]
   }
   ```

3. **Certificate Fixtures**
   ```bash
   # Generate test certificates
   ./tests/fixtures/certs/generate-test-certs.sh \
     --output-dir ./certs \
     --validity 365 \
     --ca-name "Test CA"
   ```

### Running Tests

1. **Unit Tests**
   ```bash
   # Run all unit tests
   go test ./tests/unit/...

   # Run specific test
   go test ./tests/unit/core/...

   # Run with coverage
   go test -cover ./tests/unit/...
   ```

2. **Integration Tests**
   ```bash
   # Run all integration tests
   go test ./tests/integration/...

   # Run specific provider tests
   go test ./tests/integration/cloud/...

   # Run with verbose output
   go test -v ./tests/integration/...
   ```

3. **End-to-End Tests**
   ```bash
   # Run all E2E tests
   go test ./tests/e2e/...

   # Run specific scenario
   go test ./tests/e2e/scenarios/...

   # Run performance tests
   go test ./tests/e2e/performance/...
   ```

### Test Environment Setup

1. **Prerequisites**
   - Go 1.21 or later
   - Docker
   - kubectl
   - kind (for local Kubernetes testing)
   - Cloud provider credentials

2. **Local Setup**
   ```bash
   # Set up test cluster
   make setup-test-cluster

   # Configure cloud credentials
   make setup-cloud-credentials

   # Start test dependencies
   make start-test-dependencies
   ```

### Best Practices

1. **Writing Tests**
   - Use table-driven tests
   - Mock external dependencies
   - Keep tests focused and atomic
   - Use descriptive test names

2. **Test Fixtures**
   - Keep fixtures minimal
   - Document dependencies
   - Version control fixture data
   - Clean up after use

3. **Security**
   - Use test credentials
   - Encrypt sensitive data
   - Rotate test keys
   - Clean up secrets

## Conclusion

This guide provides comprehensive instructions for developers. For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Deployment Guide](deployment_guide.md)
- [API Reference](api_reference.md)
