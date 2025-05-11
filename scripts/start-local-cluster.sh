#!/bin/bash

# Local Kubernetes Cluster Setup Script
# This script sets up a local Kubernetes cluster for development

set -e

# Function to check if a command exists
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is required but not installed."
        exit 1
    fi
}

# Check required commands
check_command "docker"
check_command "kubectl"
check_command "kind"

echo "Setting up local Kubernetes cluster..."

# Create kind configuration
cat > kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: workload-identity-dev
nodes:
- role: control-plane
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
- role: worker
- role: worker
EOF

# Create cluster
echo "Creating Kubernetes cluster..."
kind create cluster --config kind-config.yaml

# Wait for cluster to be ready
echo "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# Create namespaces
echo "Creating namespaces..."
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -
kubectl create namespace demo --dry-run=client -o yaml | kubectl apply -f -

# Apply SPIRE server configuration
echo "Applying SPIRE server configuration..."
kubectl apply -f infrastructure/kubernetes/spire/config/server-configmap.yaml
kubectl apply -f infrastructure/kubernetes/spire/server/server-deployment.yaml
kubectl apply -f infrastructure/kubernetes/spire/server/server-service.yaml

# Apply SPIRE agent configuration
echo "Applying SPIRE agent configuration..."
kubectl apply -f infrastructure/kubernetes/spire/config/agent-configmap.yaml
kubectl apply -f infrastructure/kubernetes/spire/agent/agent-daemonset.yaml

# Wait for SPIRE components to be ready
echo "Waiting for SPIRE components to be ready..."
kubectl wait --for=condition=Ready pod -l app=spire-server -n spire --timeout=300s
kubectl wait --for=condition=Ready pod -l app=spire-agent -n spire --timeout=300s

# Apply demo workloads
echo "Applying demo workloads..."
kubectl apply -f infrastructure/kubernetes/workloads/frontend/frontend.yaml
kubectl apply -f infrastructure/kubernetes/workloads/backend/backend.yaml

# Wait for demo workloads to be ready
echo "Waiting for demo workloads to be ready..."
kubectl wait --for=condition=Ready pod -l app=frontend -n demo --timeout=300s
kubectl wait --for=condition=Ready pod -l app=backend -n demo --timeout=300s

# Verify cluster setup
echo "Verifying cluster setup..."
kubectl cluster-info
kubectl get nodes
kubectl get pods -A

echo "Local Kubernetes cluster setup complete!"
echo "You can now access the cluster using kubectl."
echo "Run 'kubectl get pods -A' to see all pods." 