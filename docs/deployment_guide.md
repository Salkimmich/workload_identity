# Deployment Guide

This guide provides detailed instructions for deploying the workload identity system in various environments.

## Table of Contents
1. [Prerequisites](#prerequisites)
2. [Local Development](#local-development)
3. [Kubernetes Deployment](#kubernetes-deployment)
4. [Cloud Provider Integration](#cloud-provider-integration)
5. [Security Configuration](#security-configuration)
6. [Monitoring Setup](#monitoring-setup)

## Prerequisites

### Required Tools
```bash
# Check and install required tools
make check-prerequisites

# Required versions
go >= 1.20
docker >= 20.10
kubectl >= 1.24
kind >= 0.20
openssl >= 3.0
```

### Required Services
- Kubernetes cluster (v1.24+)
- PostgreSQL (v14+)
- SPIRE server and agent

## Local Development

### 1. Setup Development Environment
```bash
# Clone repository
git clone https://github.com/your-org/workload-identity.git
cd workload-identity

# Setup development environment
make setup-dev

# Start local cluster
make start-local-cluster
```

### 2. Generate Development Certificates
```bash
# Generate certificates
make generate-dev-certs

# Verify certificates
make verify-certs
```

## Kubernetes Deployment

### 1. SPIRE Server Configuration
```yaml
# spire-server-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-server
  namespace: spire
data:
  server.conf: |
    server {
      bind_address = "0.0.0.0"
      bind_port = "8081"
      trust_domain = "example.org"
      data_dir = "/run/spire/data"
      log_level = "INFO"
      
      ca_subject {
        country = ["US"]
        organization = ["SPIFFE"]
        common_name = ""
      }
    }

    plugins {
      DataStore "sql" {
        plugin_data {
          database_type = "postgres"
          connection_string = "host=postgres port=5432 user=spire password=spire dbname=spire sslmode=disable"
        }
      }
      
      NodeAttestor "k8s_psat" {
        plugin_data {
          clusters = {
            "demo-cluster" = {
              service_account_allow_list = ["spire:spire-agent"]
            }
          }
        }
      }
    }
```

### 2. SPIRE Agent Configuration
```yaml
# spire-agent-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-agent
  namespace: spire
data:
  agent.conf: |
    agent {
      data_dir = "/run/spire/data"
      log_level = "INFO"
      server_address = "spire-server"
      server_port = "8081"
      socket_path = "/run/spire/sockets/agent.sock"
      trust_bundle_path = "/run/spire/bundle/bundle.crt"
      trust_domain = "example.org"
    }

    plugins {
      NodeAttestor "k8s_psat" {
        plugin_data {
          cluster = "demo-cluster"
        }
      }
      
      WorkloadAttestor "k8s" {
        plugin_data {
          kubelet_read_only_port = 10255
        }
      }
    }
```

### 3. Deploy SPIRE Components
```bash
# Create namespace
kubectl create namespace spire

# Apply configurations
kubectl apply -f spire-server-config.yaml
kubectl apply -f spire-agent-config.yaml

# Deploy SPIRE server
kubectl apply -f infrastructure/kubernetes/spire/server.yaml

# Deploy SPIRE agent
kubectl apply -f infrastructure/kubernetes/spire/agent.yaml
```

## Cloud Provider Integration

### 1. AWS Integration
```yaml
# aws-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: aws-config
  namespace: spire
data:
  aws.conf: |
    aws {
      region = "us-west-2"
      role_arn = "arn:aws:iam::123456789012:role/spire-role"
      session_duration = 3600
    }
```

### 2. Azure Integration
```yaml
# azure-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: azure-config
  namespace: spire
data:
  azure.conf: |
    azure {
      tenant_id = "your-tenant-id"
      subscription_id = "your-subscription-id"
      resource_group = "your-resource-group"
    }
```

## Security Configuration

### 1. TLS Configuration
```yaml
# tls-config.yaml
apiVersion: v1
kind: Secret
metadata:
  name: spire-tls
  namespace: spire
type: kubernetes.io/tls
data:
  tls.crt: <base64-encoded-cert>
  tls.key: <base64-encoded-key>
  ca.crt: <base64-encoded-ca>
```

### 2. Network Policies
```yaml
# network-policies.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: spire-server
  namespace: spire
spec:
  podSelector:
    matchLabels:
      app: spire-server
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: spire
    ports:
    - protocol: TCP
      port: 8081
```

## Monitoring Setup

### 1. Prometheus Configuration
```yaml
# prometheus-config.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-config
  namespace: monitoring
data:
  prometheus.yml: |
    global:
      scrape_interval: 15s
    scrape_configs:
      - job_name: 'spire'
        static_configs:
          - targets: ['spire-server:8082']
```

### 2. Grafana Dashboards
```yaml
# grafana-dashboard.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-dashboard
  namespace: monitoring
data:
  dashboard.json: |
    {
      "dashboard": {
        "title": "SPIRE Metrics",
        "panels": [
          {
            "title": "Certificate Operations",
            "type": "graph",
            "datasource": "Prometheus",
            "targets": [
              {
                "expr": "rate(spire_certificate_operations_total[5m])"
              }
            ]
          }
        ]
      }
    }
```

## Verification

### 1. Check Deployment
```bash
# Verify SPIRE components
kubectl get pods -n spire

# Check SPIRE server logs
kubectl logs -n spire -l app=spire-server

# Verify agent registration
kubectl exec -n spire -it spire-server-0 -- spire-server agent list
```

### 2. Test Workload Identity
```bash
# Deploy test workload
kubectl apply -f examples/workloads/test-workload.yaml

# Verify identity
kubectl exec -n demo -it test-workload -- spire-agent api fetch x509
```

## Troubleshooting

### Common Issues
1. Certificate Issues
   ```bash
   # Regenerate certificates
   make regenerate-certs
   
   # Verify certificate chain
   make verify-certs
   ```

2. Kubernetes Issues
   ```bash
   # Check cluster status
   make check-cluster
   
   # Reset development cluster
   make reset-cluster
   ```

3. Security Issues
   ```bash
   # Run security diagnostics
   make security-diagnostics
   
   # Verify security configuration
   make verify-security
   ```

## Conclusion

This guide provides comprehensive deployment instructions for the workload identity system. For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Developer Guide](developer_guide.md)
- [API Reference](api_reference.md) 