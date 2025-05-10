#!/bin/bash

# Enable strict error handling:
# -e: Exit immediately if a command exits with a non-zero status
# -u: Treat unset variables as an error when substituting
# -o pipefail: Return value of a pipeline is the status of the last command to exit with a non-zero status
set -euo pipefail

# Error handling function that will be called when any command fails
# This ensures we clean up any partially applied policies
handle_error() {
    # Print the line number where the error occurred
    echo "Error occurred in apply-security-policies.sh at line $1"
    echo "Rolling back changes..."
    
    # Delete all applied policies in reverse order
    # --ignore-not-found prevents errors if the resources don't exist
    kubectl delete -f security/network-policies/network-policies.yaml --ignore-not-found
    kubectl delete -f security/pss/pss.yaml --ignore-not-found
    if kubectl api-resources | grep -q "securitycontextconstraints"; then
        kubectl delete -f security/scc/scc.yaml --ignore-not-found
    fi
    exit 1
}

# Set up error handling trap
# This will call handle_error with the line number when any command fails
trap 'handle_error $LINENO' ERR

# Function to verify if a policy exists in the cluster
# Parameters:
#   $1: policy_type - The type of policy (e.g., psp, networkpolicy, scc)
#   $2: namespace - The namespace where the policy should exist
#   $3: policy_name - The name of the policy to verify
verify_policy_exists() {
    local policy_type=$1
    local namespace=$2
    local policy_name=$3
    
    # Check if the policy exists using kubectl get
    # Redirect both stdout and stderr to /dev/null to suppress output
    if ! kubectl get $policy_type -n $namespace $policy_name &>/dev/null; then
        echo "Error: Failed to verify $policy_type $policy_name in namespace $namespace"
        return 1
    fi
    return 0
}

# Function to verify Pod Security Policy (PSP) configuration
# Parameters:
#   $1: psp_name - The name of the PSP to verify
#   $2: expected_privileged - The expected value for the privileged setting
verify_psp_config() {
    local psp_name=$1
    local expected_privileged=$2
    
    echo "Verifying PSP $psp_name configuration..."
    
    # Check if the privileged setting matches the expected value
    # Uses jsonpath to extract the specific field from the PSP
    local actual_privileged=$(kubectl get psp $psp_name -o jsonpath='{.spec.privileged}')
    if [[ "$actual_privileged" != "$expected_privileged" ]]; then
        echo "Error: PSP $psp_name privileged setting mismatch. Expected: $expected_privileged, Got: $actual_privileged"
        return 1
    fi
    
    # Verify that the PSP has the required security labels
    # These labels are used by the Pod Security Admission controller
    local enforce_label=$(kubectl get psp $psp_name -o jsonpath='{.metadata.labels.pod-security\.kubernetes\.io/enforce}')
    if [[ -z "$enforce_label" ]]; then
        echo "Error: PSP $psp_name missing enforce label"
        return 1
    fi
    
    echo "PSP $psp_name configuration verified successfully"
    return 0
}

# Function to verify NetworkPolicy configuration
# Parameters:
#   $1: namespace - The namespace containing the policy
#   $2: policy_name - The name of the policy to verify
#   $3: expected_ingress_count - Expected number of ingress rules
#   $4: expected_egress_count - Expected number of egress rules
verify_network_policy() {
    local namespace=$1
    local policy_name=$2
    local expected_ingress_count=$3
    local expected_egress_count=$4
    
    echo "Verifying NetworkPolicy $policy_name in namespace $namespace..."
    
    # First verify that the policy exists
    if ! verify_policy_exists "networkpolicy" "$namespace" "$policy_name"; then
        return 1
    fi
    
    # Count the number of ingress rules using jq to parse the JSON output
    local actual_ingress_count=$(kubectl get networkpolicy -n $namespace $policy_name -o jsonpath='{.spec.ingress}' | jq '. | length')
    if [[ "$actual_ingress_count" != "$expected_ingress_count" ]]; then
        echo "Error: NetworkPolicy $policy_name ingress rules count mismatch. Expected: $expected_ingress_count, Got: $actual_ingress_count"
        return 1
    fi
    
    # Count the number of egress rules
    local actual_egress_count=$(kubectl get networkpolicy -n $namespace $policy_name -o jsonpath='{.spec.egress}' | jq '. | length')
    if [[ "$actual_egress_count" != "$expected_egress_count" ]]; then
        echo "Error: NetworkPolicy $policy_name egress rules count mismatch. Expected: $expected_egress_count, Got: $actual_egress_count"
        return 1
    fi
    
    # Verify that both Ingress and Egress policy types are present
    local policy_types=$(kubectl get networkpolicy -n $namespace $policy_name -o jsonpath='{.spec.policyTypes}')
    if [[ ! "$policy_types" =~ "Ingress" ]] || [[ ! "$policy_types" =~ "Egress" ]]; then
        echo "Error: NetworkPolicy $policy_name missing required policy types"
        return 1
    fi
    
    echo "NetworkPolicy $policy_name configuration verified successfully"
    return 0
}

