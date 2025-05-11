#!/bin/bash

# Workload Identity Verification Script
# This script demonstrates basic workload identity verification
# Note: This is a simplified example. Production scripts need more error handling and security checks.

# Default token path
TOKEN_PATH=${WORKLOAD_IDENTITY_TOKEN_PATH:-"/var/run/secrets/tokens/workload-identity"}

# Check if token exists
if [ ! -f "$TOKEN_PATH" ]; then
    echo "Error: Workload identity token not found at $TOKEN_PATH"
    exit 1
fi

# Read the token
TOKEN=$(cat "$TOKEN_PATH")

# Verify token format (basic check)
if [[ ! $TOKEN =~ ^[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+\.[A-Za-z0-9-_]+$ ]]; then
    echo "Error: Invalid token format"
    exit 1
fi

# Decode token header (for demonstration)
echo "Token Header:"
echo $TOKEN | cut -d. -f1 | base64 -d 2>/dev/null | jq '.'

# Verify token expiration
EXPIRY=$(echo $TOKEN | cut -d. -f2 | base64 -d 2>/dev/null | jq -r '.exp')
CURRENT_TIME=$(date +%s)

if [ "$EXPIRY" -lt "$CURRENT_TIME" ]; then
    echo "Error: Token has expired"
    exit 1
fi

echo "Token is valid and not expired"
echo "Expires at: $(date -r $EXPIRY)"

# Note: This is a basic script. Production scripts should:
# - Include proper error handling
# - Validate token signature
# - Check token claims
# - Implement proper logging
# - Add security measures 