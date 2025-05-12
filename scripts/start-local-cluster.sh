#!/bin/bash

# Local Kubernetes Cluster Setup Script
# This script sets up a local Kubernetes cluster for development
# Required: This script must be run from the project root directory

set -e  # Required: Exit on any error

# Function to check if a command exists
# Required: Ensures all necessary tools are installed
check_command() {
    if ! command -v "$1" &> /dev/null; then
        echo "Error: $1 is required but not installed."
        exit 1
    fi
}

# Required: Check for essential tools
check_command "docker"    # Required: Container runtime
check_command "kubectl"   # Required: Kubernetes CLI
check_command "kind"      # Required: Local cluster tool

echo "Setting up local Kubernetes cluster..."

# Required: Create kind configuration
# This defines the structure of your local cluster
cat > kind-config.yaml << EOF
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
name: workload-identity-dev  # Required: Unique cluster name
nodes:
- role: control-plane  # Required: Control plane node
  extraPortMappings:   # Required: Port mappings for ingress
  - containerPort: 80  # Required: HTTP port
    hostPort: 80
    protocol: TCP
  - containerPort: 443 # Required: HTTPS port
    hostPort: 443
    protocol: TCP
  kubeadmConfigPatches:  # Required: Control plane configuration
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"  # Required: For ingress controller
- role: worker  # Required: Worker node
- role: worker  # Optional: Additional worker node
EOF

# Required: Create cluster
echo "Creating Kubernetes cluster..."
kind create cluster --config kind-config.yaml

# Required: Wait for cluster to be ready
echo "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=300s

# Required: Create namespaces
echo "Creating namespaces..."
kubectl create namespace spire --dry-run=client -o yaml | kubectl apply -f -  # Required: SPIRE namespace
kubectl create namespace demo --dry-run=client -o yaml | kubectl apply -f -   # Required: Demo namespace

# Required: Apply SPIRE server configuration
echo "Applying SPIRE server configuration..."
kubectl apply -f infrastructure/kubernetes/spire/config/server-configmap.yaml  # Required: Server config
kubectl apply -f infrastructure/kubernetes/spire/server/server-deployment.yaml  # Required: Server deployment
kubectl apply -f infrastructure/kubernetes/spire/server/server-service.yaml    # Required: Server service

# Required: Apply SPIRE agent configuration
echo "Applying SPIRE agent configuration..."
kubectl apply -f infrastructure/kubernetes/spire/config/agent-configmap.yaml  # Required: Agent config
kubectl apply -f infrastructure/kubernetes/spire/agent/agent-daemonset.yaml   # Required: Agent deployment

# Required: Wait for SPIRE components to be ready
echo "Waiting for SPIRE components to be ready..."
kubectl wait --for=condition=Ready pod -l app=spire-server -n spire --timeout=300s  # Required: Server readiness
kubectl wait --for=condition=Ready pod -l app=spire-agent -n spire --timeout=300s   # Required: Agent readiness

# Optional: Apply demo workloads
echo "Applying demo workloads..."
kubectl apply -f infrastructure/kubernetes/workloads/frontend/frontend.yaml  # Optional: Frontend demo
kubectl apply -f infrastructure/kubernetes/workloads/backend/backend.yaml    # Optional: Backend demo

# Optional: Wait for demo workloads to be ready
echo "Waiting for demo workloads to be ready..."
kubectl wait --for=condition=Ready pod -l app=frontend -n demo --timeout=300s  # Optional: Frontend readiness
kubectl wait --for=condition=Ready pod -l app=backend -n demo --timeout=300s   # Optional: Backend readiness

# Required: Verify cluster setup
echo "Verifying cluster setup..."
kubectl cluster-info  # Required: Check cluster status
kubectl get nodes     # Required: Check node status
kubectl get pods -A   # Required: Check pod status

echo "Local Kubernetes cluster setup complete!"
echo "You can now access the cluster using kubectl."
echo "Run 'kubectl get pods -A' to see all pods."

# Optional: Additional setup steps that might be needed:
# - Configure ingress controller
# - Set up monitoring stack
# - Configure logging
# - Set up backup solution
# - Configure network policies
# - Set up service mesh
# - Configure storage classes
# - Set up resource quotas 