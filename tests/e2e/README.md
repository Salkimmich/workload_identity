# End-to-End Tests

This directory contains end-to-end tests that verify complete workflows and scenarios in the Workload Identity system.

## Directory Structure

```
e2e/
├── scenarios/         # Common usage scenarios
└── performance/       # Performance test scenarios
```

## Testing Guidelines

1. Test complete user workflows
2. Include cleanup procedures
3. Handle timeouts and retries
4. Document test prerequisites
5. Use realistic test data

## Test Scenarios

### Common Scenarios
- Complete identity lifecycle
- Multi-cloud deployment
- Failover and recovery
- Upgrade paths
- Security breach scenarios

### Performance Scenarios
- Load testing
- Latency measurements
- Resource usage
- Scalability tests

## Environment Requirements

### Prerequisites
- Kubernetes cluster
- Cloud provider access
- SPIRE deployment
- Test data sets

### Configuration
```bash
# Set test environment
export TEST_ENV=staging
export TEST_CLUSTER=workload-identity-test

# Configure test parameters
export TEST_TIMEOUT=30m
export TEST_PARALLEL=4
```

## Running Tests

```bash
# Run all E2E tests
go test ./tests/e2e/...

# Run specific scenario
go test ./tests/e2e/scenarios/...

# Run performance tests
go test ./tests/e2e/performance/...

# Run with timeout
go test -timeout 30m ./tests/e2e/...
```

## Test Data Management

### Test Data Sets
- Identity configurations
- Policy definitions
- Certificate chains
- Token samples

### Data Cleanup
- Automatic cleanup after tests
- Manual cleanup procedures
- Data retention policies

## Performance Metrics

### Key Metrics
- Response time
- Throughput
- Resource utilization
- Error rates

### Benchmarking
- Baseline measurements
- Performance thresholds
- Regression detection 