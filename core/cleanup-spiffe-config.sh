#!/bin/bash

set -euo pipefail

echo "Deleting frontend deployment..."
kubectl delete -f frontend-deployment.yaml --ignore-not-found

echo "Deleting backend deployment..."
kubectl delete -f backend-deployment.yaml --ignore-not-found

echo "Deleting SPIRE ConfigMap..."
kubectl delete -f spiffe-sidecar-configmap.yaml --ignore-not-found

echo "Deleting ClusterSPIFFEID for backend..."
kubectl delete -f backend-clusterspiffeid.yaml --ignore-not-found

echo "Deleting RegistrationEntry for frontend..."
kubectl delete -f frontend-registration-entry.yaml --ignore-not-found

echo "Cleanup complete."
