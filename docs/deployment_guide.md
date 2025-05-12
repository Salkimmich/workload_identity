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

### Overview
The workload identity system supports integration with AWS, Azure, and GCP. The high-level workflow is the same (workload receives a short-lived identity token, SPIRE agent verifies it, etc.), but the setup and required configuration differ between providers.

#### Key Differences
| Provider | Service Account Annotation | IAM/Role Mapping | Token Path | Additional Setup |
|----------|---------------------------|------------------|------------|-----------------|
| GCP      | `iam.gke.io/gcp-service-account` | GCP Service Account | `/var/run/secrets/tokens/` | Workload Identity Pool, Federation |
| AWS      | `eks.amazonaws.com/role-arn`     | IAM Role for Service Account (IRSA) | `/var/run/secrets/eks.amazonaws.com/serviceaccount/token` | OIDC Provider, IAM Role |
| Azure    | `azure.workload.identity/client-id` | Azure AD Application Client ID | `/var/run/secrets/azure/tokens/azure-identity-token` | Azure AD Workload Identity, Federated Credentials |

> **Note:** The SPIRE-based identity flow is conceptually similar across providers, but the details of token issuance, required annotations, and trust establishment are cloud-specific.

### 1. GCP Integration
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: gcp-workload
  namespace: demo
  annotations:
    iam.gke.io/gcp-service-account: "workload-identity@project.iam.gserviceaccount.com" # Required for GCP
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gcp-workload
  namespace: demo
spec:
  template:
    spec:
      serviceAccountName: gcp-workload
      containers:
      - name: gcp-workload
        image: gcp-workload:latest
        env:
        - name: GOOGLE_APPLICATION_CREDENTIALS
          value: "/var/run/secrets/tokens/GOOGLE-APPLICATION-CREDENTIALS"
```
- **Required:** `iam.gke.io/gcp-service-account` annotation
- **Setup:** Configure GCP Workload Identity Pool and map KSA to GSA

### 2. AWS Integration
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: aws-workload
  namespace: demo
  annotations:
    eks.amazonaws.com/role-arn: "arn:aws:iam::123456789012:role/workload-role" # Required for AWS
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
- **Required:** `eks.amazonaws.com/role-arn` annotation
- **Setup:** Create OIDC provider in AWS, map KSA to IAM Role

### 3. Azure Integration
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: azure-workload
  namespace: demo
  annotations:
    azure.workload.identity/client-id: "<azure-ad-app-client-id>" # Required for Azure
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
          value: "<azure-ad-app-client-id>"
        - name: AZURE_TENANT_ID
          value: "<azure-ad-tenant-id>"
```
- **Required:** `azure.workload.identity/client-id` annotation
- **Setup:** Register Azure AD Application, configure federated credentials

### Implementation Notes
- **Required fields** are marked in the YAML and comments above.
- **Optional fields** can be omitted or customized as needed.
- Always refer to your cloud provider's documentation for the latest requirements and best practices.

## Security Configuration

### 1. TLS Configuration
```