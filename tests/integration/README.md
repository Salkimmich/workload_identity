# Integration Tests

This directory contains integration tests that verify the interaction between different components of the Workload Identity system.

## Directory Structure

```
integration/
├── kubernetes/         # Kubernetes integration tests
├── cloud/             # Cloud provider integration tests
└── spire/             # SPIRE integration tests
```

## Testing Guidelines

1. Use real dependencies where possible
2. Clean up resources after tests
3. Handle timeouts appropriately
4. Use test fixtures for common setup
5. Document external dependencies

## Environment Setup

### Kubernetes
```bash
# Start local cluster
kind create cluster --name workload-identity-test

# Deploy test dependencies
kubectl apply -f fixtures/k8s/
```

### Cloud Providers
```bash
# AWS
export AWS_PROFILE=test
export AWS_REGION=us-west-2

# GCP
export GOOGLE_APPLICATION_CREDENTIALS=./fixtures/gcp/credentials.json

# Azure
export AZURE_TENANT_ID=test
export AZURE_CLIENT_ID=test
export AZURE_CLIENT_SECRET=test
```

### SPIRE
```bash
# Deploy SPIRE server
kubectl apply -f fixtures/spire/server.yaml

# Deploy SPIRE agent
kubectl apply -f fixtures/spire/agent.yaml
```

## Running Tests

```bash
# Run all integration tests
go test ./tests/integration/...

# Run specific provider tests
go test ./tests/integration/cloud/...

# Run with verbose output
go test -v ./tests/integration/...
```

## Test Categories

### Kubernetes Tests
- Pod identity injection
- Service account token projection
- Network policy enforcement
- RBAC integration

### Cloud Provider Tests
- IAM role assumption
- Token exchange
- Service account creation
- Permission validation

### SPIRE Tests
- Node attestation
- Workload attestation
- Certificate issuance
- Identity validation 