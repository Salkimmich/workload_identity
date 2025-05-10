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

# Delete Kubernetes resources
echo "Deleting Kubernetes resources..."
kubectl delete -f ../workloads/frontend-deployment.yaml
print_status $? "Frontend deployment deleted"

kubectl delete -f ../workloads/backend-deployment.yaml
print_status $? "Backend deployment deleted"

# Delete SPIFFE registration entries
echo "Deleting SPIFFE registration entries..."
kubectl delete -f ../spiffe/frontend-registration-entry.yaml
print_status $? "Frontend registration entry deleted"

kubectl delete -f ../spiffe/backend-clusterspiffeid.yaml
print_status $? "Backend registration entry deleted"

# Delete SPIFFE Helper configuration
echo "Deleting SPIFFE Helper configuration..."
kubectl delete configmap spiffe-helper-config -n demo
print_status $? "SPIFFE Helper configuration deleted"

# Delete namespace
echo "Deleting namespace..."
kubectl delete namespace demo
print_status $? "Namespace deleted"

echo -e "\n${GREEN}Uninstallation completed successfully!${NC}" 