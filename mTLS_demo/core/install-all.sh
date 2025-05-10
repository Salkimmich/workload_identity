#!/bin/bash

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to print status
print_status() {
    if [ $1 -eq 0 ]; then
        echo -e "${GREEN}✓ $2${NC}"
    else
        echo -e "${RED}✗ $2${NC}"
        exit 1
    fi
}

# Check if kubectl is available
if ! command -v kubectl &> /dev/null; then
    echo -e "${RED}Error: kubectl is not installed${NC}"
    exit 1
fi

# Create namespace if it doesn't exist
echo "Creating namespace..."
kubectl create namespace demo --dry-run=client -o yaml | kubectl apply -f -
print_status $? "Namespace created"

# Apply SPIFFE Helper configuration
echo "Applying SPIFFE Helper configuration..."
kubectl create configmap spiffe-helper-config \
    --from-file=helper.conf=../config/sidecar-config.json \
    -n demo --dry-run=client -o yaml | kubectl apply -f -
print_status $? "SPIFFE Helper configuration applied"

# Apply SPIFFE registration entries
echo "Applying SPIFFE registration entries..."
kubectl apply -f ../spiffe/frontend-registration-entry.yaml
print_status $? "Frontend registration entry applied"

kubectl apply -f ../spiffe/backend-clusterspiffeid.yaml
print_status $? "Backend registration entry applied"

# Apply Kubernetes deployments and services
echo "Applying Kubernetes resources..."
kubectl apply -f ../workloads/frontend-deployment.yaml
print_status $? "Frontend deployment applied"

kubectl apply -f ../workloads/backend-deployment.yaml
print_status $? "Backend deployment applied"

# Wait for pods to be ready
echo "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod -l app=frontend -n demo --timeout=60s
print_status $? "Frontend pod is ready"

kubectl wait --for=condition=ready pod -l app=backend -n demo --timeout=60s
print_status $? "Backend pod is ready"

echo -e "\n${GREEN}Installation completed successfully!${NC}"
echo "You can now run the test script to verify the setup:"
echo "cd ../test && ./test-mtls.sh" 