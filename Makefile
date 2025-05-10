# Workload Identity Project Makefile
# This file contains common tasks for managing the workload identity project

.PHONY: all clean test verify lint docs

# Default target
all: verify lint test

# Clean build artifacts
clean:
	rm -rf build/
	rm -rf dist/
	find . -type f -name "*.pyc" -delete
	find . -type d -name "__pycache__" -delete

# Run tests
test:
	@echo "Running tests..."
	# Add your test commands here

# Verify configuration
verify:
	@echo "Verifying configuration..."
	./scripts/verify_identity.sh

# Run linters
lint:
	@echo "Running linters..."
	# Add your lint commands here

# Generate documentation
docs:
	@echo "Generating documentation..."
	# Add your documentation generation commands here

# Install dependencies
deps:
	@echo "Installing dependencies..."
	# Add your dependency installation commands here

# Build the project
build:
	@echo "Building project..."
	# Add your build commands here

# Help target
help:
	@echo "Available targets:"
	@echo "  all        - Run verify, lint, and test"
	@echo "  clean      - Remove build artifacts"
	@echo "  test       - Run tests"
	@echo "  verify     - Verify configuration"
	@echo "  lint       - Run linters"
	@echo "  docs       - Generate documentation"
	@echo "  deps       - Install dependencies"
	@echo "  build      - Build the project" 