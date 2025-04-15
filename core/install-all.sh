#!/bin/bash
# install-all.sh
# This script sets up the SPIFFE mTLS demo environment, including SPIRE registration,
# Kubernetes deployments, and services for frontend and backend workloads.

set -e

echo "=== Starting SPIFFE mTLS Environment Setup ==="

echo "--- Applying SPIFFE ConfigMap ---"
kubectl apply -f spiffe/spiffe-sidecar-configmap.yaml

echo "--- Applying Frontend Deployment ---"
kubectl apply -f workloads/frontend-deployment.yaml

echo "--- Applying Backend Deployment ---"
kubectl apply -f workloads/backend-deployment.yaml

echo "--- Applying SPIFFE ClusterSPIFFEID ---"
kubectl apply -f spiffe/backend-clusterspiffeid.yaml

echo "--- Applying SPIFFE Registration Entry ---"
kubectl apply -f spiffe/frontend-registration-entry.yaml

echo "--- Applying Backend Service ---"
kubectl apply -f workloads/backend-service.yaml

echo "--- Applying Frontend Service ---"
kubectl apply -f workloads/frontend-service.yaml

echo "=== Setup Complete ==="
echo "You can now test mTLS connectivity using the test-mtls.sh script."
