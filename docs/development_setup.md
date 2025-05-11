# Development Environment Setup Guide

This guide provides detailed instructions for setting up a development environment for the workload identity system. It covers all necessary prerequisites, tools, and configurations needed to start development.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Local Development Environment](#local-development-environment)
3. [Security Setup](#security-setup)
4. [Development Tools](#development-tools)
5. [Testing Environment](#testing-environment)
6. [Troubleshooting](#troubleshooting)

## Prerequisites

### Required Tools
```bash
# Required tools and their minimum versions
- Go >= 1.20
- Docker >= 20.10
- kubectl >= 1.24
- make >= 4.0
- golangci-lint >= 1.50
- trivy >= 0.30
- kind >= 0.20
- openssl >= 3.0
```

### Required Services
```bash
# Required services and their minimum versions
- Kubernetes >= 1.24
- PostgreSQL >= 14.0
```

### Security Tools
```bash
# Required security tools and their minimum versions
- gosec >= 2.15
- dependency-check >= 7.0
```

## Local Development Environment

### 1. Clone the Repository
```bash
# Clone the repository
git clone https://github.com/your-org/workload-identity.git
cd workload-identity

# Set up git hooks for security
make setup-git-hooks
```

### 2. Development Environment Setup
```bash
# Set up development environment
make setup-dev

# This will:
# - Install required dependencies
# - Set up local Kubernetes cluster
# - Configure development tools
# - Initialize security configurations
```

### 3. Local Kubernetes Cluster
```bash
# Start local development cluster
make start-local-cluster

# This will:
# - Start a local Kubernetes cluster using kind
# - Deploy required components
# - Configure SPIRE server and agent
# - Set up monitoring tools
```

## Security Setup

### 1. Certificate Generation
```bash
# Generate development certificates
make generate-dev-certs

# This will:
# - Generate root CA
# - Create server certificates
# - Generate agent certificates
# - Set up certificate rotation
# - Create Kubernetes secrets
```

### 2. Security Configuration
```yaml
# Development Security Configuration
security:
  tls:
    enabled: true
    min_version: "TLS1.3"
    cipher_suites:
      - "TLS_AES_128_GCM_SHA256"
      - "TLS_AES_256_GCM_SHA384"
  
  auth:
    required: true
    jwt:
      issuer: "dev-environment"
      audience: ["development"]
  
  monitoring:
    audit_logging: true
    security_metrics: true
```

## Development Tools

### 1. IDE Setup
```yaml
# VSCode Configuration
vscode:
  extensions:
    - "golang.go"
    - "ms-kubernetes-tools.vscode-kubernetes-tools"
    - "redhat.vscode-yaml"
    - "github.copilot"
    - "github.vscode-codeql"
    - "aquasecurity.trivy-vulnerability-scanner"
  
  settings:
    "go.formatTool": "gofmt"
    "go.lintTool": "golangci-lint"
    "security.enableCodeAnalysis": true
    "security.enableDependencyScanning": true
```

### 2. Development Workflow
```bash
# Common development commands
make test        # Run tests
make lint        # Run linters
make security    # Run security checks
make build       # Build the project
make verify      # Verify configuration
```

## Testing Environment

### 1. Test Setup
```bash
# Set up test environment
make setup-test-env

# This will:
# - Create test namespace
# - Deploy test components
# - Configure test certificates
# - Set up test monitoring
```

### 2. Running Tests
```bash
# Run different types of tests
make test-unit        # Run unit tests
make test-integration # Run integration tests
make test-security    # Run security tests
make test-e2e         # Run end-to-end tests
```

## Troubleshooting

### Common Issues

1. **Certificate Issues**
```bash
# Check certificate validity
make verify-certs

# Regenerate certificates if needed
make regenerate-certs
```

2. **Kubernetes Issues**
```bash
# Check cluster status
make check-cluster

# Reset development cluster
make reset-cluster
```

3. **Security Issues**
```bash
# Run security diagnostics
make security-diagnostics

# Check security configuration
make verify-security
```

### Getting Help

- Check the [Architecture Guide](docs/architecture_guide.md) for system design details
- Refer to the [Security Best Practices](docs/security_best_practices.md) for security guidelines
- Join the development Slack channel for real-time support
- Open an issue on GitHub for bug reports or feature requests

## Additional Resources

- [SPIFFE Documentation](https://spiffe.io/docs/latest/)
- [SPIRE Documentation](https://spiffe.io/spire/docs/latest/)
- [Kubernetes Security Best Practices](https://kubernetes.io/docs/concepts/security/)
- [Go Security Best Practices](https://golang.org/doc/security) 