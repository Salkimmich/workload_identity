#!/bin/bash

# Create namespace
kubectl create namespace demo --dry-run=client -o yaml | kubectl apply -f -

# Deploy frontend
echo "Deploying frontend..."
kubectl apply -f frontend/frontend.yaml

# Deploy backend
echo "Deploying backend..."
kubectl apply -f backend/backend.yaml

# Deploy API
echo "Deploying API..."
kubectl apply -f api/api.yaml

# Wait for deployments to be ready
echo "Waiting for deployments to be ready..."
kubectl wait --for=condition=available --timeout=300s deployment/frontend -n demo
kubectl wait --for=condition=available --timeout=300s deployment/backend -n demo
kubectl wait --for=condition=available --timeout=300s deployment/api -n demo

# Verify SPIFFE IDs
echo "Verifying SPIFFE IDs..."
kubectl get clusterspiffeid -n demo

# Check pod status
echo "Checking pod status..."
kubectl get pods -n demo

echo "Deployment complete!" 