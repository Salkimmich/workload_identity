# Unit Tests

This directory contains unit tests for individual components of the Workload Identity system.

## Directory Structure

```
unit/
├── core/               # Core functionality tests
├── identity/          # Identity management tests
└── security/          # Security-related tests
```

## Testing Guidelines

1. Each test file should be named `*_test.go`
2. Use table-driven tests for multiple test cases
3. Mock external dependencies
4. Keep tests focused and atomic
5. Use descriptive test names

## Example Test Structure

```go
func TestComponent(t *testing.T) {
    tests := []struct {
        name     string
        input    interface{}
        expected interface{}
        wantErr  bool
    }{
        // Test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Coverage Requirements

- Minimum coverage: 80%
- Critical paths: 100%
- Error handling: 100%

## Running Tests

```bash
# Run all unit tests
go test ./tests/unit/...

# Run specific test
go test ./tests/unit/core/...

# Run with coverage
go test -cover ./tests/unit/...
``` 