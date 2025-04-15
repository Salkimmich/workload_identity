#!/bin/bash
# uninstall-all.sh
# This script removes all resources deployed by install-all.sh, including SPIFFE identity configuration, workloads, and services.
# Use this when you want to clean up your Kubernetes environment, reset the system, or prepare for a fresh install.
# WARNING: This will delete all deployed SPIFFE and workload components.

set -e

echo "=== Uninstalling all SPIFFE and workload resources ==="

# Delete workloads
kubectl delete -f workloads/frontend-deployment.yaml --ignore-not-found
kubectl delete -f workloads/backend-deployment.yaml --ignore-not-found
kubectl delete -f workloads/frontend-service.yaml --ignore-not-found
kubectl delete -f workloads/backend-service.yaml --ignore-not-found

# Delete SPIFFE identity config
kubectl delete -f spiffe/frontend-registration-entry.yaml --ignore-not-found
kubectl delete -f spiffe/backend-clusterspiffeid.yaml --ignore-not-found
kubectl delete -f spiffe/spiffe-sidecar-configmap.yaml --ignore-not-found

# Optional: cleanup mounted config
kubectl delete configmap spiffe-sidecar-config --ignore-not-found

echo "=== Uninstallation complete. Kubernetes environment cleaned up. ==="
