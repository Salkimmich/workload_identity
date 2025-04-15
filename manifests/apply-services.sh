#!/bin/bash
# apply-services.sh
# This script applies the frontend and backend Kubernetes Services for SPIFFE-enabled workloads.

set -e

echo "Applying backend service..."
kubectl apply -f workloads/backend-service.yaml

echo "Applying frontend service..."
kubectl apply -f workloads/frontend-service.yaml

echo "Services applied successfully."
