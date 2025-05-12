# Security Tests

This directory contains security-focused tests for the Workload Identity system, including penetration tests and fuzzing tests.

## Directory Structure

```
security/
├── penetration/       # Penetration tests
└── fuzzing/          # Fuzzing tests
```

## Testing Guidelines

1. Follow security testing best practices
2. Document all test cases
3. Handle sensitive data appropriately
4. Include cleanup procedures
5. Report security findings

## Test Categories

### Penetration Tests
- Token manipulation
- Identity spoofing
- Access control bypass
- Network security
- Certificate attacks
- API security

### Fuzzing Tests
- Input validation
- Certificate handling
- Token parsing
- Configuration validation
- Protocol fuzzing

## Security Requirements

### Prerequisites
- Security testing tools
- Test credentials
- Network access
- Monitoring tools

### Configuration
```bash
# Security test environment
export SECURITY_TEST_MODE=true
export FUZZ_DURATION=1h
export PEN_TEST_TIMEOUT=2h

# Monitoring configuration
export ENABLE_SECURITY_LOGGING=true
export LOG_LEVEL=debug
```

## Running Tests

```bash
# Run all security tests
go test ./tests/security/...

# Run penetration tests
go test ./tests/security/penetration/...

# Run fuzzing tests
go test -fuzz ./tests/security/fuzzing/...

# Run with security logging
go test -v -security-logging ./tests/security/...
```

## Security Test Cases

### Penetration Test Cases
1. Token Manipulation
   - Invalid signatures
   - Expired tokens
   - Modified claims
   - Token replay

2. Identity Spoofing
   - Pod identity spoofing
   - Service account spoofing
   - Node identity spoofing

3. Access Control
   - Permission escalation
   - Role bypass
   - Policy circumvention

### Fuzzing Test Cases
1. Input Validation
   - Malformed requests
   - Invalid parameters
   - Boundary conditions

2. Certificate Handling
   - Invalid certificates
   - Expired certificates
   - Malformed chains

## Reporting

### Security Findings
- Vulnerability reports
- Risk assessments
- Mitigation steps
- Remediation tracking

### Test Results
- Test coverage
- Success/failure rates
- Performance impact
- Resource usage 