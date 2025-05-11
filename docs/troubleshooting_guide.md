# Troubleshooting Guide

This guide provides solutions for common issues and problems that may arise when working with the workload identity system.

## Table of Contents
1. [Certificate Issues](#certificate-issues)
2. [Kubernetes Integration](#kubernetes-integration)
3. [Security Issues](#security-issues)
4. [Performance Issues](#performance-issues)
5. [Common Commands](#common-commands)

## Certificate Issues

### 1. Certificate Generation Failures
```bash
# Check certificate generation logs
kubectl logs -n spire deployment/spire-server

# Verify certificate directory permissions
ls -la certs/dev/

# Check OpenSSL version
openssl version
```

Common Solutions:
- Ensure OpenSSL 3.0+ is installed
- Verify directory permissions (should be 700)
- Check for sufficient disk space
- Verify system time is correct

### 2. Certificate Validation Errors
```bash
# Verify certificate chain
openssl verify -CAfile certs/dev/ca.crt certs/dev/server.crt

# Check certificate details
openssl x509 -in certs/dev/server.crt -text -noout

# Verify certificate expiration
openssl x509 -in certs/dev/server.crt -noout -dates
```

Common Solutions:
- Ensure CA certificate is trusted
- Check certificate expiration dates
- Verify certificate chain is complete
- Check for certificate revocation

### 3. Certificate Renewal Issues
```bash
# Check renewal logs
kubectl logs -n spire deployment/spire-agent

# Verify renewal configuration
kubectl get configmap -n spire spire-agent-config -o yaml

# Check certificate status
kubectl get secret -n spire spire-agent-cert -o yaml
```

Common Solutions:
- Verify renewal configuration
- Check agent connectivity
- Ensure sufficient permissions
- Monitor renewal logs

## Kubernetes Integration

### 1. SPIRE Server Issues
```bash
# Check server status
kubectl get pods -n spire -l app=spire-server

# View server logs
kubectl logs -n spire deployment/spire-server

# Verify server configuration
kubectl get configmap -n spire spire-server-config -o yaml
```

Common Solutions:
- Verify server configuration
- Check resource limits
- Ensure proper networking
- Verify service account permissions

### 2. SPIRE Agent Issues
```bash
# Check agent status
kubectl get pods -n spire -l app=spire-agent

# View agent logs
kubectl logs -n spire daemonset/spire-agent

# Verify agent configuration
kubectl get configmap -n spire spire-agent-config -o yaml
```

Common Solutions:
- Verify agent configuration
- Check node connectivity
- Ensure proper permissions
- Monitor agent logs

### 3. Workload Registration Issues
```bash
# Check workload status
kubectl get pods -n demo

# View workload logs
kubectl logs -n demo deployment/frontend

# Verify workload configuration
kubectl get configmap -n demo frontend-config -o yaml
```

Common Solutions:
- Verify workload configuration
- Check service account permissions
- Ensure proper annotations
- Monitor workload logs

## Security Issues

### 1. Authentication Failures
```bash
# Check authentication logs
kubectl logs -n spire deployment/spire-server | grep "authentication"

# Verify mTLS configuration
kubectl get configmap -n spire spire-server-config -o yaml

# Check certificate validity
openssl verify -CAfile certs/dev/ca.crt certs/dev/server.crt
```

Common Solutions:
- Verify mTLS configuration
- Check certificate validity
- Ensure proper authentication headers
- Monitor authentication logs

### 2. Authorization Issues
```bash
# Check authorization logs
kubectl logs -n spire deployment/spire-server | grep "authorization"

# Verify policy configuration
kubectl get configmap -n spire spire-server-config -o yaml

# Check workload permissions
kubectl get serviceaccount -n demo
```

Common Solutions:
- Verify policy configuration
- Check workload permissions
- Ensure proper service accounts
- Monitor authorization logs

### 3. Security Policy Violations
```bash
# Check security logs
kubectl logs -n spire deployment/spire-server | grep "security"

# Verify security policies
kubectl get configmap -n spire security-policies -o yaml

# Check audit logs
kubectl logs -n spire deployment/spire-server | grep "audit"
```

Common Solutions:
- Verify security policies
- Check audit logs
- Ensure proper configurations
- Monitor security events

## Performance Issues

### 1. High Latency
```bash
# Check server metrics
kubectl top pod -n spire deployment/spire-server

# View performance logs
kubectl logs -n spire deployment/spire-server | grep "performance"

# Check resource usage
kubectl describe pod -n spire deployment/spire-server
```

Common Solutions:
- Monitor resource usage
- Check network latency
- Verify configuration
- Optimize performance

### 2. Resource Exhaustion
```bash
# Check resource limits
kubectl describe pod -n spire deployment/spire-server

# View resource metrics
kubectl top pod -n spire deployment/spire-server

# Check node resources
kubectl describe node
```

Common Solutions:
- Adjust resource limits
- Monitor resource usage
- Scale resources
- Optimize configuration

### 3. Connection Issues
```bash
# Check network connectivity
kubectl exec -n spire deployment/spire-server -- ping spire-agent

# View connection logs
kubectl logs -n spire deployment/spire-server | grep "connection"

# Verify network policies
kubectl get networkpolicy -n spire
```

Common Solutions:
- Verify network policies
- Check connectivity
- Monitor connections
- Optimize networking

## Common Commands

### 1. System Status
```bash
# Check system status
kubectl get pods -n spire
kubectl get services -n spire
kubectl get configmaps -n spire

# View system logs
kubectl logs -n spire deployment/spire-server
kubectl logs -n spire daemonset/spire-agent

# Check system metrics
kubectl top pod -n spire
```

### 2. Certificate Management
```bash
# Generate certificates
./scripts/generate-dev-certs.sh

# Verify certificates
openssl verify -CAfile certs/dev/ca.crt certs/dev/server.crt
openssl x509 -in certs/dev/server.crt -text -noout

# Check certificate status
kubectl get secret -n spire spire-server-cert -o yaml
kubectl get secret -n spire spire-agent-cert -o yaml
```

### 3. Security Verification
```bash
# Run security checks
./scripts/verify-security.sh

# Check security policies
kubectl get configmap -n spire security-policies -o yaml

# View security logs
kubectl logs -n spire deployment/spire-server | grep "security"
```

## Conclusion

This troubleshooting guide provides solutions for common issues in the workload identity system. For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Developer Guide](developer_guide.md)
- [Deployment Guide](deployment_guide.md) 