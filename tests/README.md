# Workload Identity Testing Infrastructure

This directory contains the complete test suite for the Workload Identity system. The testing infrastructure is designed to ensure reliability, security, and performance across all components.

## Directory Structure

```
tests/
├── unit/                    # Unit tests for individual components
├── integration/            # Integration tests for component interactions
├── e2e/                    # End-to-end tests for complete workflows
├── security/              # Security-focused tests
└── fixtures/              # Test fixtures and mocks
```

## Test Categories

### Unit Tests
- Core functionality tests
- Identity management tests
- Security-related tests

### Integration Tests
- Kubernetes integration tests
- Cloud provider integration tests
- SPIRE integration tests

### End-to-End Tests
- Common usage scenarios
- Performance test scenarios

### Security Tests
- Penetration tests
- Fuzzing tests

## Running Tests

### Prerequisites
- Go 1.21 or later
- Docker
- kubectl
- kind (for local Kubernetes testing)
- Cloud provider credentials (for cloud integration tests)

### Basic Test Commands

```bash
# Run all unit tests
go test ./tests/unit/...

# Run all integration tests
go test ./tests/integration/...

# Run all e2e tests
go test ./tests/e2e/...

# Run all security tests
go test ./tests/security/...
```

## Test Environment Setup

1. Set up local Kubernetes cluster:
```bash
make setup-test-cluster
```

2. Configure cloud provider credentials:
```bash
make setup-cloud-credentials
```

3. Start test dependencies:
```bash
make start-test-dependencies
```

## Contributing Tests

1. Follow the existing test patterns in each directory
2. Use the provided test fixtures and mocks
3. Document any new test dependencies
4. Update this README if adding new test categories

## Test Coverage Requirements

- Unit tests: > 80% coverage
- Integration tests: > 70% coverage
- E2E tests: Critical paths only
- Security tests: All security-critical paths 