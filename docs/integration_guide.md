# Integration Guide

This guide provides instructions for integrating the workload identity system with various platforms and services.

## Table of Contents
1. [Kubernetes Integration](#kubernetes-integration)
2. [Service Mesh Integration](#service-mesh-integration)
3. [Cloud Provider Integration](#cloud-provider-integration)
4. [CI/CD Integration](#cicd-integration)
5. [Monitoring Integration](#monitoring-integration)

## Kubernetes Integration

### 1. SPIRE Server Setup
```yaml
# SPIRE Server Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-server-config
  namespace: spire
data:
  server.conf: |
    server {
      bind_address = "0.0.0.0"
      bind_port = "8081"
      trust_domain = "example.org"
      data_dir = "/run/spire/data"
      log_level = "INFO"
      ca_key_type = "rsa-2048"
      ca_ttl = "168h"
    }
    plugins {
      DataStore "sql" {
        plugin_data {
          database_type = "sqlite3"
          connection_string = "/run/spire/data/datastore.sqlite3"
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

### 2. SPIRE Agent Setup
```yaml
# SPIRE Agent Configuration
apiVersion: v1
kind: ConfigMap
metadata:
  name: spire-agent-config
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
    }
    plugins {
      NodeAttestor "k8s_psat" {
        plugin_data {
          cluster = "demo-cluster"
        }
      }
      WorkloadAttestor "k8s" {
        plugin_data {
          kubelet_read_only_port = "10255"
        }
      }
    }
```

### 3. Workload Registration
```yaml
# Workload Registration
apiVersion: v1
kind: ServiceAccount
metadata:
  name: frontend
  namespace: demo
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend
  namespace: demo
spec:
  template:
    metadata:
      annotations:
        spire-workload: "true"
    spec:
      serviceAccountName: frontend
      containers:
      - name: frontend
        image: frontend:latest
        volumeMounts:
        - name: spire-agent-socket
          mountPath: /run/spire/sockets
      volumes:
      - name: spire-agent-socket
        hostPath:
          path: /run/spire/sockets
```

## Service Mesh Integration

### 1. Istio Integration
```yaml
# Istio Configuration
apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: istio-system
spec:
  profile: default
  components:
    pilot:
      k8s:
        overlays:
        - apiVersion: apps/v1
          kind: Deployment
          name: istiod
          patches:
          - path: spec.template.spec.volumes
            value:
            - name: spire-agent-socket
              hostPath:
                path: /run/spire/sockets
          - path: spec.template.spec.containers
            value:
            - name: discovery
              volumeMounts:
              - name: spire-agent-socket
                mountPath: /run/spire/sockets
```

### 2. mTLS Configuration
```yaml
# mTLS Configuration
apiVersion: security.istio.io/v1beta1
kind: PeerAuthentication
metadata:
  name: default
  namespace: demo
spec:
  mtls:
    mode: STRICT
---
apiVersion: security.istio.io/v1beta1
kind: AuthorizationPolicy
metadata:
  name: frontend-policy
  namespace: demo
spec:
  selector:
    matchLabels:
      app: frontend
  rules:
  - from:
    - source:
        principals: ["spiffe://example.org/ns/demo/sa/frontend"]
    to:
    - operation:
        methods: ["GET"]
        paths: ["/api/*"]
```

## Cloud Provider Integration

### 1. AWS Integration
```yaml
# AWS Configuration
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aws-workload
  namespace: demo
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/workload-role"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: aws-workload
  namespace: demo
spec:
  template:
    spec:
      serviceAccountName: aws-workload
      containers:
      - name: aws-workload
        image: aws-workload:latest
        env:
        - name: AWS_ROLE_ARN
          value: "arn:aws:iam::123456789012:role/workload-role"
        - name: AWS_WEB_IDENTITY_TOKEN_FILE
          value: "/var/run/secrets/eks.amazonaws.com/serviceaccount/token"
```

### 2. Azure Integration
```yaml
# Azure Configuration
apiVersion: v1
kind: ServiceAccount
metadata:
  name: azure-workload
  namespace: demo
  annotations:
    azure.workload.identity/client-id: "client-id"
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: azure-workload
  namespace: demo
spec:
  template:
    spec:
      serviceAccountName: azure-workload
      containers:
      - name: azure-workload
        image: azure-workload:latest
        env:
        - name: AZURE_CLIENT_ID
          value: "client-id"
        - name: AZURE_TENANT_ID
          value: "tenant-id"
```

## CI/CD Integration

### 1. GitHub Actions Integration
```yaml
# GitHub Actions Workflow
name: Deploy Workload
on:
  push:
    branches: [ main ]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v2
    - name: Configure AWS Credentials
      uses: aws-actions/configure-aws-credentials@v1
      with:
        role-to-assume: arn:aws:iam::123456789012:role/github-actions
        aws-region: us-west-2
    - name: Deploy to EKS
      run: |
        aws eks update-kubeconfig --name demo-cluster
        kubectl apply -f k8s/
```

### 2. GitLab CI Integration
```yaml
# GitLab CI Configuration
stages:
  - deploy
deploy:
  stage: deploy
  image: bitnami/kubectl:latest
  script:
    - kubectl config use-context demo-cluster
    - kubectl apply -f k8s/
  only:
    - main
```

## Monitoring Integration

### 1. Prometheus Integration
```yaml
# Prometheus Configuration
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: spire-monitor
  namespace: monitoring
spec:
  selector:
    matchLabels:
      app: spire-server
  endpoints:
  - port: metrics
    interval: 15s
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: prometheus-rules
  namespace: monitoring
data:
  spire-rules.yaml: |
    groups:
    - name: spire
      rules:
      - alert: SpireServerDown
        expr: up{job="spire-server"} == 0
        for: 5m
        labels:
          severity: critical
```

### 2. Grafana Integration
```yaml
# Grafana Dashboard
apiVersion: v1
kind: ConfigMap
metadata:
  name: grafana-dashboards
  namespace: monitoring
data:
  spire-dashboard.json: |
    {
      "dashboard": {
        "id": null,
        "title": "SPIRE Metrics",
        "panels": [
          {
            "title": "Server Health",
            "type": "graph",
            "datasource": "Prometheus",
            "targets": [
              {
                "expr": "up{job=\"spire-server\"}",
                "legendFormat": "{{pod}}"
              }
            ]
          }
        ]
      }
    }
```

## Conclusion

This integration guide provides instructions for integrating the workload identity system with various platforms and services. For additional information, refer to:
- [Architecture Guide](architecture_guide.md)
- [Security Best Practices](security_best_practices.md)
- [Developer Guide](developer_guide.md)
- [Deployment Guide](deployment_guide.md) 