# Function to verify Security Context Constraints (SCC) configuration
# Parameters:
#   $1: scc_name - The name of the SCC to verify
#   $2: expected_privileged - The expected value for the privileged setting
verify_scc_config() {
    local scc_name=$1
    local expected_privileged=$2
    
    echo "Verifying SCC $scc_name configuration..."
    
    # Check if the privileged container setting matches the expected value
    local actual_privileged=$(kubectl get scc $scc_name -o jsonpath='{.allowPrivilegedContainer}')
    if [[ "$actual_privileged" != "$expected_privileged" ]]; then
        echo "Error: SCC $scc_name privileged setting mismatch. Expected: $expected_privileged, Got: $actual_privileged"
        return 1
    fi
    
    # Verify that required capabilities are properly configured
    local required_capabilities=$(kubectl get scc $scc_name -o jsonpath='{.requiredDropCapabilities}')
    if [[ -z "$required_capabilities" ]]; then
        echo "Error: SCC $scc_name missing required drop capabilities"
        return 1
    fi
    
    echo "SCC $scc_name configuration verified successfully"
    return 0
}

# Main execution starts here

# Create the demo namespace if it doesn't exist
# --dry-run=client ensures we don't actually create the namespace if it exists
echo "Creating demo namespace..."
kubectl create namespace demo --dry-run=client -o yaml | kubectl apply -f -

# Apply and verify Pod Security Standards
echo "Applying Pod Security Standards..."
kubectl apply -f security/pss/pss.yaml
# Verify baseline PSP (non-privileged)
verify_psp_config "baseline-pss" "false"
# Verify privileged PSP
verify_psp_config "privileged-pss" "true"

# Apply and verify Network Policies
echo "Applying Network Policies..."
kubectl apply -f security/network-policies/network-policies.yaml
# Verify frontend policy (2 ingress rules: ingress-nginx, health check; 3 egress rules)
verify_network_policy "demo" "frontend-network-policy" 2 3
# Verify backend policy (2 ingress rules: frontend, health check; 3 egress rules)
verify_network_policy "demo" "backend-network-policy" 2 3
# Verify API policy (2 ingress rules: backend, health check; 2 egress rules)
verify_network_policy "demo" "api-network-policy" 2 2

# Apply and verify Security Context Constraints (only if running on OpenShift)
if kubectl api-resources | grep -q "securitycontextconstraints"; then
    echo "Applying Security Context Constraints..."
    kubectl apply -f security/scc/scc.yaml
    # Verify restricted SCC (non-privileged)
    verify_scc_config "restricted-scc" "false"
    # Verify privileged SCC
    verify_scc_config "privileged-scc" "true"
fi

# Final verification of all policies
echo "Performing final verification of all policies..."
# List all PSPs
kubectl get psp
# List all NetworkPolicies in the demo namespace
kubectl get networkpolicy -n demo
# List all SCCs if on OpenShift
if kubectl api-resources | grep -q "securitycontextconstraints"; then
    kubectl get scc
fi

echo "All security policies have been applied and verified successfully!" 