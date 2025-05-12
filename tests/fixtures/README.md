# Test Fixtures

This directory contains test fixtures, mocks, and helper utilities for the Workload Identity test suite.

## Directory Structure

```
fixtures/
├── k8s/              # Kubernetes test fixtures
├── cloud/            # Cloud provider test fixtures
├── spire/            # SPIRE test fixtures
├── certs/            # Certificate fixtures
└── mocks/            # Mock implementations
```

## Usage Guidelines

1. Keep fixtures minimal and focused
2. Document fixture dependencies
3. Version control fixture data
4. Clean up after use
5. Use realistic test data

## Fixture Categories

### Kubernetes Fixtures
- Test cluster configurations
- Pod specifications
- Service account definitions
- Network policies
- RBAC rules

### Cloud Provider Fixtures
- IAM role definitions
- Service account configurations
- Token samples
- API responses

### SPIRE Fixtures
- Server configurations
- Agent configurations
- Node attestation data
- Workload attestation data

### Certificate Fixtures
- Root certificates
- Intermediate certificates
- Leaf certificates
- Key pairs
- Certificate chains

### Mock Implementations
- Cloud provider clients
- Kubernetes clients
- SPIRE clients
- Token providers

## Creating Fixtures

### Example: Kubernetes Fixture
```yaml
# fixtures/k8s/test-pod.yaml
apiVersion: v1
kind: Pod
metadata:
  name: test-pod
  namespace: default
spec:
  serviceAccountName: test-sa
  containers:
  - name: test-container
    image: test-image:latest
```

### Example: Certificate Fixture
```bash
# Generate test certificates
./scripts/generate-test-certs.sh \
  --output-dir fixtures/certs \
  --validity 365 \
  --ca-name "Test CA"
```

## Best Practices

1. Fixture Management
   - Use version control
   - Document dependencies
   - Include cleanup scripts
   - Validate fixture data

2. Mock Implementation
   - Implement interfaces
   - Document behavior
   - Include test cases
   - Version mock data

3. Security Considerations
   - Use test credentials
   - Encrypt sensitive data
   - Rotate test keys
   - Clean up secrets

## Contributing

1. Follow existing patterns
2. Document new fixtures
3. Include test cases
4. Update this README 