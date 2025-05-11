#!/bin/bash

# Development Environment Setup Script
# This script sets up the development environment for the workload identity project

set -e

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is required but not installed."
        exit 1
    fi
}

# Function to check version requirements
check_version() {
    local cmd=$1
    local min_version=$2
    local version
    
    version=$($cmd --version 2>&1 | grep -oE '[0-9]+\.[0-9]+(\.[0-9]+)?' | head -1)
    
    if [ "$(printf '%s\n' "$min_version" "$version" | sort -V | head -n1)" != "$min_version" ]; then
        echo "Error: $cmd version $min_version or higher is required. Found version $version"
        exit 1
    fi
}

echo "Checking prerequisites..."

# Check required commands
check_command "go"
check_command "docker"
check_command "kubectl"
check_command "make"
check_command "golangci-lint"
check_command "trivy"

# Check versions
check_version "go" "1.20"
check_version "docker" "20.10"
check_version "kubectl" "1.24"
check_version "make" "4.0"
check_version "golangci-lint" "1.50"
check_version "trivy" "0.30"

echo "Installing development dependencies..."

# Install Go dependencies
go mod download
go mod tidy

# Install development tools
if [ "$(uname)" == "Darwin" ]; then
    # macOS
    brew install golangci-lint trivy
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.0
    curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.30.0
fi

echo "Setting up development environment..."

# Create necessary directories
mkdir -p build
mkdir -p dist
mkdir -p .cache

# Set up git hooks
if [ -d ".git" ]; then
    echo "Setting up git hooks..."
    cp scripts/git-hooks/* .git/hooks/
    chmod +x .git/hooks/*
fi

# Set up development certificates
echo "Generating development certificates..."
./scripts/generate-dev-certs.sh

# Set up local Kubernetes cluster
echo "Setting up local Kubernetes cluster..."
./scripts/start-local-cluster.sh

# Verify the setup
echo "Verifying development environment..."
./scripts/verify_identity.sh

echo "Development environment setup complete!"
echo "You can now start developing the workload identity system."
echo "Run 'make help' to see available commands." 