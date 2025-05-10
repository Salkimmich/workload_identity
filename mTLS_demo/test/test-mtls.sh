#!/bin/bash
# test-mtls.sh
# This script tests the mutual TLS communication between frontend and 
backend workloads
# using the SPIFFE SVIDs issued by SPIRE. 

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

# Check if the frontend pod is running
echo "Checking frontend pod status..."
FRONTEND_POD=$(kubectl get pod -l app=frontend -o jsonpath="{.items[0].metadata.name}")
if [ -z "$FRONTEND_POD" ]; then
    echo -e "${RED}Error: Frontend pod not found${NC}"
    exit 1
fi

# Check if the backend pod is running
echo "Checking backend pod status..."
BACKEND_POD=$(kubectl get pod -l app=backend -o jsonpath="{.items[0].metadata.name}")
if [ -z "$BACKEND_POD" ]; then
    echo -e "${RED}Error: Backend pod not found${NC}"
    exit 1
fi

# Wait for pods to be ready
echo "Waiting for pods to be ready..."
kubectl wait --for=condition=ready pod/$FRONTEND_POD --timeout=60s
print_status $? "Frontend pod is ready"

kubectl wait --for=condition=ready pod/$BACKEND_POD --timeout=60s
print_status $? "Backend pod is ready"

# Check if SPIFFE Helper is running in both pods
echo "Checking SPIFFE Helper status..."
kubectl get pod $FRONTEND_POD -o jsonpath="{.status.containerStatuses[?(@.name=='spiffe-helper')].ready}" | grep -q "true"
print_status $? "SPIFFE Helper is running in frontend pod"

kubectl get pod $BACKEND_POD -o jsonpath="{.status.containerStatuses[?(@.name=='spiffe-helper')].ready}" | grep -q "true"
print_status $? "SPIFFE Helper is running in backend pod"

# Check if SVIDs are present
echo "Checking for SVIDs..."
kubectl exec $FRONTEND_POD -c frontend -- ls /tmp/svid.pem /tmp/key.pem /tmp/bundle.pem &> /dev/null
print_status $? "SVIDs are present in frontend pod"

kubectl exec $BACKEND_POD -c backend -- ls /tmp/svid.pem /tmp/key.pem /tmp/bundle.pem &> /dev/null
print_status $? "SVIDs are present in backend pod"

# Test mTLS communication
echo "Testing mTLS communication..."
FRONTEND_SERVICE=$(kubectl get service frontend -o jsonpath="{.spec.clusterIP}")
RESPONSE=$(kubectl exec $FRONTEND_POD -c frontend -- curl -s http://localhost:8080)
if [[ $RESPONSE == *"Hello from backend"* ]]; then
    print_status 0 "mTLS communication successful"
else
    print_status 1 "mTLS communication failed"
fi

echo -e "\n${GREEN}All tests completed successfully!${NC}"
