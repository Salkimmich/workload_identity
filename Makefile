# Workload Identity Project Makefile
# This file contains common tasks for managing the workload identity project

.PHONY: all clean test verify lint docs setup-dev setup-git-hooks start-local-cluster generate-dev-certs setup-test-env verify-certs regenerate-certs check-cluster reset-cluster security-diagnostics verify-security

# Default target
all: verify lint test

# Development Environment Setup
setup-dev:
	@echo "Setting up development environment..."
	@./scripts/setup-dev.sh

setup-git-hooks:
	@echo "Setting up git hooks..."
	@cp scripts/git-hooks/* .git/hooks/
	@chmod +x .git/hooks/*

start-local-cluster:
	@echo "Starting local Kubernetes cluster..."
	@./scripts/start-local-cluster.sh

generate-dev-certs:
	@echo "Generating development certificates..."
	@./scripts/generate-dev-certs.sh

# Testing Environment
setup-test-env:
	@echo "Setting up test environment..."
	@./scripts/setup-test-env.sh

test-unit:
	@echo "Running unit tests..."
	@go test -v ./pkg/... -tags=unit

test-integration:
	@echo "Running integration tests..."
	@go test -v ./tests/integration/... -tags=integration

test-security:
	@echo "Running security tests..."
	@./scripts/run-security-tests.sh

test-e2e:
	@echo "Running end-to-end tests..."
	@./scripts/run-e2e-tests.sh

# Security Commands
security:
	@echo "Running security checks..."
	@gosec ./...
	@trivy fs .
	@dependency-check --scan .

verify-certs:
	@echo "Verifying certificates..."
	@./scripts/verify-certs.sh

regenerate-certs:
	@echo "Regenerating certificates..."
	@./scripts/regenerate-certs.sh

security-diagnostics:
	@echo "Running security diagnostics..."
	@./scripts/security-diagnostics.sh

verify-security:
	@echo "Verifying security configuration..."
	@./scripts/verify-security.sh

# Kubernetes Commands
check-cluster:
	@echo "Checking cluster status..."
	@kubectl cluster-info
	@kubectl get nodes
	@kubectl get pods -A

reset-cluster:
	@echo "Resetting development cluster..."
	@./scripts/reset-cluster.sh

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf build/
	rm -rf dist/
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete

# Run tests
test: test-unit test-integration test-security test-e2e

# Verify configuration
verify:
	@echo "Verifying configuration..."
	@./scripts/verify_identity.sh

# Run linters
lint:
	@echo "Running linters..."
	@golangci-lint run

# Generate documentation
docs:
	@echo "Generating documentation..."
	@./scripts/generate-docs.sh

# Install dependencies
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Build the project
build:
	@echo "Building project..."
	@go build -o bin/workload-identity ./cmd/workload-identity

# Help target
help:
	@echo "Available targets:"
	@echo "  all                - Run verify, lint, and test"
	@echo "  clean              - Remove build artifacts"
	@echo "  test               - Run all tests"
	@echo "  test-unit          - Run unit tests"
	@echo "  test-integration   - Run integration tests"
	@echo "  test-security      - Run security tests"
	@echo "  test-e2e           - Run end-to-end tests"
	@echo "  verify             - Verify configuration"
	@echo "  lint               - Run linters"
	@echo "  docs               - Generate documentation"
	@echo "  deps               - Install dependencies"
	@echo "  build              - Build the project"
	@echo "  setup-dev          - Set up development environment"
	@echo "  setup-git-hooks    - Set up git hooks"
	@echo "  start-local-cluster - Start local Kubernetes cluster"
	@echo "  generate-dev-certs - Generate development certificates"
	@echo "  setup-test-env     - Set up test environment"
	@echo "  verify-certs       - Verify certificates"
	@echo "  regenerate-certs   - Regenerate certificates"
	@echo "  check-cluster      - Check cluster status"
	@echo "  reset-cluster      - Reset development cluster"
	@echo "  security           - Run security checks"
	@echo "  security-diagnostics - Run security diagnostics"
	@echo "  verify-security    - Verify security configuration" 