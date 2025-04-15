#!/bin/bash
# cleanup-services.sh
# This script deletes the frontend and backend Kubernetes Services created for SPIFFE-enabled workloads.

set -e

echo "Deleting backend service..."
kubectl delete -f workloads/backend-service.yaml || echo "Backend service not found."

echo "Deleting frontend service..."
kubectl delete -f workloads/frontend-service.yaml || echo "Frontend service not found."

echo "Services deleted successfully."
