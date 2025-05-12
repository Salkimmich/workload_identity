#!/bin/bash

# Development Environment Setup Script
# This script sets up the development environment for the workload identity project
# Required: This script must be run from the project root directory

set -e  # Required: Exit on any error

# Function to check if a command exists
# Required: Ensures all necessary tools are installed
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is required but not installed."
        exit 1
    fi
}

# Function to check version requirements
# Required: Ensures tools meet minimum version requirements
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

# Required: Check for essential development tools
check_command "go"        # Required: Go programming language
check_command "docker"    # Required: Container runtime
check_command "kubectl"   # Required: Kubernetes CLI
check_command "make"      # Required: Build automation
check_command "golangci-lint"  # Required: Code quality
check_command "trivy"     # Required: Security scanning

# Required: Verify minimum versions for compatibility
check_version "go" "1.20"           # Required: Minimum Go version
check_version "docker" "20.10"      # Required: Minimum Docker version
check_version "kubectl" "1.24"      # Required: Minimum kubectl version
check_version "make" "4.0"          # Required: Minimum make version
check_version "golangci-lint" "1.50"  # Required: Minimum linter version
check_version "trivy" "0.30"        # Required: Minimum security scanner version

echo "Installing development dependencies..."

# Required: Install Go dependencies
go mod download  # Required: Download all dependencies
go mod tidy      # Required: Clean up dependencies

# Required: Install development tools based on OS
if [ "$(uname)" == "Darwin" ]; then
    # macOS
    brew install golangci-lint trivy  # Required: Install via Homebrew
elif [ "$(expr substr $(uname -s) 1 5)" == "Linux" ]; then
    # Linux
    # Required: Install golangci-lint
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.50.0
    # Required: Install trivy
    curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b /usr/local/bin v0.30.0
fi

echo "Setting up development environment..."

# Required: Create essential directories
mkdir -p build    # Required: For build artifacts
mkdir -p dist     # Required: For distribution files
mkdir -p .cache   # Required: For caching

# Optional: Set up git hooks if in a git repository
if [ -d ".git" ]; then
    echo "Setting up git hooks..."
    cp scripts/git-hooks/* .git/hooks/  # Optional: Copy git hooks
    chmod +x .git/hooks/*               # Required: Make hooks executable
fi

# Required: Set up development certificates
echo "Generating development certificates..."
./scripts/generate-dev-certs.sh  # Required: Generate TLS certificates

# Required: Set up local Kubernetes cluster
echo "Setting up local Kubernetes cluster..."
./scripts/start-local-cluster.sh  # Required: Start local development cluster

# Required: Verify the setup
echo "Verifying development environment..."
./scripts/verify_identity.sh  # Required: Verify identity system is working

echo "Development environment setup complete!"
echo "You can now start developing the workload identity system."
echo "Run 'make help' to see available commands."

# Optional: Additional setup steps that might be needed:
# - Configure IDE settings
# - Set up additional development tools
# - Configure local DNS
# - Set up monitoring tools
# - Configure logging 