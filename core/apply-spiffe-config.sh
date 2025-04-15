#!/bin/bash

set -euo pipefail

echo "Applying SPIRE ConfigMap for sidecar config..."
kubectl apply -f spiffe-sidecar-configmap.yaml

echo "Deploying frontend workload..."
kubectl apply -f frontend-deployment.yaml

echo "Deploying backend workload..."
kubectl apply -f backend-deployment.yaml

echo "Applying ClusterSPIFFEID for backend..."
kubectl apply -f backend-clusterspiffeid.yaml

echo "Applying RegistrationEntry for frontend..."
kubectl apply -f frontend-registration-entry.yaml

echo "All manifests applied successfully."
